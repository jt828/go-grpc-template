package model

import (
	"time"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
)

func (dataEntity *IdempotencyRecordDataEntity) ToDomain() idempotency.Record {
	return idempotency.Record{
		Id:           dataEntity.Id,
		RequestType:  string(dataEntity.RequestType),
		ReferenceId:  dataEntity.ReferenceId,
		ResponseData: dataEntity.ResponseData,
		CreatedAt:    dataEntity.CreatedAt,
	}
}

type IdempotencyRecordDataEntity struct {
	Id           int64                `gorm:"column:id"`
	RequestType  constant.RequestType `gorm:"column:request_type"`
	ReferenceId  int64                `gorm:"column:reference_id"`
	ResponseData string               `gorm:"column:response_data"`
	CreatedAt    time.Time            `gorm:"column:created_at"`
}

func (dataEntity *IdempotencyRecordDataEntity) TableName() string {
	return "main.idempotency_records"
}

type IdempotencyRecord struct {
	Id           int64
	RequestType  constant.RequestType
	ReferenceId  int64
	ResponseData string
	CreatedAt    time.Time
}
