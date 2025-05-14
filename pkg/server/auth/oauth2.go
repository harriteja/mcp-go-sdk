package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/oauth2"
)

// ClientInfo represents OAuth2 client information
type ClientInfo struct {
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret,omitempty"`
	RedirectURIs []string  `json:"redirect_uris"`
	Scopes       []string  `json:"scopes"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

// TokenInfo represents OAuth2 token information
type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scopes       []string  `json:"scopes"`
}

// OAuthProvider defines the interface for OAuth2 operations
type OAuthProvider interface {
	// RegisterClient registers a new OAuth2 client
	RegisterClient(ctx context.Context, info *ClientInfo) error

	// ValidateToken validates an OAuth2 token
	ValidateToken(ctx context.Context, token string) (*TokenInfo, error)

	// RevokeToken revokes an OAuth2 token
	RevokeToken(ctx context.Context, token string) error

	// GenerateToken generates a new OAuth2 token
	GenerateToken(ctx context.Context, clientID, code string) (*TokenInfo, error)

	// RefreshToken refreshes an OAuth2 token
	RefreshToken(ctx context.Context, refreshToken string) (*TokenInfo, error)
}

// DefaultOAuthProvider implements the OAuthProvider interface
type DefaultOAuthProvider struct {
	config *oauth2.Config
	store  TokenStore
}

// TokenStore defines the interface for token storage
type TokenStore interface {
	// StoreClient stores client information
	StoreClient(ctx context.Context, info *ClientInfo) error

	// GetClient retrieves client information
	GetClient(ctx context.Context, clientID string) (*ClientInfo, error)

	// StoreToken stores token information
	StoreToken(ctx context.Context, clientID string, info *TokenInfo) error

	// GetToken retrieves token information
	GetToken(ctx context.Context, token string) (*TokenInfo, error)

	// DeleteToken deletes token information
	DeleteToken(ctx context.Context, token string) error
}

// NewOAuthProvider creates a new OAuth provider
func NewOAuthProvider(config *oauth2.Config, store TokenStore) OAuthProvider {
	return &DefaultOAuthProvider{
		config: config,
		store:  store,
	}
}

// RegisterClient implements OAuthProvider.RegisterClient
func (p *DefaultOAuthProvider) RegisterClient(ctx context.Context, info *ClientInfo) error {
	// Generate client ID and secret if not provided
	if info.ClientID == "" {
		clientID, err := generateRandomString(32)
		if err != nil {
			return fmt.Errorf("failed to generate client ID: %w", err)
		}
		info.ClientID = clientID
	}

	if info.ClientSecret == "" {
		secret, err := generateRandomString(64)
		if err != nil {
			return fmt.Errorf("failed to generate client secret: %w", err)
		}
		info.ClientSecret = secret
	}

	info.CreatedAt = time.Now()

	return p.store.StoreClient(ctx, info)
}

// ValidateToken implements OAuthProvider.ValidateToken
func (p *DefaultOAuthProvider) ValidateToken(ctx context.Context, token string) (*TokenInfo, error) {
	info, err := p.store.GetToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if info == nil {
		return nil, fmt.Errorf("token not found")
	}

	if time.Now().After(info.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return info, nil
}

// RevokeToken implements OAuthProvider.RevokeToken
func (p *DefaultOAuthProvider) RevokeToken(ctx context.Context, token string) error {
	return p.store.DeleteToken(ctx, token)
}

// GenerateToken implements OAuthProvider.GenerateToken
func (p *DefaultOAuthProvider) GenerateToken(ctx context.Context, clientID, code string) (*TokenInfo, error) {
	// Verify client
	client, err := p.store.GetClient(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if client == nil {
		return nil, fmt.Errorf("client not found")
	}

	// Generate tokens
	accessToken, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	info := &TokenInfo{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes:       client.Scopes,
	}

	if err := p.store.StoreToken(ctx, clientID, info); err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return info, nil
}

// RefreshToken implements OAuthProvider.RefreshToken
func (p *DefaultOAuthProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenInfo, error) {
	// Get existing token
	oldToken, err := p.store.GetToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	if oldToken == nil {
		return nil, fmt.Errorf("refresh token not found")
	}

	// Generate new tokens
	accessToken, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	info := &TokenInfo{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		RefreshToken: newRefreshToken,
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes:       oldToken.Scopes,
	}

	// Store new token and delete old one
	if err := p.store.DeleteToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to delete old token: %w", err)
	}

	if err := p.store.StoreToken(ctx, "", info); err != nil {
		return nil, fmt.Errorf("failed to store new token: %w", err)
	}

	return info, nil
}

// Helper function to generate random strings
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}
