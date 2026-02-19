package implementation

import (
	"context"

	"github.com/jt828/go-grpc-template/pkg/observability"
)

type Config struct {
	ServiceName string
}

func NewObservability(cfg Config) (observability.Observability, error) {
	log, err := NewZapLogger()
	if err != nil {
		return nil, err
	}

	meter := NewPrometheusMeter()

	tracer, shutdown, err := NewOtelTracer(context.Background(), cfg.ServiceName)
	if err != nil {
		return nil, err
	}

	return &observabilityImplementation{
		log:        log,
		meter:      meter,
		tracer:     tracer,
		traceClose: shutdown,
	}, nil
}
