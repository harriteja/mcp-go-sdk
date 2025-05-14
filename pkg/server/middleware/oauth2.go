package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

var (
	ErrNoToken          = errors.New("no token provided")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidGrantType = errors.New("invalid grant type")
	ErrInvalidScope     = errors.New("invalid scope")
)

// OAuth2Config holds the configuration for OAuth2 authentication
type OAuth2Config struct {
	// OAuth2 configuration
	Config *oauth2.Config
	// Optional custom token validation function
	ValidateToken func(ctx context.Context, token *oauth2.Token) error
	// Required scopes for the endpoint
	RequiredScopes []string
	// Token cache duration
	TokenCacheDuration time.Duration
}

// tokenCache provides thread-safe caching of validated tokens
type tokenCache struct {
	mu    sync.RWMutex
	cache map[string]tokenCacheEntry
}

type tokenCacheEntry struct {
	token      *oauth2.Token
	validUntil time.Time
}

// OAuth2Middleware creates a new OAuth2 authentication middleware
func OAuth2Middleware(config OAuth2Config) func(http.Handler) http.Handler {
	if config.TokenCacheDuration == 0 {
		config.TokenCacheDuration = 5 * time.Minute // Default cache duration
	}

	cache := &tokenCache{
		cache: make(map[string]tokenCacheEntry),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractToken(r)
			if err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Check cache first
			if validToken := cache.get(token.AccessToken); validToken != nil {
				// Token is valid and cached, proceed
				ctx := context.WithValue(r.Context(), oauth2.HTTPClient, &http.Client{
					Transport: &oauth2.Transport{
						Source: oauth2.StaticTokenSource(validToken),
					},
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Validate token
			if err := validateToken(r.Context(), token, config); err != nil {
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
				return
			}

			// Cache valid token
			cache.set(token.AccessToken, token, config.TokenCacheDuration)

			// Add token to context and proceed
			ctx := context.WithValue(r.Context(), oauth2.HTTPClient, &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(token),
				},
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) (*oauth2.Token, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, ErrNoToken
	}

	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, ErrInvalidToken
	}

	return &oauth2.Token{
		AccessToken: parts[1],
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour), // Default expiry
	}, nil
}

func validateToken(ctx context.Context, token *oauth2.Token, config OAuth2Config) error {
	if token == nil || token.AccessToken == "" {
		return ErrInvalidToken
	}

	// Skip token.Valid() check since we're using custom validation
	// and the token expiry is handled by the cache

	// Call custom validation if provided
	if config.ValidateToken != nil {
		if err := config.ValidateToken(ctx, token); err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}
		return nil // Return immediately if custom validation passes
	}

	// Only proceed with scope validation if no custom validation is provided
	if len(config.RequiredScopes) > 0 {
		tokenScopes, err := getTokenScopes(ctx, token, config)
		if err != nil {
			return fmt.Errorf("scope validation failed: %w", err)
		}

		if !hasRequiredScopes(tokenScopes, config.RequiredScopes) {
			return ErrInvalidScope
		}
	}

	return nil
}

func getTokenScopes(ctx context.Context, token *oauth2.Token, config OAuth2Config) ([]string, error) {
	// First try to get scopes from token extras
	if scope, ok := token.Extra("scope").(string); ok {
		return strings.Split(scope, " "), nil
	}

	// If not in extras, try to introspect token if endpoint is configured
	if config.Config != nil && config.Config.Endpoint.TokenURL != "" {
		return introspectToken(ctx, token, config)
	}

	// If no scopes are found and none are required, return empty list
	if len(config.RequiredScopes) == 0 {
		return []string{}, nil
	}

	return nil, fmt.Errorf("no scopes found for token")
}

func introspectToken(ctx context.Context, token *oauth2.Token, config OAuth2Config) ([]string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create introspection request
	req, err := http.NewRequestWithContext(ctx, "POST", config.Config.Endpoint.TokenURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token introspection failed: %s", resp.Status)
	}

	var result struct {
		Active bool   `json:"active"`
		Scope  string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.Active {
		return nil, ErrInvalidToken
	}

	if result.Scope == "" {
		return []string{}, nil
	}

	return strings.Split(result.Scope, " "), nil
}

func hasRequiredScopes(tokenScopes, requiredScopes []string) bool {
	if len(requiredScopes) == 0 {
		return true
	}

	scopeMap := make(map[string]bool)
	for _, scope := range tokenScopes {
		scopeMap[scope] = true
	}

	for _, required := range requiredScopes {
		if !scopeMap[required] {
			return false
		}
	}

	return true
}

func (c *tokenCache) get(accessToken string) *oauth2.Token {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[accessToken]
	if !exists {
		return nil
	}

	// Check if token has expired in cache
	if time.Now().After(entry.validUntil) {
		// Token expired, remove it from cache
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.cache, accessToken)
		c.mu.Unlock()
		c.mu.RLock()
		return nil
	}

	return entry.token
}

func (c *tokenCache) set(accessToken string, token *oauth2.Token, duration time.Duration) {
	if duration <= 0 {
		return // Don't cache tokens with no duration
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[accessToken] = tokenCacheEntry{
		token:      token,
		validUntil: time.Now().Add(duration),
	}
}
