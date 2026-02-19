package implementation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
)

type idempotencyImpl struct{}

func NewIdempotency() idempotency.Idempotency {
	return &idempotencyImpl{}
}

func (i *idempotencyImpl) Execute(
	ctx context.Context,
	repo idempotency.RecordRepository,
	id int64,
	requestType constant.RequestType,
	referenceId int64,
	newResult func() any,
	fn func() (any, error),
) (any, error) {
	record, err := repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if record != nil {
		result := newResult()
		if err := json.Unmarshal([]byte(record.ResponseData), result); err != nil {
			return nil, err
		}
		return result, nil
	}

	result, err := fn()
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	err = repo.Insert(ctx, &idempotency.Record{
		Id:           id,
		RequestType:  string(requestType),
		ReferenceId:  referenceId,
		ResponseData: string(data),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}
