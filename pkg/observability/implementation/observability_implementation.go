package implementation

import (
	"context"
	"net/http"

	"github.com/jt828/go-grpc-template/pkg/observability"
)

type observabilityImplementation struct {
	log    observability.Logger
	meter  observability.Meter
	tracer observability.Tracer

	metricsServer *http.Server
	traceClose    func(context.Context) error
}

func (o *observabilityImplementation) Close(ctx context.Context) error {
	var err error
	if o.metricsServer != nil {
		err = o.metricsServer.Shutdown(ctx)
	}
	if o.traceClose != nil {
		if e := o.traceClose(ctx); err == nil {
			err = e
		}
	}
	return err
}
func (o *observabilityImplementation) Logger() observability.Logger { return o.log }
func (o *observabilityImplementation) Meter() observability.Meter   { return o.meter }
func (o *observabilityImplementation) Start(ctx context.Context) error {
	if pm, ok := o.meter.(*prometheusMeter); ok {
		o.metricsServer = StartMetricsServer(":9090", pm.Registry())
	}
	return nil
}
func (o *observabilityImplementation) Tracer() observability.Tracer { return o.tracer }
