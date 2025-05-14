package types

import (
	"fmt"
	"regexp"
)

var (
	// validMetricNameRegex defines the pattern for valid metric names
	validMetricNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// validLabelNameRegex defines the pattern for valid label names
	validLabelNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// MetricType represents the type of metric
type MetricType string

const (
	// MetricTypeCounter represents a counter metric that only increases
	MetricTypeCounter MetricType = "counter"
	// MetricTypeGauge represents a gauge metric that can go up and down
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeHistogram represents a histogram metric for value distributions
	MetricTypeHistogram MetricType = "histogram"
	// MetricTypeSummary represents a summary metric for value distributions with quantiles
	MetricTypeSummary MetricType = "summary"
)

// MetricLabel represents a label/tag for a metric with validation
type MetricLabel struct {
	Name  string
	Value string
}

// Validate checks if the label name and value are valid
func (l MetricLabel) Validate() error {
	if !validLabelNameRegex.MatchString(l.Name) {
		return fmt.Errorf("invalid label name: %s", l.Name)
	}
	return nil
}

// MetricOpts represents options for creating a metric
type MetricOpts struct {
	// Namespace for grouping metrics (e.g., "app" or "system")
	Namespace string
	// Subsystem for grouping metrics within a namespace (e.g., "http" or "db")
	Subsystem string
	// Name of the metric
	Name string
	// Help text describing the metric
	Help string
	// Type of metric
	Type MetricType
	// Labels/tags for the metric
	Labels []MetricLabel
	// Buckets for histogram metrics (must be sorted in ascending order)
	Buckets []float64
	// Objectives for summary metrics (map of quantile to error margin)
	Objectives map[float64]float64
}

// Validate checks if the metric options are valid
func (o MetricOpts) Validate() error {
	if o.Namespace != "" && !validMetricNameRegex.MatchString(o.Namespace) {
		return fmt.Errorf("invalid namespace: %s", o.Namespace)
	}
	if o.Subsystem != "" && !validMetricNameRegex.MatchString(o.Subsystem) {
		return fmt.Errorf("invalid subsystem: %s", o.Subsystem)
	}
	if !validMetricNameRegex.MatchString(o.Name) {
		return fmt.Errorf("invalid metric name: %s", o.Name)
	}
	for _, label := range o.Labels {
		if err := label.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Metric represents a single metric with validation and batch operations
type Metric interface {
	// Inc increments a counter metric
	Inc(labels ...MetricLabel)
	// Add adds a value to a counter metric
	Add(value float64, labels ...MetricLabel)
	// Set sets the value of a gauge metric
	Set(value float64, labels ...MetricLabel)
	// Observe records a value for histogram/summary metrics
	Observe(value float64, labels ...MetricLabel)
	// BatchInc increments multiple counter metrics with the same labels
	BatchInc(count int, labels ...MetricLabel)
	// BatchObserve records multiple observations with the same labels
	BatchObserve(values []float64, labels ...MetricLabel)
}

// Timer represents a timer for measuring durations with additional features
type Timer interface {
	// ObserveDuration records the duration since the timer was created
	ObserveDuration(labels ...MetricLabel)
	// ObserveDurationWithQuantiles records the duration and returns quantile estimates
	ObserveDurationWithQuantiles(quantiles []float64, labels ...MetricLabel) map[float64]float64
}

// MetricsCollector collects and manages metrics with validation and batching
type MetricsCollector interface {
	// NewMetric creates a new metric with validation
	NewMetric(opts MetricOpts) (Metric, error)
	// NewTimer creates a new timer with the given name and labels
	NewTimer(name string, labels ...MetricLabel) Timer
	// Register registers metrics with the collector
	Register(metrics ...Metric) error
	// Unregister removes metrics from the collector
	Unregister(metrics ...Metric) error
	// WithNamespace returns a new collector with the given namespace
	WithNamespace(namespace string) MetricsCollector
	// WithDefaultLabels returns a new collector with default labels
	WithDefaultLabels(labels ...MetricLabel) MetricsCollector
}

// NoOpMetric implements Metric with no-op operations
type NoOpMetric struct{}

func (m *NoOpMetric) Inc(labels ...MetricLabel)                            {}
func (m *NoOpMetric) Add(value float64, labels ...MetricLabel)             {}
func (m *NoOpMetric) Set(value float64, labels ...MetricLabel)             {}
func (m *NoOpMetric) Observe(value float64, labels ...MetricLabel)         {}
func (m *NoOpMetric) BatchInc(count int, labels ...MetricLabel)            {}
func (m *NoOpMetric) BatchObserve(values []float64, labels ...MetricLabel) {}

// NoOpTimer implements Timer with no-op operations
type NoOpTimer struct{}

func (t *NoOpTimer) ObserveDuration(labels ...MetricLabel) {}
func (t *NoOpTimer) ObserveDurationWithQuantiles(quantiles []float64, labels ...MetricLabel) map[float64]float64 {
	return make(map[float64]float64)
}

// NoOpMetricsCollector implements MetricsCollector with no-op operations
type NoOpMetricsCollector struct{}

func (c *NoOpMetricsCollector) NewMetric(opts MetricOpts) (Metric, error) {
	return &NoOpMetric{}, nil
}

func (c *NoOpMetricsCollector) NewTimer(name string, labels ...MetricLabel) Timer {
	return &NoOpTimer{}
}

func (c *NoOpMetricsCollector) Register(metrics ...Metric) error {
	return nil
}

func (c *NoOpMetricsCollector) Unregister(metrics ...Metric) error {
	return nil
}

func (c *NoOpMetricsCollector) WithNamespace(namespace string) MetricsCollector {
	return c
}

func (c *NoOpMetricsCollector) WithDefaultLabels(labels ...MetricLabel) MetricsCollector {
	return c
}

// NewNoOpMetricsCollector creates a new no-op metrics collector
func NewNoOpMetricsCollector() MetricsCollector {
	return &NoOpMetricsCollector{}
}
