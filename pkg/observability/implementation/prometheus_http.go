package implementation

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(
	addr string,
	reg *prometheus.Registry,
) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() { _ = srv.ListenAndServe() }()

	return srv
}
