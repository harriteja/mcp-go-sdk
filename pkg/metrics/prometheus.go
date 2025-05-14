package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/harriteja/mcp-go-sdk/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetric adapts Prometheus metrics to our Metric interface
type PrometheusMetric struct {
	collector prometheus.Collector
	vec       interface{} // Can be Counter, Gauge, Histogram, or Summary vec
	labels    []string
}

// convertLabels converts our MetricLabels to Prometheus labels
func convertLabels(labels []types.MetricLabel) prometheus.Labels {
	if len(labels) == 0 {
		return nil
	}
	promLabels := make(prometheus.Labels)
	for _, l := range labels {
		promLabels[l.Name] = l.Value
	}
	return promLabels
}

// getLabelNames extracts label names from MetricLabels
func getLabelNames(labels []types.MetricLabel) []string {
	if len(labels) == 0 {
		return nil
	}
	names := make([]string, len(labels))
	for i, l := range labels {
		names[i] = l.Name
	}
	return names
}

func (m *PrometheusMetric) Inc(labels ...types.MetricLabel) {
	if counter, ok := m.vec.(*prometheus.CounterVec); ok {
		counter.With(convertLabels(labels)).Inc()
	}
}

func (m *PrometheusMetric) Add(value float64, labels ...types.MetricLabel) {
	if counter, ok := m.vec.(*prometheus.CounterVec); ok {
		counter.With(convertLabels(labels)).Add(value)
	}
}

func (m *PrometheusMetric) Set(value float64, labels ...types.MetricLabel) {
	if gauge, ok := m.vec.(*prometheus.GaugeVec); ok {
		gauge.With(convertLabels(labels)).Set(value)
	}
}

func (m *PrometheusMetric) Observe(value float64, labels ...types.MetricLabel) {
	switch vec := m.vec.(type) {
	case *prometheus.HistogramVec:
		vec.With(convertLabels(labels)).Observe(value)
	case *prometheus.SummaryVec:
		vec.With(convertLabels(labels)).Observe(value)
	}
}

func (m *PrometheusMetric) BatchInc(count int, labels ...types.MetricLabel) {
	if counter, ok := m.vec.(*prometheus.CounterVec); ok {
		c := counter.With(convertLabels(labels))
		for i := 0; i < count; i++ {
			c.Inc()
		}
	}
}

func (m *PrometheusMetric) BatchObserve(values []float64, labels ...types.MetricLabel) {
	switch vec := m.vec.(type) {
	case *prometheus.HistogramVec:
		h := vec.With(convertLabels(labels))
		for _, v := range values {
			h.Observe(v)
		}
	case *prometheus.SummaryVec:
		s := vec.With(convertLabels(labels))
		for _, v := range values {
			s.Observe(v)
		}
	}
}

// PrometheusTimer adapts Prometheus timer to our Timer interface
type PrometheusTimer struct {
	start     time.Time
	observer  prometheus.Observer
	labelKeys []string
}

func (t *PrometheusTimer) ObserveDuration(labels ...types.MetricLabel) {
	duration := time.Since(t.start).Seconds()
	if h, ok := t.observer.(prometheus.Histogram); ok {
		h.Observe(duration)
	} else if s, ok := t.observer.(prometheus.Summary); ok {
		s.Observe(duration)
	}
}

func (t *PrometheusTimer) ObserveDurationWithQuantiles(quantiles []float64, labels ...types.MetricLabel) map[float64]float64 {
	duration := time.Since(t.start).Seconds()
	if s, ok := t.observer.(prometheus.Summary); ok {
		s.Observe(duration)
		// Note: Prometheus doesn't provide direct access to quantile values
		// This is a limitation of the implementation
		return make(map[float64]float64)
	}
	return make(map[float64]float64)
}

// PrometheusCollector implements our MetricsCollector interface
type PrometheusCollector struct {
	registry      *prometheus.Registry
	metrics       sync.Map
	namespace     string
	defaultLabels []types.MetricLabel
}

// NewPrometheusCollector creates a new PrometheusCollector
func NewPrometheusCollector(registry *prometheus.Registry) types.MetricsCollector {
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	return &PrometheusCollector{
		registry: registry,
	}
}

func (c *PrometheusCollector) buildMetricName(name string) string {
	if c.namespace == "" {
		return name
	}
	return fmt.Sprintf("%s_%s", c.namespace, name)
}

