package repository

import (
	"context"
	"sync"

	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
	"github.com/jt828/go-grpc-template/pkg/retry"
	"gorm.io/gorm"
)

type UnitOfWork interface {
	Commit(ctx context.Context) error
	Abort(ctx context.Context) error
	UserRepository() UserRepository
	LedgerRepository() LedgerRepository
	IdempotencyRecordRepository() idempotency.RecordRepository
}

type transactionDbUnitOfWork struct {
	tx                              *gorm.DB
	cb                              circuitbreaker.CircuitBreaker
	retry                           retry.Retry
	userRepository                  UserRepository
	userRepositoryOnce              sync.Once
	ledgerRepository                LedgerRepository
	ledgerRepositoryOnce            sync.Once
	idempotencyRecordRepository     idempotency.RecordRepository
	idempotencyRecordRepositoryOnce sync.Once
}

func (u *transactionDbUnitOfWork) UserRepository() UserRepository {
	u.userRepositoryOnce.Do(func() {
		u.userRepository = NewUserRepository(u.tx, u.cb, u.retry, false)
	})
	return u.userRepository
}

func (u *transactionDbUnitOfWork) LedgerRepository() LedgerRepository {
	u.ledgerRepositoryOnce.Do(func() {
		u.ledgerRepository = NewLedgerRepository(u.tx, u.cb, u.retry, false)
	})
	return u.ledgerRepository
}

func (u *transactionDbUnitOfWork) IdempotencyRecordRepository() idempotency.RecordRepository {
	u.idempotencyRecordRepositoryOnce.Do(func() {
		u.idempotencyRecordRepository = NewIdempotencyRecordRepository(u.tx, u.cb, u.retry, false)
	})
	return u.idempotencyRecordRepository
}

func (u *transactionDbUnitOfWork) Commit(ctx context.Context) error {
	return u.tx.WithContext(ctx).Commit().Error
}

func (u *transactionDbUnitOfWork) Abort(ctx context.Context) error {
	return u.tx.WithContext(ctx).Rollback().Error
}
