package unit

import (
	"context"
	"errors"
	"testing"

	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/internal/service"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLedgerRepository struct {
	getFunc    func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error)
	insertFunc func(ctx context.Context, ledger *model.Ledger) error
}

func (m *mockLedgerRepository) Get(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
	return m.getFunc(ctx, query)
}

func (m *mockLedgerRepository) Insert(ctx context.Context, ledger *model.Ledger) error {
	return m.insertFunc(ctx, ledger)
}

func TestLedgerService_GetLedgers(t *testing.T) {
	ctx := context.Background()

	t.Run("returns ledgers successfully", func(t *testing.T) {
		expected := []*model.Ledger{
			{Id: 1, UserId: 10, TransactionType: "deposit", Token: "ETH", Amount: decimal.NewFromFloat(1.5)},
			{Id: 2, UserId: 10, TransactionType: "withdraw", Token: "BTC", Amount: decimal.NewFromFloat(0.5)},
		}
		committed := false

		uow := &mockUnitOfWork{
			ledgerRepo: &mockLedgerRepository{
				getFunc: func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
					assert.Equal(t, int64(10), query.UserIdEq)
					assert.Equal(t, "ETH", query.TokenEq)
					return expected, nil
				},
			},
			commitFunc: func(ctx context.Context) error { committed = true; return nil },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
		)

		ledgers, err := svc.GetLedgers(ctx, service.GetParams{UserIdEq: 10, TokenEq: "ETH"})
		require.NoError(t, err)
		assert.Equal(t, expected, ledgers)
		assert.True(t, committed)
	})

	t.Run("maps all params to query fields", func(t *testing.T) {
		var capturedQuery repository.GetQuery

		uow := &mockUnitOfWork{
			ledgerRepo: &mockLedgerRepository{
				getFunc: func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
					capturedQuery = query
					return nil, nil
				},
			},
			commitFunc: func(ctx context.Context) error { return nil },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
		)

		svc.GetLedgers(ctx, service.GetParams{
			IdEq:              42,
			UserIdEq:          10,
			TransactionTypeEq: "deposit",
			TokenEq:           "USDC",
		})

		assert.Equal(t, int64(42), capturedQuery.IdEq)
		assert.Equal(t, int64(10), capturedQuery.UserIdEq)
		assert.Equal(t, "deposit", capturedQuery.TransactionTypeEq)
		assert.Equal(t, "USDC", capturedQuery.TokenEq)
	})

	t.Run("returns empty slice when no results", func(t *testing.T) {
		uow := &mockUnitOfWork{
			ledgerRepo: &mockLedgerRepository{
				getFunc: func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
					return []*model.Ledger{}, nil
				},
			},
			commitFunc: func(ctx context.Context) error { return nil },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
		)

		ledgers, err := svc.GetLedgers(ctx, service.GetParams{})
		require.NoError(t, err)
		assert.Empty(t, ledgers)
	})

	t.Run("uow factory error is propagated", func(t *testing.T) {
		factoryErr := errors.New("cannot begin transaction")

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return nil, factoryErr }},
		)

		ledgers, err := svc.GetLedgers(ctx, service.GetParams{})
		assert.Nil(t, ledgers)
		assert.ErrorIs(t, err, factoryErr)
	})

	t.Run("repository error aborts and is propagated", func(t *testing.T) {
		repoErr := errors.New("db error")
		aborted := false

		uow := &mockUnitOfWork{
			ledgerRepo: &mockLedgerRepository{
				getFunc: func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
					return nil, repoErr
				},
			},
			commitFunc: func(ctx context.Context) error { t.Fatal("commit should not be called"); return nil },
			abortFunc:  func(ctx context.Context) error { aborted = true; return nil },
		}

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
		)

		ledgers, err := svc.GetLedgers(ctx, service.GetParams{})
		assert.Nil(t, ledgers)
		assert.ErrorIs(t, err, repoErr)
		assert.True(t, aborted)
	})

	t.Run("commit error is propagated", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		uow := &mockUnitOfWork{
			ledgerRepo: &mockLedgerRepository{
				getFunc: func(ctx context.Context, query repository.GetQuery) ([]*model.Ledger, error) {
					return []*model.Ledger{{Id: 1}}, nil
				},
			},
			commitFunc: func(ctx context.Context) error { return commitErr },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewLedgerService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
		)

		ledgers, err := svc.GetLedgers(ctx, service.GetParams{})
		assert.Nil(t, ledgers)
		assert.ErrorIs(t, err, commitErr)
	})
}
