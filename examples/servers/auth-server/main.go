package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/metrics"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/ratelimit"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role"`
}

type AuthService struct {
	logger    types.Logger
	metrics   types.MetricsCollector
	users     map[string]User
	userMutex sync.RWMutex
	jwtKey    []byte
	limiter   ratelimit.Limiter

	// Metrics
	loginAttempts   types.Metric
	loginSuccess    types.Metric
	loginFailure    types.Metric
	requestDuration types.Metric
}

func NewAuthService(logger types.Logger, metrics types.MetricsCollector, jwtKey string, rateLimit int) (*AuthService, error) {
	// Create metrics
	loginAttempts, err := metrics.NewMetric(types.MetricOpts{
		Name: "auth_login_attempts_total",
		Help: "Total number of login attempts",
		Type: types.MetricTypeCounter,
		Labels: []types.MetricLabel{
			{Name: "status", Value: ""},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create login attempts metric: %w", err)
	}

	loginSuccess, err := metrics.NewMetric(types.MetricOpts{
		Name: "auth_login_success_total",
		Help: "Total number of successful logins",
		Type: types.MetricTypeCounter,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create login success metric: %w", err)
	}

	loginFailure, err := metrics.NewMetric(types.MetricOpts{
		Name: "auth_login_failure_total",
		Help: "Total number of failed logins",
		Type: types.MetricTypeCounter,
		Labels: []types.MetricLabel{
			{Name: "reason", Value: ""},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create login failure metric: %w", err)
	}

	requestDuration, err := metrics.NewMetric(types.MetricOpts{
		Name: "auth_request_duration_seconds",
		Help: "Request duration in seconds",
		Type: types.MetricTypeHistogram,
		Labels: []types.MetricLabel{
			{Name: "endpoint", Value: ""},
		},
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration metric: %w", err)
	}

	// Register metrics
	if err := metrics.Register(loginAttempts, loginSuccess, loginFailure, requestDuration); err != nil {
		return nil, fmt.Errorf("failed to register metrics: %w", err)
	}

	return &AuthService{
		logger:          logger,
		metrics:         metrics,
		users:           make(map[string]User),
		jwtKey:          []byte(jwtKey),
		limiter:         ratelimit.New(rateLimit),
		loginAttempts:   loginAttempts,
		loginSuccess:    loginSuccess,
		loginFailure:    loginFailure,
		requestDuration: requestDuration,
	}, nil
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}

func (s *AuthService) HandleRegister(w http.ResponseWriter, r *http.Request) {
	timer := s.metrics.NewTimer("auth_request_duration_seconds", types.MetricLabel{Name: "endpoint", Value: "register"})
	defer timer.ObserveDuration()

	// Apply rate limiting
	s.limiter.Take()

	if r.Method != http.MethodPost {
		s.logger.Warn("Method not allowed", types.LogField{Key: "method", Value: r.Method})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		s.logger.Error("Invalid request body", types.LogField{Key: "error", Value: err.Error()})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", types.LogField{Key: "error", Value: err.Error()})
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	s.userMutex.Lock()
	if _, exists := s.users[user.Username]; exists {
		s.userMutex.Unlock()
		s.logger.Warn("Username already exists", types.LogField{Key: "username", Value: user.Username})
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	// Store user with hashed password
	user.Password = string(hashedPassword)
	s.users[user.Username] = user
	s.userMutex.Unlock()

	s.logger.Info("User registered successfully", types.LogField{Key: "username", Value: user.Username})
	w.WriteHeader(http.StatusCreated)
}

func (s *AuthService) HandleLogin(w http.ResponseWriter, r *http.Request) {
	timer := s.metrics.NewTimer("auth_request_duration_seconds", types.MetricLabel{Name: "endpoint", Value: "login"})
	defer timer.ObserveDuration()

	// Apply rate limiting
	s.limiter.Take()

	if r.Method != http.MethodPost {
		s.logger.Warn("Method not allowed", types.LogField{Key: "method", Value: r.Method})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Error("Invalid request body", types.LogField{Key: "error", Value: err.Error()})
		s.loginFailure.Inc(types.MetricLabel{Name: "reason", Value: "invalid_request"})
		writeJSON(w, LoginResponse{Error: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	s.loginAttempts.Inc(types.MetricLabel{Name: "status", Value: "attempt"})

	s.userMutex.RLock()
	user, exists := s.users[req.Username]
	s.userMutex.RUnlock()

	if !exists {
		s.logger.Warn("Invalid credentials - user not found", types.LogField{Key: "username", Value: req.Username})
		s.loginFailure.Inc(types.MetricLabel{Name: "reason", Value: "user_not_found"})
		writeJSON(w, LoginResponse{Error: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logger.Warn("Invalid credentials - wrong password", types.LogField{Key: "username", Value: req.Username})
		s.loginFailure.Inc(types.MetricLabel{Name: "reason", Value: "wrong_password"})
		writeJSON(w, LoginResponse{Error: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		s.logger.Error("Failed to generate token", types.LogField{Key: "error", Value: err.Error()})
		s.loginFailure.Inc(types.MetricLabel{Name: "reason", Value: "token_generation_failed"})
		writeJSON(w, LoginResponse{Error: "Failed to generate token"}, http.StatusInternalServerError)
		return
	}

	s.logger.Info("Login successful", types.LogField{Key: "username", Value: user.Username})
	s.loginSuccess.Inc()
	writeJSON(w, LoginResponse{Token: tokenString}, http.StatusOK)
}

func (s *AuthService) HandleProtected(w http.ResponseWriter, r *http.Request) {
	timer := s.metrics.NewTimer("auth_request_duration_seconds", types.MetricLabel{Name: "endpoint", Value: "protected"})
	defer timer.ObserveDuration()

	// Apply rate limiting
	s.limiter.Take()

	if r.Method != http.MethodGet {
		s.logger.Warn("Method not allowed", types.LogField{Key: "method", Value: r.Method})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		s.logger.Warn("Invalid authorization header")
		http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
		return
	}

	tokenString := authHeader[7:]

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})

	if err != nil || !token.Valid {
		s.logger.Warn("Invalid token", types.LogField{Key: "error", Value: err.Error()})
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Get claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Error("Invalid token claims")
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	s.logger.Info("Protected endpoint accessed",
		types.LogField{Key: "username", Value: claims["username"].(string)},
		types.LogField{Key: "role", Value: claims["role"].(string)},
	)

	// Return user info
	writeJSON(w, map[string]interface{}{
		"username": claims["username"],
		"role":     claims["role"],
	}, http.StatusOK)
}

func writeJSON(w http.ResponseWriter, response interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Server address")
	jwtKey := flag.String("jwt-key", "your-secret-key", "JWT signing key")
	rateLimit := flag.Int("rate-limit", 100, "Requests per second limit")
	flag.Parse()

	// Initialize logger
	loggerFactory := logger.NewZapLoggerFactory(logger.DefaultZapConfig())
	log := loggerFactory.CreateLogger("auth-server")

	// Initialize metrics collector
	metricsCollector := metrics.NewPrometheusCollector(prometheus.NewRegistry())

	// Create auth service
	authService, err := NewAuthService(log, metricsCollector, *jwtKey, *rateLimit)
	if err != nil {
		log.Error("Failed to create auth service", types.LogField{Key: "error", Value: err.Error()})
		os.Exit(1)
	}

	// Create server mux
	mux := http.NewServeMux()

	// Register handlers
	mux.HandleFunc("/register", authService.HandleRegister)
	mux.HandleFunc("/login", authService.HandleLogin)
	mux.HandleFunc("/protected", authService.HandleProtected)

	// Add metrics endpoint
	if collector, ok := metricsCollector.(*metrics.PrometheusCollector); ok {
		mux.Handle("/metrics", promhttp.HandlerFor(collector.GetRegistry(), promhttp.HandlerOpts{}))
	}

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
	go func() {
		log.Info("Starting server", types.LogField{Key: "addr", Value: *addr})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", types.LogField{Key: "error", Value: err.Error()})
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Failed to stop server gracefully", types.LogField{Key: "error", Value: err.Error()})
	}
}
