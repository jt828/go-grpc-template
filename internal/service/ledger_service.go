package service

import (
	"context"

	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/pkg/model"
)

type GetParams struct {
	IdEq              int64
	UserIdEq          int64
	TransactionTypeEq string
	TokenEq           string
}

type LedgerService interface {
	GetLedgers(ctx context.Context, params GetParams) ([]*model.Ledger, error)
}

type ledgerService struct {
	uowFactory repository.UnitOfWorkFactory
}

func NewLedgerService(uowFactory repository.UnitOfWorkFactory) LedgerService {
	return &ledgerService{uowFactory: uowFactory}
}

func (s *ledgerService) GetLedgers(ctx context.Context, params GetParams) ([]*model.Ledger, error) {
	uow, err := s.uowFactory.New()
	if err != nil {
		return nil, err
	}

	ledgers, err := uow.LedgerRepository().Get(ctx, repository.GetQuery{
		IdEq:              params.IdEq,
		UserIdEq:          params.UserIdEq,
		TransactionTypeEq: params.TransactionTypeEq,
		TokenEq:           params.TokenEq,
	})
	if err != nil {
		_ = uow.Abort(ctx)
		return nil, err
	}

	if err := uow.Commit(ctx); err != nil {
		return nil, err
	}

	return ledgers, nil
}
