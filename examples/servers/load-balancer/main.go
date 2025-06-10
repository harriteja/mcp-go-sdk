package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/logger"
	"github.com/harriteja/mcp-go-sdk/pkg/server/middleware"
	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

// Backend represents a backend server
type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy http.Handler
	mux          sync.RWMutex
	// Additional fields for monitoring
	Requests       int64
	LastRequestAt  time.Time
	ResponseTimes  []time.Duration
	maxResponseLog int
}

// SetAlive updates the alive status of the backend
func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Alive = alive
}

// IsAlive returns true if the backend is alive
func (b *Backend) IsAlive() bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.Alive
}

// RecordRequest records a request to the backend
func (b *Backend) RecordRequest(start time.Time) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.Requests++
	b.LastRequestAt = time.Now()
	responseTime := time.Since(start)
	// Keep only the most recent response times
	if len(b.ResponseTimes) >= b.maxResponseLog {
		// Shift array left, dropping the oldest entry
		copy(b.ResponseTimes, b.ResponseTimes[1:])
		b.ResponseTimes[len(b.ResponseTimes)-1] = responseTime
	} else {
		b.ResponseTimes = append(b.ResponseTimes, responseTime)
	}
}

// LoadBalancer represents a load balancer
type LoadBalancer struct {
	logger        types.Logger
	backends      []*Backend
	strategy      string
	roundRobinIdx int
	mutex         sync.Mutex
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(logger types.Logger, backendURLs []string, strategy string) *LoadBalancer {
	backends := make([]*Backend, 0, len(backendURLs))
	ctx := context.Background()

	for _, backendURL := range backendURLs {
		u, err := url.Parse(backendURL)
		if err != nil {
			logger.Error(ctx, "loadbalancer", "backend", fmt.Sprintf("Failed to parse backend URL: %v", err))
			continue
		}

		backend := &Backend{
			URL:            u,
			Alive:          true,
			ResponseTimes:  make([]time.Duration, 0, 100),
			maxResponseLog: 100,
		}

		backends = append(backends, backend)
		logger.Info(ctx, "loadbalancer", "backend", fmt.Sprintf("Registered backend: %s", backendURL))
	}

	lb := &LoadBalancer{
		logger:        logger,
		backends:      backends,
		strategy:      strategy,
		roundRobinIdx: 0,
	}

	// Start health check for all backends
	go lb.healthCheck()

	return lb
}

// nextBackend returns the next available backend based on the selected strategy
func (lb *LoadBalancer) nextBackend() *Backend {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Filter alive backends
	aliveBackends := make([]*Backend, 0)
	for _, b := range lb.backends {
		if b.IsAlive() {
			aliveBackends = append(aliveBackends, b)
		}
	}

	// If no backends are alive, return nil
	if len(aliveBackends) == 0 {
		return nil
	}

	var next *Backend
	switch lb.strategy {
	case "round-robin":
		// Simple round-robin strategy
		next = aliveBackends[lb.roundRobinIdx%len(aliveBackends)]
		lb.roundRobinIdx++
	case "random":
		// Random selection strategy
		next = aliveBackends[rand.Intn(len(aliveBackends))]
	case "least-connections":
		// Select the backend with the least requests
		leastRequests := int64(1<<63 - 1) // max int64
		for _, b := range aliveBackends {
			if b.Requests < leastRequests {
				leastRequests = b.Requests
				next = b
			}
		}
	default:
		// Default to round-robin
		next = aliveBackends[lb.roundRobinIdx%len(aliveBackends)]
		lb.roundRobinIdx++
	}

	return next
}

// ServeHTTP handles HTTP requests
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Get the next backend
	backend := lb.nextBackend()
	if backend == nil {
		lb.logger.Error(ctx, "loadbalancer", "proxy", "No available backend servers")
		http.Error(w, "No available backend servers", http.StatusServiceUnavailable)
		return
	}

	// Log the request
	lb.logger.Info(ctx, "loadbalancer", "proxy", fmt.Sprintf("Proxying request to %s", backend.URL.String()))

	// Record the request start time
	start := time.Now()

	// Forward the request
	req, err := http.NewRequest(r.Method, backend.URL.String()+r.URL.Path, r.Body)
	if err != nil {
		lb.logger.Error(ctx, "loadbalancer", "proxy", fmt.Sprintf("Error creating proxy request: %v", err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Copy request headers
	for name, values := range r.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		lb.logger.Error(ctx, "loadbalancer", "proxy", fmt.Sprintf("Error proxying request: %v", err))
		http.Error(w, "Error proxying request", http.StatusBadGateway)
		backend.SetAlive(false)
		return
	}
	defer resp.Body.Close()

	// Record the request
	backend.RecordRequest(start)

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		lb.logger.Error(ctx, "loadbalancer", "proxy", fmt.Sprintf("Error copying response: %v", err))
	}

	lb.logger.Info(ctx, "loadbalancer", "proxy", fmt.Sprintf("Request completed in %v", time.Since(start)))
}

// healthCheck performs periodic health checks on all backends
func (lb *LoadBalancer) healthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		for _, b := range lb.backends {
			status := "up"
			if !b.IsAlive() {
				status = "down"
			}
			lb.logger.Info(ctx, "loadbalancer", "health", fmt.Sprintf("Checking backend %s (current status: %s)", b.URL.String(), status))

			alive := isBackendAlive(b.URL)
			b.SetAlive(alive)
			if alive {
				lb.logger.Info(ctx, "loadbalancer", "health", fmt.Sprintf("Backend %s is alive", b.URL.String()))
			} else {
				lb.logger.Warn(ctx, "loadbalancer", "health", fmt.Sprintf("Backend %s is down", b.URL.String()))
			}
		}
	}
}

