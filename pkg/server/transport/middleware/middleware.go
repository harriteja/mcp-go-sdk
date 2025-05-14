package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/harriteja/mcp-go-sdk/pkg/server/transport/errors"
)

// Middleware represents a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain combines multiple middleware into a single middleware
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// RateLimit creates a rate limiting middleware
func RateLimit(rps float64, burst int) Middleware {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limiter.Wait(r.Context()); err != nil {
				errors.WriteError(w, http.StatusTooManyRequests, "Too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodySize creates a request size limiting middleware
func MaxBodySize(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				errors.WriteError(w, http.StatusRequestEntityTooLarge, "Request too large")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SQLInjectionPrevention creates a middleware that checks for basic SQL injection attempts
func SQLInjectionPrevention() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			if strings.Contains(strings.ToUpper(query), "DROP TABLE") ||
				strings.Contains(strings.ToUpper(query), "DELETE FROM") {
				errors.WriteError(w, http.StatusBadRequest, "Invalid request")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout creates a middleware that adds a timeout to the request context
func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Auth creates an authentication middleware
func Auth(validateToken func(string) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get(errors.HeaderAuthorization)
			if !validateToken(token) {
				errors.WriteError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS creates a CORS middleware
func CORS(allowedOrigins []string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set(errors.HeaderAccessControlAllowOrigin, origin)
					break
				}
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Recovery creates a middleware that recovers from panics
func Recovery(logger interface{ Error(string, ...interface{}) }) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic recovered",
						"error", err,
						"path", r.URL.Path,
						"method", r.Method,
					)
					errors.WriteError(w, http.StatusInternalServerError, "Internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
