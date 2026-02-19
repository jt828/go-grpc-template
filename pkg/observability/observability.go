package observability

import "context"

type Observability interface {
	Close(ctx context.Context) error
	Logger() Logger
	Meter() Meter
	Start(ctx context.Context) error
	Tracer() Tracer
}
