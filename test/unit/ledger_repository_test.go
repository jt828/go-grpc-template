package unit

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type passthroughCB struct{}

func (p *passthroughCB) Execute(fn func() (any, error)) (any, error) { return fn() }
func (p *passthroughCB) State() circuitbreaker.State                 { return circuitbreaker.Closed }

type passthroughRetry struct{}

func (p *passthroughRetry) Execute(ctx context.Context, fn func() error) error { return fn() }

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	gormDB, err := gorm.Open(pgdriver.New(pgdriver.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	return gormDB, mock
}

func ledgerColumns() []string {
	return []string{"id", "user_id", "transaction_type", "token", "amount", "created_at"}
}

func TestLedgerRepository_Get(t *testing.T) {
	ctx := context.Background()
	cb := &passthroughCB{}
	r := &passthroughRetry{}
	now := time.Now().Truncate(time.Second)
	amt := decimal.NewFromFloat(1.5)

	t.Run("no filters returns all records", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers"`)).
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now).
					AddRow(2, 20, "withdraw", "BTC", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{})
		require.NoError(t, err)
		assert.Len(t, ledgers, 2)
		assert.Equal(t, int64(1), ledgers[0].Id)
		assert.Equal(t, int64(2), ledgers[1].Id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filter by IdEq", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE id = $1`)).
			WithArgs(int64(5)).
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(5, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{IdEq: 5})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.Equal(t, int64(5), ledgers[0].Id)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filter by UserIdEq", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE user_id = $1`)).
			WithArgs(int64(10)).
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{UserIdEq: 10})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.Equal(t, int64(10), ledgers[0].UserId)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filter by TransactionTypeEq", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE transaction_type = $1`)).
			WithArgs("deposit").
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{TransactionTypeEq: "deposit"})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.Equal(t, "deposit", ledgers[0].TransactionType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("filter by TokenEq", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE token = $1`)).
			WithArgs("ETH").
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{TokenEq: "ETH"})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.Equal(t, "ETH", ledgers[0].Token)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("multiple filters combined", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "main"."ledgers" WHERE user_id = $1 AND transaction_type = $2 AND token = $3`,
		)).
			WithArgs(int64(10), "deposit", "ETH").
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{
			UserIdEq:          10,
			TransactionTypeEq: "deposit",
			TokenEq:           "ETH",
		})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("all filters combined", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(
			`SELECT * FROM "main"."ledgers" WHERE id = $1 AND user_id = $2 AND transaction_type = $3 AND token = $4`,
		)).
			WithArgs(int64(1), int64(10), "deposit", "ETH").
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(1, 10, "deposit", "ETH", amt, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{
			IdEq:              1,
			UserIdEq:          10,
			TransactionTypeEq: "deposit",
			TokenEq:           "ETH",
		})
		require.NoError(t, err)
		assert.Len(t, ledgers, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no results returns empty slice", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE id = $1`)).
			WithArgs(int64(999)).
			WillReturnRows(sqlmock.NewRows(ledgerColumns()))

		ledgers, err := repo.Get(ctx, repository.GetQuery{IdEq: 999})
		require.NoError(t, err)
		assert.Empty(t, ledgers)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error is propagated", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		dbErr := errors.New("connection refused")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers"`)).
			WillReturnError(dbErr)

		ledgers, err := repo.Get(ctx, repository.GetQuery{})
		assert.Nil(t, ledgers)
		assert.ErrorContains(t, err, "connection refused")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("entity to domain conversion preserves all fields", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		amount := decimal.NewFromFloat(123.456789)

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "main"."ledgers" WHERE id = $1`)).
			WithArgs(int64(42)).
			WillReturnRows(
				sqlmock.NewRows(ledgerColumns()).
					AddRow(42, 100, "transfer", "USDC", amount, now),
			)

		ledgers, err := repo.Get(ctx, repository.GetQuery{IdEq: 42})
		require.NoError(t, err)
		require.Len(t, ledgers, 1)

		l := ledgers[0]
		assert.Equal(t, int64(42), l.Id)
		assert.Equal(t, int64(100), l.UserId)
		assert.Equal(t, "transfer", l.TransactionType)
		assert.Equal(t, "USDC", l.Token)
		assert.True(t, amount.Equal(l.Amount))
		assert.Equal(t, now, l.CreatedAt)
	})
}

func TestLedgerRepository_Insert(t *testing.T) {
	ctx := context.Background()
	cb := &passthroughCB{}
	r := &passthroughRetry{}
	now := time.Now().Truncate(time.Second)
	amt := decimal.NewFromFloat(50.25)

	t.Run("successful insert", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "main"."ledgers" ("user_id","transaction_type","token","amount","created_at","id") VALUES ($1,$2,$3,$4,$5,$6) RETURNING "id"`,
		)).
			WithArgs(int64(10), "deposit", "ETH", amt, now, int64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectCommit()

		err := repo.Insert(ctx, &model.Ledger{
			Id:              1,
			UserId:          10,
			TransactionType: "deposit",
			Token:           "ETH",
			Amount:          amt,
			CreatedAt:       now,
		})
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("insert error is propagated", func(t *testing.T) {
		gormDB, mock := setupMockDB(t)
		repo := repository.NewLedgerRepository(gormDB, cb, r, false)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(
			`INSERT INTO "main"."ledgers"`,
		)).
			WillReturnError(errors.New("duplicate key"))
		mock.ExpectRollback()

		err := repo.Insert(ctx, &model.Ledger{
			Id:              1,
			UserId:          10,
			TransactionType: "deposit",
			Token:           "ETH",
			Amount:          amt,
			CreatedAt:       now,
		})
		assert.ErrorContains(t, err, "duplicate key")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
