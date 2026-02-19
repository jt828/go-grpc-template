package unit

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jt828/go-grpc-template/internal/interceptor"
	"github.com/jt828/go-grpc-template/pkg/apperror"
	"github.com/jt828/go-grpc-template/pkg/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockLogger struct {
	errorCalls []struct {
		msg    string
		fields []observability.Field
	}
}

func (m *mockLogger) Debug(msg string, fields ...observability.Field) {}
func (m *mockLogger) Error(msg string, fields ...observability.Field) {
	m.errorCalls = append(m.errorCalls, struct {
		msg    string
		fields []observability.Field
	}{msg, fields})
}
func (m *mockLogger) Fatal(msg string, fields ...observability.Field)            {}
func (m *mockLogger) Info(msg string, fields ...observability.Field)             {}
func (m *mockLogger) Warn(msg string, fields ...observability.Field)             {}
func (m *mockLogger) With(fields ...observability.Field) observability.Logger    { return m }

func TestErrorInterceptor(t *testing.T) {
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	t.Run("no error passes through unchanged", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		resp, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return "ok", nil
		})

		require.NoError(t, err)
		assert.Equal(t, "ok", resp)
		assert.Len(t, log.errorCalls, 0)
	})

	t.Run("ErrNotFound maps to codes.NotFound", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, apperror.ErrNotFound
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Equal(t, apperror.ErrNotFound.Error(), st.Message())
		assert.Len(t, log.errorCalls, 0)
	})

	t.Run("wrapped ErrNotFound maps to codes.NotFound", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, fmt.Errorf("user lookup: %w", apperror.ErrNotFound)
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Len(t, log.errorCalls, 0)
	})

	t.Run("ErrInvalidArgument maps to codes.InvalidArgument", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, apperror.ErrInvalidArgument
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, apperror.ErrInvalidArgument.Error(), st.Message())
		assert.Len(t, log.errorCalls, 0)
	})

	t.Run("wrapped ErrInvalidArgument maps to codes.InvalidArgument", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, fmt.Errorf("validation: %w", apperror.ErrInvalidArgument)
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Len(t, log.errorCalls, 0)
	})

	t.Run("unknown error maps to codes.Internal with generic message", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, errors.New("database exploded")
		})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "internal server error", st.Message())
	})

	t.Run("unknown error logs with error and method fields", func(t *testing.T) {
		log := &mockLogger{}
		i := interceptor.ErrorInterceptor(log)

		unknownErr := errors.New("some internal failure")
		_, err := i(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
			return nil, unknownErr
		})

		require.Error(t, err)
		require.Len(t, log.errorCalls, 1)
		assert.Equal(t, "unhandled error", log.errorCalls[0].msg)
		assert.Contains(t, log.errorCalls[0].fields, observability.Err(unknownErr))
		assert.Contains(t, log.errorCalls[0].fields, observability.String("method", info.FullMethod))
	})
}