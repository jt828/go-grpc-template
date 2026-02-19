package observability

type Label struct {
	Key   string
	Value string
}

type MetricOpt struct {
	Help        string
	Buckets     []float64
	ConstLabels []Label
	LabelKeys   []string
	Unit        string
}
