package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsMiddleware(t *testing.T) {
	// Create a new registry for testing
	registry := prometheus.NewRegistry()

	// Create middleware config
	config := MetricsConfig{
		Registry:     registry,
		Subsystem:    "test",
		ExcludePaths: []string{"/metrics"},
	}

	// Create middleware
	middleware := MetricsMiddleware(config)

	tests := []struct {
		name           string
		path           string
		method         string
		responseStatus int
		responseSize   int64
		requestSize    int64
		excluded       bool
	}{
		{
			name:           "Basic request",
			path:           "/test",
			method:         "GET",
			responseStatus: http.StatusOK,
			responseSize:   100,
			requestSize:    50,
		},
		{
			name:           "Error request",
			path:           "/error",
			method:         "POST",
			responseStatus: http.StatusBadRequest,
			responseSize:   200,
			requestSize:    150,
		},
		{
			name:           "Excluded path",
			path:           "/metrics",
			method:         "GET",
			responseStatus: http.StatusOK,
			excluded:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseStatus)
				if tt.responseSize > 0 {
					_, _ = w.Write(make([]byte, tt.responseSize))
				}
			}))

			// Create request
			body := strings.NewReader(string(make([]byte, tt.requestSize)))
			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.requestSize > 0 {
				req.ContentLength = tt.requestSize
			}
			w := httptest.NewRecorder()

			// Process request
			handler.ServeHTTP(w, req)

			// Verify response
			if w.Code != tt.responseStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					w.Code, tt.responseStatus)
			}

			if !tt.excluded {
				// Verify metrics
				metrics, err := registry.Gather()
				if err != nil {
					t.Fatalf("Failed to gather metrics: %v", err)
				}

				// Helper function to get metric value
				getMetricValue := func(name string, labels map[string]string) float64 {
					for _, mf := range metrics {
						if *mf.Name == "test_"+name {
							for _, m := range mf.Metric {
								matches := true
								for k, v := range labels {
									for _, l := range m.Label {
										if *l.Name == k && *l.Value != v {
											matches = false
											break
										}
									}
								}
								if matches {
									if m.Counter != nil {
										return *m.Counter.Value
									}
									if m.Histogram != nil {
										return float64(*m.Histogram.SampleCount)
									}
									if m.Gauge != nil {
										return *m.Gauge.Value
									}
								}
							}
						}
					}
					return 0
				}

				// Verify total requests
				labels := map[string]string{
					"method": tt.method,
					"path":   tt.path,
					"status": strconv.Itoa(tt.responseStatus),
				}
				value := getMetricValue("requests_total", labels)
				if value != 1 {
					t.Errorf("Wrong request count: got %v want 1", value)
				}

				// Verify request size
				if tt.requestSize > 0 {
					labels = map[string]string{
						"method": tt.method,
						"path":   tt.path,
					}
					value = getMetricValue("request_size_bytes", labels)
					if value == 0 {
						t.Error("Expected non-zero request size metric")
					}
				}

				// Verify response size
				if tt.responseSize > 0 {
					labels = map[string]string{
						"method": tt.method,
						"path":   tt.path,
					}
					value = getMetricValue("response_size_bytes", labels)
					if value == 0 {
						t.Error("Expected non-zero response size metric")
					}
				}

				// Verify request duration
				labels = map[string]string{
					"method": tt.method,
					"path":   tt.path,
				}
				value = getMetricValue("request_duration_seconds", labels)
				if value == 0 {
					t.Error("Expected non-zero duration metric")
				}

				// Verify in-flight requests
				value = getMetricValue("requests_in_flight", nil)
				if value != 0 {
					t.Errorf("Expected 0 in-flight requests, got %v", value)
				}
			} else {
				// Verify no metrics were recorded for excluded paths
				metrics, err := registry.Gather()
				if err != nil {
					t.Fatalf("Failed to gather metrics: %v", err)
				}

				for _, mf := range metrics {
					for _, m := range mf.Metric {
						for _, l := range m.Label {
							if *l.Name == "path" && *l.Value == tt.path {
								t.Error("Got metrics for excluded path")
							}
						}
					}
				}
			}
		})
	}
}

func TestMetricsRegistration(t *testing.T) {
	tests := []struct {
		name      string
		registry  prometheus.Registerer
		subsystem string
		wantErr   bool
	}{
		{
			name:      "Valid registration",
			registry:  prometheus.NewRegistry(),
			subsystem: "test",
			wantErr:   false,
		},
		{
			name:      "Duplicate registration",
			registry:  prometheus.NewRegistry(),
			subsystem: "test",
			wantErr:   false, // Should not error due to AlreadyRegisteredError handling
		},
		{
			name:      "Nil registry",
			registry:  nil,
			subsystem: "test",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMetrics(tt.subsystem)
			err := m.register(tt.registry)

			if (err != nil) != tt.wantErr {
				t.Errorf("register() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Try registering again to test duplicate registration
			if tt.name == "Duplicate registration" {
				err = m.register(tt.registry)
				if err != nil {
					t.Errorf("register() error on duplicate = %v", err)
				}
			}
		})
	}
}
