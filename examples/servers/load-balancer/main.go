package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	ReverseProxy *httputil.ReverseProxy
	Requests     uint64
	LastChecked  time.Time
	ResponseTime time.Duration
	mux          sync.RWMutex
}

type LoadBalancer struct {
	logger       *zap.Logger
	backends     []*Backend
	strategy     string
	current      uint64
	healthCheck  time.Duration
	registry     *prometheus.Registry
	requestCount *prometheus.CounterVec
	latency      *prometheus.HistogramVec
}

func NewLoadBalancer(logger *zap.Logger, backends []string, strategy string, healthCheck time.Duration) (*LoadBalancer, error) {
	var backendList []*Backend
	for _, b := range backends {
		url, err := url.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("invalid backend URL %s: %w", b, err)
		}

		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("Reverse proxy error",
				zap.String("backend", url.String()),
				zap.Error(err),
			)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		}

		backendList = append(backendList, &Backend{
			URL:          url,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	// Create metrics
	registry := prometheus.NewRegistry()
	requestCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadbalancer_requests_total",
			Help: "Total number of requests by backend and status",
		},
		[]string{"backend", "status"},
	)
	latency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loadbalancer_response_time_seconds",
			Help:    "Response time by backend",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend"},
	)

	registry.MustRegister(requestCount, latency)

	return &LoadBalancer{
		logger:       logger,
		backends:     backendList,
		strategy:     strategy,
		healthCheck:  healthCheck,
		registry:     registry,
		requestCount: requestCount,
		latency:      latency,
	}, nil
}

func (lb *LoadBalancer) getNextPeer() *Backend {
	switch lb.strategy {
	case "round-robin":
		next := atomic.AddUint64(&lb.current, 1) % uint64(len(lb.backends))
		return lb.backends[next]

	case "least-connections":
		var minReq uint64 = ^uint64(0)
		var selected *Backend
		for _, b := range lb.backends {
			b.mux.RLock()
			if b.Alive && b.Requests < minReq {
				minReq = b.Requests
				selected = b
			}
			b.mux.RUnlock()
		}
		return selected

	case "response-time":
		var minTime time.Duration = time.Hour
		var selected *Backend
		for _, b := range lb.backends {
			b.mux.RLock()
			if b.Alive && b.ResponseTime < minTime {
				minTime = b.ResponseTime
				selected = b
			}
			b.mux.RUnlock()
		}
		return selected

	default:
		return lb.backends[0]
	}
}

func (lb *LoadBalancer) healthCheckBackends() {
	ticker := time.NewTicker(lb.healthCheck)
	defer ticker.Stop()

	for range ticker.C {
		for _, backend := range lb.backends {
			start := time.Now()
			resp, err := http.Get(backend.URL.String() + "/health")
			responseTime := time.Since(start)

			backend.mux.Lock()
			if err != nil {
				backend.Alive = false
				lb.logger.Warn("Backend is down",
					zap.String("backend", backend.URL.String()),
					zap.Error(err),
				)
			} else {
				backend.Alive = resp.StatusCode == http.StatusOK
				backend.ResponseTime = responseTime
				resp.Body.Close()
			}
			backend.LastChecked = time.Now()
			backend.mux.Unlock()
		}
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		return
	}

	peer := lb.getNextPeer()
	if peer == nil {
		http.Error(w, "No available backends", http.StatusServiceUnavailable)
		return
	}

	start := time.Now()
	atomic.AddUint64(&peer.Requests, 1)
	defer atomic.AddUint64(&peer.Requests, ^uint64(0))

	peer.ReverseProxy.ServeHTTP(w, r)

	duration := time.Since(start)
	lb.latency.WithLabelValues(peer.URL.String()).Observe(duration.Seconds())
	lb.requestCount.WithLabelValues(peer.URL.String(), fmt.Sprint(http.StatusOK)).Inc()
}

func main() {
	// Parse command line flags
	addr := flag.String("addr", ":8080", "Load balancer address")
	strategy := flag.String("strategy", "round-robin", "Load balancing strategy (round-robin, least-connections, response-time)")
	healthCheck := flag.Duration("health-check", 5*time.Second, "Health check interval")
	flag.Parse()

	// Get backend list from environment
	backends := os.Args[1:]
	if len(backends) == 0 {
		fmt.Println("Usage: load-balancer [options] backend1 backend2 ...")
		os.Exit(1)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create load balancer
	lb, err := NewLoadBalancer(logger, backends, *strategy, *healthCheck)
	if err != nil {
		logger.Fatal("Failed to create load balancer", zap.Error(err))
	}

	// Start health checker
	go lb.healthCheckBackends()

	// Create HTTP server
	server := &http.Server{
		Addr:    *addr,
		Handler: lb,
	}

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(lb.registry, promhttp.HandlerOpts{}))
		if err := http.ListenAndServe(":9090", mux); err != nil {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start server
	go func() {
		logger.Info("Starting load balancer", zap.String("addr", *addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}
}
