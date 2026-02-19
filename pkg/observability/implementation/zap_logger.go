package implementation

import (
	"github.com/jt828/go-grpc-template/pkg/observability"
	"go.uber.org/zap"
)

type zapLogger struct {
	l *zap.Logger
}

func NewZapLogger() (observability.Logger, error) {
	l, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &zapLogger{l: l}, nil
}

func toZap(fields []observability.Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	out := make([]zap.Field, 0, len(fields))

	for _, f := range fields {
		out = append(out, zap.Any(f.Key, f.Value))
	}

	return out
}

func (z *zapLogger) Debug(msg string, fields ...observability.Field) {
	z.l.Debug(msg, toZap(fields)...)
}

func (z *zapLogger) Error(msg string, fields ...observability.Field) {
	z.l.Error(msg, toZap(fields)...)
}

func (z *zapLogger) Fatal(msg string, fields ...observability.Field) {
	z.l.Fatal(msg, toZap(fields)...)
}

func (z *zapLogger) Info(msg string, fields ...observability.Field) {
	z.l.Info(msg, toZap(fields)...)
}

func (z *zapLogger) Warn(msg string, fields ...observability.Field) {
	z.l.Warn(msg, toZap(fields)...)
}

func (z *zapLogger) With(fields ...observability.Field) observability.Logger {
	return &zapLogger{
		l: z.l.With(toZap(fields)...),
	}
}
