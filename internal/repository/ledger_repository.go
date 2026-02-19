package repository

import (
	"context"

	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/pkg/retry"
	"gorm.io/gorm"
)

type LedgerRepository interface {
	Get(ctx context.Context, query GetQuery) ([]*model.Ledger, error)
	Insert(ctx context.Context, ledger *model.Ledger) error
}

type GetQuery struct {
	IdEq              int64
	UserIdEq          int64
	TransactionTypeEq string
	TokenEq           string
}

type LedgerRepositoryImpl struct {
	db              *gorm.DB
	cb              circuitbreaker.CircuitBreaker
	retry           retry.Retry
	notFoundAsError bool
}

func NewLedgerRepository(db *gorm.DB, cb circuitbreaker.CircuitBreaker, retry retry.Retry, notFoundAsError bool) LedgerRepository {
	return &LedgerRepositoryImpl{db: db, cb: cb, retry: retry, notFoundAsError: notFoundAsError}
}

func (r *LedgerRepositoryImpl) Get(ctx context.Context, query GetQuery) ([]*model.Ledger, error) {
	result, err := r.cb.Execute(func() (any, error) {
		var ledgers []*model.Ledger
		err := r.retry.Execute(ctx, func() error {
			var entities []model.LedgerDataEntity
			db := r.db.WithContext(ctx)
			if query.IdEq != 0 {
				db = db.Where("id = ?", query.IdEq)
			}
			if query.UserIdEq != 0 {
				db = db.Where("user_id = ?", query.UserIdEq)
			}
			if query.TransactionTypeEq != "" {
				db = db.Where("transaction_type = ?", query.TransactionTypeEq)
			}
			if query.TokenEq != "" {
				db = db.Where("token = ?", query.TokenEq)
			}
			if err := db.Find(&entities).Error; err != nil {
				return err
			}
			ledgers = make([]*model.Ledger, len(entities))
			for i := range entities {
				l := entities[i].ToDomain()
				ledgers[i] = &l
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		return ledgers, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]*model.Ledger), nil
}

func (r *LedgerRepositoryImpl) Insert(ctx context.Context, ledger *model.Ledger) error {
	_, err := r.cb.Execute(func() (any, error) {
		err := r.retry.Execute(ctx, func() error {
			entity := model.LedgerDataEntity{
				Id:              ledger.Id,
				UserId:          ledger.UserId,
				TransactionType: ledger.TransactionType,
				Token:           ledger.Token,
				Amount:          ledger.Amount,
				CreatedAt:       ledger.CreatedAt,
			}
			return r.db.WithContext(ctx).Create(&entity).Error
		})
		return nil, err
	})
	return err
}
