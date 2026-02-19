package idempotency

import (
	"context"

	"github.com/jt828/go-grpc-template/internal/constant"
)

type RecordRepository interface {
	Get(ctx context.Context, id int64) (*Record, error)
	Insert(ctx context.Context, record *Record) error
}

type Idempotency interface {
	Execute(ctx context.Context, repo RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error)
}
