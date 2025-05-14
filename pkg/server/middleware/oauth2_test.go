package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestOAuth2Middleware(t *testing.T) {
	config := OAuth2Config{
		Config: &oauth2.Config{
			ClientID: "test-client",
			Endpoint: oauth2.Endpoint{
				TokenURL: "http://localhost:8080/token",
			},
		},
		RequiredScopes:     []string{"read", "write"},
		TokenCacheDuration: time.Minute,
	}

	tests := []struct {
		name       string
		token      string
		wantStatus int
		setup      func(*OAuth2Config)
		validate   func(*testing.T, http.Handler, *httptest.ResponseRecorder)
	}{
		{
			name:       "No token",
			token:      "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Invalid token format",
			token:      "invalid-format",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Valid token",
			token:      "valid-token",
			wantStatus: http.StatusOK,
			setup: func(c *OAuth2Config) {
				c.ValidateToken = func(ctx context.Context, token *oauth2.Token) error {
					if token.AccessToken != "valid-token" {
						return ErrInvalidToken
					}
					return nil
				}
			},
		},
		{
			name:       "Invalid token validation",
			token:      "invalid-token",
			wantStatus: http.StatusUnauthorized,
			setup: func(c *OAuth2Config) {
				c.ValidateToken = func(ctx context.Context, token *oauth2.Token) error {
					return ErrInvalidToken
				}
			},
		},
		{
			name:       "Cached token",
			token:      "cached-token",
			wantStatus: http.StatusOK,
			setup: func(c *OAuth2Config) {
				c.ValidateToken = func(ctx context.Context, token *oauth2.Token) error {
					if token.AccessToken != "cached-token" {
						return ErrInvalidToken
					}
					return nil
				}
			},
			validate: func(t *testing.T, handler http.Handler, w *httptest.ResponseRecorder) {
				// Make a second request with the same token to test caching
				req := httptest.NewRequest("GET", "http://example.com/foo", nil)
				req.Header.Set("Authorization", "Bearer cached-token")
				w2 := httptest.NewRecorder()

				// Use the same handler instance to ensure cache persistence
				handler.ServeHTTP(w2, req)

				if w2.Code != http.StatusOK {
					t.Errorf("cached token request returned wrong status code: got %v want %v",
						w2.Code, http.StatusOK)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := config
			if tt.setup != nil {
				tt.setup(&testConfig)
			}

			handler := OAuth2Middleware(testConfig)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					w.Code, tt.wantStatus)
			}

			if tt.validate != nil {
				tt.validate(t, handler, w)
			}
		})
	}
}

func TestTokenCache(t *testing.T) {
	cache := &tokenCache{
		cache: make(map[string]tokenCacheEntry),
	}

	token := &oauth2.Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}

	// Test setting and getting a token
	cache.set(token.AccessToken, token, time.Minute)
	got := cache.get(token.AccessToken)
	if got == nil {
		t.Error("Expected to get cached token, got nil")
	}

	// Test expired token
	expiredToken := &oauth2.Token{
		AccessToken: "expired-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}
	cache.set(expiredToken.AccessToken, expiredToken, -time.Minute) // Set with negative duration to force expiration
	time.Sleep(time.Millisecond)                                    // Ensure expiration time has passed
	got = cache.get(expiredToken.AccessToken)
	if got != nil {
		t.Error("Expected nil for expired token, got token")
	}

	// Verify expired token was removed from cache
	if _, exists := cache.cache[expiredToken.AccessToken]; exists {
		t.Error("Expected expired token to be removed from cache")
	}

	// Test non-existent token
	got = cache.get("non-existent")
	if got != nil {
		t.Error("Expected nil for non-existent token, got token")
	}
}

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{
			name:    "Valid token",
			header:  "Bearer valid-token",
			want:    "valid-token",
			wantErr: false,
		},
		{
			name:    "No token",
			header:  "",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			header:  "invalid-format",
			wantErr: true,
		},
		{
			name:    "Wrong scheme",
			header:  "Basic valid-token",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			got, err := extractToken(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.AccessToken != tt.want {
				t.Errorf("extractToken() = %v, want %v", got.AccessToken, tt.want)
			}
		})
	}
}