func (c *PrometheusCollector) mergeLabels(labels []types.MetricLabel) []types.MetricLabel {
	if len(c.defaultLabels) == 0 {
		return labels
	}

	merged := make([]types.MetricLabel, len(c.defaultLabels)+len(labels))
	copy(merged, c.defaultLabels)
	copy(merged[len(c.defaultLabels):], labels)
	return merged
}

func (c *PrometheusCollector) NewMetric(opts types.MetricOpts) (types.Metric, error) {
	// Validate metric options
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metric options: %w", err)
	}

	// Build full metric name with namespace
	name := c.buildMetricName(opts.Name)
	if opts.Subsystem != "" {
		name = fmt.Sprintf("%s_%s", name, opts.Subsystem)
	}

	// Merge default labels with metric labels
	labels := c.mergeLabels(opts.Labels)
	labelNames := getLabelNames(labels)

	var collector prometheus.Collector
	var vec interface{}

	switch opts.Type {
	case types.MetricTypeCounter:
		counterVec := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: name,
				Help: opts.Help,
			},
			labelNames,
		)
		collector = counterVec
		vec = counterVec

	case types.MetricTypeGauge:
		gaugeVec := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: name,
				Help: opts.Help,
			},
			labelNames,
		)
		collector = gaugeVec
		vec = gaugeVec

	case types.MetricTypeHistogram:
		histogramVec := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    name,
				Help:    opts.Help,
				Buckets: opts.Buckets,
			},
			labelNames,
		)
		collector = histogramVec
		vec = histogramVec

	case types.MetricTypeSummary:
		summaryVec := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       name,
				Help:       opts.Help,
				Objectives: opts.Objectives,
			},
			labelNames,
		)
		collector = summaryVec
		vec = summaryVec

	default:
		return nil, fmt.Errorf("unsupported metric type: %s", opts.Type)
	}

	metric := &PrometheusMetric{
		collector: collector,
		vec:       vec,
		labels:    labelNames,
	}

	c.metrics.Store(name, metric)
	return metric, nil
}

func (c *PrometheusCollector) NewTimer(name string, labels ...types.MetricLabel) types.Timer {
	fullName := c.buildMetricName(name)
	metric, ok := c.metrics.Load(fullName)
	if !ok {
		// Create a new histogram metric if it doesn't exist
		m, err := c.NewMetric(types.MetricOpts{
			Name:    name,
			Help:    fmt.Sprintf("Timer metric %s", name),
			Type:    types.MetricTypeHistogram,
			Labels:  c.mergeLabels(labels),
			Buckets: prometheus.DefBuckets,
		})
		if err != nil {
			return &PrometheusTimer{start: time.Now()}
		}
		metric = m
	}

	if pm, ok := metric.(*PrometheusMetric); ok {
		switch vec := pm.vec.(type) {
		case *prometheus.HistogramVec:
			return &PrometheusTimer{
				start:     time.Now(),
				observer:  vec.With(convertLabels(labels)),
				labelKeys: pm.labels,
			}
		case *prometheus.SummaryVec:
			return &PrometheusTimer{
				start:     time.Now(),
				observer:  vec.With(convertLabels(labels)),
				labelKeys: pm.labels,
			}
		}
	}

	return &PrometheusTimer{start: time.Now()}
}

func (c *PrometheusCollector) Register(metrics ...types.Metric) error {
	for _, m := range metrics {
		if pm, ok := m.(*PrometheusMetric); ok {
			if err := c.registry.Register(pm.collector); err != nil {
				return fmt.Errorf("failed to register metric: %w", err)
			}
		}
	}
	return nil
}

func (c *PrometheusCollector) Unregister(metrics ...types.Metric) error {
	for _, m := range metrics {
		if pm, ok := m.(*PrometheusMetric); ok {
			c.registry.Unregister(pm.collector)
		}
	}
	return nil
}

func (c *PrometheusCollector) WithNamespace(namespace string) types.MetricsCollector {
	return &PrometheusCollector{
		registry:      c.registry,
		namespace:     namespace,
		defaultLabels: c.defaultLabels,
	}
}

func (c *PrometheusCollector) WithDefaultLabels(labels ...types.MetricLabel) types.MetricsCollector {
	return &PrometheusCollector{
		registry:      c.registry,
		namespace:     c.namespace,
		defaultLabels: labels,
	}
}

// GetRegistry returns the underlying Prometheus registry
func (c *PrometheusCollector) GetRegistry() *prometheus.Registry {
	return c.registry
}
