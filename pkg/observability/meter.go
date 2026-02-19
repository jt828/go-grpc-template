package observability

type Meter interface {
	Counter(name string, opts ...MetricOpt) Counter
	Histogram(name string, opts ...MetricOpt) Histogram
	Gauge(name string, opts ...MetricOpt) Gauge
	Timer(name string, opts ...MetricOpt) Timer
}

type Counter interface {
	Inc(v float64, labels ...Label)
}

type Histogram interface {
	Observe(v float64, labels ...Label)
}

type Gauge interface {
	Set(v float64, labels ...Label)
	Add(v float64, labels ...Label)
}

type Timer interface {
	Start(labels ...Label) func()
}
