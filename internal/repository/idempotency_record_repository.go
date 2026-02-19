package repository

import (
	"context"
	"errors"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/pkg/retry"
	"gorm.io/gorm"
)

type IdempotencyRecordRepositoryImpl struct {
	db              *gorm.DB
	cb              circuitbreaker.CircuitBreaker
	retry           retry.Retry
	notFoundAsError bool
}

func NewIdempotencyRecordRepository(db *gorm.DB, cb circuitbreaker.CircuitBreaker, retry retry.Retry, notFoundAsError bool) idempotency.RecordRepository {
	return &IdempotencyRecordRepositoryImpl{db: db, cb: cb, retry: retry, notFoundAsError: notFoundAsError}
}

func (r *IdempotencyRecordRepositoryImpl) Get(ctx context.Context, id int64) (*idempotency.Record, error) {
	result, err := r.cb.Execute(func() (any, error) {
		var record *idempotency.Record
		err := r.retry.Execute(ctx, func() error {
			var entity model.IdempotencyRecordDataEntity
			if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
				if !r.notFoundAsError && errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			domain := entity.ToDomain()
			record = &domain
			return nil
		})
		if err != nil {
			return nil, err
		}
		return record, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*idempotency.Record), nil
}

func (r *IdempotencyRecordRepositoryImpl) Insert(ctx context.Context, record *idempotency.Record) error {
	_, err := r.cb.Execute(func() (any, error) {
		err := r.retry.Execute(ctx, func() error {
			entity := model.IdempotencyRecordDataEntity{
				Id:           record.Id,
				RequestType:  constant.RequestType(record.RequestType),
				ReferenceId:  record.ReferenceId,
				ResponseData: record.ResponseData,
				CreatedAt:    record.CreatedAt,
			}
			return r.db.WithContext(ctx).Create(&entity).Error
		})
		return nil, err
	})
	return err
}
