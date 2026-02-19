package repository

import (
	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/retry"
	"gorm.io/gorm"
)

type UnitOfWorkFactory interface {
	New() (UnitOfWork, error)
}

type transactionDbUnitOfWorkFactory struct {
	db    *gorm.DB
	cb    circuitbreaker.CircuitBreaker
	retry retry.Retry
}

func NewTransactionDbUnitOfWorkFactory(db *gorm.DB, cb circuitbreaker.CircuitBreaker, retry retry.Retry) UnitOfWorkFactory {
	return &transactionDbUnitOfWorkFactory{db: db, cb: cb, retry: retry}
}

func (f *transactionDbUnitOfWorkFactory) New() (UnitOfWork, error) {
	tx := f.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &transactionDbUnitOfWork{tx: tx, cb: f.cb, retry: f.retry}, nil
}
