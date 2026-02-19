package observability

import "context"

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
	End()
	RecordError(err error)
}
