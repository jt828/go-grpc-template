package integration

import (
	"context"
	"fmt"
	"net"
	"testing"

	"github.com/jt828/go-grpc-template/internal/interceptor"
	"github.com/jt828/go-grpc-template/pkg/apperror"
	"github.com/jt828/go-grpc-template/pkg/observability"
	v1 "github.com/jt828/go-grpc-template/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type noopLogger struct{}

func (n *noopLogger) Debug(msg string, fields ...observability.Field)         {}
func (n *noopLogger) Error(msg string, fields ...observability.Field)         {}
func (n *noopLogger) Fatal(msg string, fields ...observability.Field)         {}
func (n *noopLogger) Info(msg string, fields ...observability.Field)          {}
func (n *noopLogger) Warn(msg string, fields ...observability.Field)          {}
func (n *noopLogger) With(fields ...observability.Field) observability.Logger { return n }

type errorControlledServer struct {
	v1.UnimplementedUserServiceServer
	err error
}

func (s *errorControlledServer) GetUserById(_ context.Context, _ *v1.GetUserByIdRequest) (*v1.GetUserByIdResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &v1.GetUserByIdResponse{}, nil
}

func setupInterceptorServer(t *testing.T, svc v1.UserServiceServer) v1.UserServiceClient {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := grpc.NewServer(grpc.UnaryInterceptor(interceptor.ErrorInterceptor(&noopLogger{})))
	v1.RegisterUserServiceServer(srv, svc)
	t.Cleanup(func() { srv.GracefulStop() })
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return v1.NewUserServiceClient(conn)
}

func TestErrorInterceptor_Integration(t *testing.T) {
	t.Run("no error returns response", func(t *testing.T) {
		client := setupInterceptorServer(t, &errorControlledServer{err: nil})

		resp, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 1})

		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("ErrNotFound becomes codes.NotFound", func(t *testing.T) {
		client := setupInterceptorServer(t, &errorControlledServer{err: apperror.ErrNotFound})

		_, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 1})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Equal(t, apperror.ErrNotFound.Error(), st.Message())
	})

	t.Run("wrapped ErrNotFound becomes codes.NotFound", func(t *testing.T) {
		wrapped := fmt.Errorf("user 42: %w", apperror.ErrNotFound)
		client := setupInterceptorServer(t, &errorControlledServer{err: wrapped})

		_, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 42})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("ErrInvalidArgument becomes codes.InvalidArgument", func(t *testing.T) {
		client := setupInterceptorServer(t, &errorControlledServer{err: apperror.ErrInvalidArgument})

		_, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 1})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, apperror.ErrInvalidArgument.Error(), st.Message())
	})

	t.Run("wrapped ErrInvalidArgument becomes codes.InvalidArgument", func(t *testing.T) {
		wrapped := fmt.Errorf("validation failed: %w", apperror.ErrInvalidArgument)
		client := setupInterceptorServer(t, &errorControlledServer{err: wrapped})

		_, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 1})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("unknown error becomes codes.Internal with generic message", func(t *testing.T) {
		client := setupInterceptorServer(t, &errorControlledServer{err: fmt.Errorf("database exploded")})

		_, err := client.GetUserById(context.Background(), &v1.GetUserByIdRequest{Id: 1})

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Equal(t, "internal server error", st.Message())
	})
}