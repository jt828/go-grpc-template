package implementation

import (
	"time"

	"github.com/jt828/go-grpc-template/pkg/observability"
	"github.com/prometheus/client_golang/prometheus"
)

type prometheusMeter struct {
	registry    *prometheus.Registry
	constLabels []observability.Label
}

func NewPrometheusMeter() observability.Meter {
	return &prometheusMeter{
		registry: prometheus.NewRegistry(),
	}
}

func (m *prometheusMeter) Registry() *prometheus.Registry {
	return m.registry
}

func PromRegistry(m observability.Meter) *prometheus.Registry {
	if pm, ok := m.(*prometheusMeter); ok {
		return pm.Registry()
	}
	return nil
}

// -------------------- Counter --------------------

type promCounter struct {
	vec *prometheus.CounterVec
}

func (m *prometheusMeter) Counter(name string, opts ...observability.MetricOpt) observability.Counter {
	opt := firstOpt(opts)
	labelKeys := opt.LabelKeys

	vec := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        name,
			Help:        opt.Help,
			ConstLabels: toPromConstLabels(opt.ConstLabels),
		},
		labelKeys,
	)

	m.registry.MustRegister(vec)
	return &promCounter{vec: vec}
}

func (c *promCounter) Inc(v float64, labels ...observability.Label) {
	if len(labels) == 0 {
		c.vec.WithLabelValues().Add(v)
		return
	}
	c.vec.With(toPromLabelsMap(labels)).Add(v)
}

// -------------------- Histogram --------------------

type promHistogram struct {
	vec *prometheus.HistogramVec
}

func (m *prometheusMeter) Histogram(name string, opts ...observability.MetricOpt) observability.Histogram {
	opt := firstOpt(opts)
	labelKeys := opt.LabelKeys

	vec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        opt.Help,
			Buckets:     opt.Buckets,
			ConstLabels: toPromConstLabels(opt.ConstLabels),
		},
		labelKeys,
	)

	m.registry.MustRegister(vec)
	return &promHistogram{vec: vec}
}

func (h *promHistogram) Observe(v float64, labels ...observability.Label) {
	if len(labels) == 0 {
		h.vec.WithLabelValues().Observe(v)
		return
	}
	h.vec.With(toPromLabelsMap(labels)).Observe(v)
}

// -------------------- Gauge --------------------

type promGauge struct {
	vec *prometheus.GaugeVec
}

func (m *prometheusMeter) Gauge(name string, opts ...observability.MetricOpt) observability.Gauge {
	opt := firstOpt(opts)
	labelKeys := getLabelKeys(opt.ConstLabels)

	vec := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        name,
			Help:        opt.Help,
			ConstLabels: toPromConstLabels(opt.ConstLabels),
		},
		labelKeys,
	)

	m.registry.MustRegister(vec)
	return &promGauge{vec: vec}
}

func (g *promGauge) Set(v float64, labels ...observability.Label) {
	if len(labels) == 0 {
		g.vec.WithLabelValues().Set(v)
		return
	}
	g.vec.With(toPromLabelsMap(labels)).Set(v)
}

func (g *promGauge) Add(v float64, labels ...observability.Label) {
	if len(labels) == 0 {
		g.vec.WithLabelValues().Add(v)
		return
	}
	g.vec.With(toPromLabelsMap(labels)).Add(v)
}

// -------------------- Timer --------------------

type promTimer struct {
	histogram   *prometheus.HistogramVec
	constLabels []observability.Label
}

func (m *prometheusMeter) Timer(name string, opts ...observability.MetricOpt) observability.Timer {
	opt := firstOpt(opts)
	labelKeys := getLabelKeys(opt.ConstLabels)

	vec := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        name,
			Help:        opt.Help,
			Buckets:     opt.Buckets,
			ConstLabels: toPromConstLabels(opt.ConstLabels),
		},
		labelKeys,
	)

	m.registry.MustRegister(vec)

	return &promTimer{
		histogram:   vec,
		constLabels: opt.ConstLabels,
	}
}

func (t *promTimer) Start(labels ...observability.Label) func() {
	start := time.Now()
	return func() {
		merged := mergeLabels(t.constLabels, labels)
		t.histogram.With(merged).Observe(time.Since(start).Seconds())
	}
}

// -------------------- Helpers --------------------

func firstOpt(opts []observability.MetricOpt) observability.MetricOpt {
	if len(opts) == 0 {
		return observability.MetricOpt{}
	}
	return opts[0]
}

func getLabelKeys(labels []observability.Label) []string {
	keys := make([]string, len(labels))
	for i, l := range labels {
		keys[i] = l.Key
	}
	return keys
}

func toPromLabelsMap(labels []observability.Label) prometheus.Labels {
	m := make(prometheus.Labels, len(labels))
	for _, l := range labels {
		m[l.Key] = l.Value
	}
	return m
}

func toPromConstLabels(labels []observability.Label) prometheus.Labels {
	return toPromLabelsMap(labels)
}

func mergeLabels(constLabels, dynamicLabels []observability.Label) prometheus.Labels {
	m := make(prometheus.Labels, len(constLabels)+len(dynamicLabels))
	for _, l := range constLabels {
		m[l.Key] = l.Value
	}
	for _, l := range dynamicLabels {
		m[l.Key] = l.Value
	}
	return m
}
