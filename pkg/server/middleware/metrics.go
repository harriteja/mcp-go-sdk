package middleware

import (
	"net/http"
	"strconv"
	"time"

	httputil "github.com/harriteja/mcp-go-sdk/pkg/server/transport/http"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsConfig represents configuration for the metrics middleware
type MetricsConfig struct {
	// Registry is the Prometheus registry to use
	Registry prometheus.Registerer

	// Subsystem is the metrics subsystem name
	Subsystem string

	// ExcludePaths are paths to exclude from metrics
	ExcludePaths []string
}

type metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestSize      *prometheus.HistogramVec
	responseSize     *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

func newMetrics(subsystem string) *metrics {
	m := &metrics{}

	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	m.requestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "request_size_bytes",
			Help:      "HTTP request size in bytes",
			Buckets:   []float64{16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		},
		[]string{"method", "path"},
	)

	m.responseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: subsystem,
			Name:      "response_size_bytes",
			Help:      "HTTP response size in bytes",
			Buckets:   []float64{16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536},
		},
		[]string{"method", "path"},
	)

	m.requestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: subsystem,
			Name:      "requests_in_flight",
			Help:      "Current number of requests being served",
		},
	)

	return m
}

// register registers all metrics with the provided registry
func (m *metrics) register(reg prometheus.Registerer) error {
	if reg == nil {
		return nil
	}

	collectors := []prometheus.Collector{
		m.requestsTotal,
		m.requestDuration,
		m.requestSize,
		m.responseSize,
		m.requestsInFlight,
	}

	for _, collector := range collectors {
		if err := reg.Register(collector); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	return nil
}

// MetricsMiddleware creates a new metrics middleware
func MetricsMiddleware(config MetricsConfig) func(http.Handler) http.Handler {
	// Create metrics
	m := newMetrics(config.Subsystem)

	// Register metrics
	if config.Registry != nil {
		if err := m.register(config.Registry); err != nil {
			panic(err)
		}
	}

	// Create exclude paths map
	excludePaths := make(map[string]bool)
	for _, path := range config.ExcludePaths {
		excludePaths[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip excluded paths
			if excludePaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Track in-flight requests
			m.requestsInFlight.Inc()
			defer m.requestsInFlight.Dec()

			// Track request size
			if r.ContentLength > 0 {
				m.requestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
			}

			// Create response writer wrapper
			rw := httputil.NewResponseWriter(w)

			// Track duration
			start := time.Now()
			next.ServeHTTP(rw, r)
			duration := time.Since(start).Seconds()

			// Record metrics
			status := strconv.Itoa(rw.Status())
			m.requestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
			m.requestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)

			// Only record response size if bytes were written
			if bytesWritten := rw.BytesWritten(); bytesWritten > 0 {
				m.responseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(bytesWritten))
			}
		})
	}
}
