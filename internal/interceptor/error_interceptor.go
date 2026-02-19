package interceptor

import (
	"context"
	"errors"
	"fmt"

	"github.com/jt828/go-grpc-template/pkg/apperror"
	"github.com/jt828/go-grpc-template/pkg/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ErrorInterceptor(log observability.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", observability.String("panic", fmt.Sprintf("%v", r)), observability.String("method", info.FullMethod))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		resp, err = handler(ctx, req)
		if err == nil {
			return resp, nil
		}

		switch {
		case errors.Is(err, apperror.ErrNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, apperror.ErrInvalidArgument):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			log.Error("unhandled error", observability.Err(err), observability.String("method", info.FullMethod))
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}
}