// isBackendAlive checks if the backend is alive
func isBackendAlive(u *url.URL) bool {
	resp, err := http.Get(u.String() + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// BackendStatus represents the status of a backend server
type BackendStatus struct {
	URL          string        `json:"url"`
	Alive        bool          `json:"alive"`
	Requests     int64         `json:"requests"`
	LastRequest  string        `json:"last_request,omitempty"`
	ResponseTime time.Duration `json:"avg_response_time,omitempty"`
}

// StatusHandler returns a handler for the /status endpoint
func (lb *LoadBalancer) StatusHandler(w http.ResponseWriter, r *http.Request) {
	statuses := make([]BackendStatus, len(lb.backends))

	for i, b := range lb.backends {
		b.mux.RLock()
		status := BackendStatus{
			URL:      b.URL.String(),
			Alive:    b.Alive,
			Requests: b.Requests,
		}

		if !b.LastRequestAt.IsZero() {
			status.LastRequest = b.LastRequestAt.Format(time.RFC3339)
		}

		// Calculate average response time
		if len(b.ResponseTimes) > 0 {
			var total time.Duration
			for _, d := range b.ResponseTimes {
				total += d
			}
			status.ResponseTime = total / time.Duration(len(b.ResponseTimes))
		}
		b.mux.RUnlock()

		statuses[i] = status
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Load balancer address")
	strategy := flag.String("strategy", "round-robin", "Load balancing strategy (round-robin, random, least-connections)")
	flag.Parse()

	// Get backend servers from args
	backends := flag.Args()
	if len(backends) == 0 {
		fmt.Println("No backend servers specified")
		os.Exit(1)
	}

	// Initialize logger
	stdLogger := logger.New("load-balancer")
	ctx := context.Background()

	// Create load balancer
	lb := NewLoadBalancer(stdLogger, backends, *strategy)

	// Create metrics registry
	registry := prometheus.NewRegistry()

	// Create server mux
	mux := http.NewServeMux()

	// Register handlers
	mux.Handle("/", lb)
	mux.HandleFunc("/status", lb.StatusHandler)

	// Add metrics middleware
	handler := middleware.MetricsMiddleware(middleware.MetricsConfig{
		Registry:     registry,
		Subsystem:    "loadbalancer",
		ExcludePaths: []string{"/metrics", "/status"},
	})(mux)

	// Create HTTP server
	server := &http.Server{
		Addr:         *addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
	go func() {
		stdLogger.Info(ctx, "loadbalancer", "server", fmt.Sprintf("Starting load balancer on %s", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stdLogger.Error(ctx, "loadbalancer", "server", fmt.Sprintf("Failed to start server: %v", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		stdLogger.Error(ctx, "loadbalancer", "server", fmt.Sprintf("Failed to stop server gracefully: %v", err))
	}
}
