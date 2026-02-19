package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/internal/service"
	cbImpl "github.com/jt828/go-grpc-template/pkg/circuitbreaker/implementation"
	idempotencyImpl "github.com/jt828/go-grpc-template/pkg/idempotency/implementation"
	"github.com/jt828/go-grpc-template/pkg/retry"
	retryImpl "github.com/jt828/go-grpc-template/pkg/retry/implementation"
	snowflakeImpl "github.com/jt828/go-grpc-template/pkg/snowflake/implementation"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupCreateUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts(
			"testdata/init_schema.sql",
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(pgdriver.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	return db
}

func TestCreateUser_Idempotency(t *testing.T) {
	db := setupCreateUserTestDB(t)
	ctx := context.Background()

	cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})
	r := retryImpl.NewRetry(3, retry.WithInterval(100*time.Millisecond), retry.WithRetryable(func(err error) bool {
		return false
	}))

	uowFactory := repository.NewTransactionDbUnitOfWorkFactory(db, cb, r)
	idem := idempotencyImpl.NewIdempotency()
	sf, err := snowflakeImpl.NewSnowflake(1)
	require.NoError(t, err)

	userSvc := service.NewUserService(uowFactory, idem, sf)

	t.Run("create new user with new idempotency key", func(t *testing.T) {
		user := &model.User{
			Email:     "test@example.com",
			Username:  "testuser",
			Password:  "hashed_password",
			CreatedAt: time.Now().Truncate(time.Second),
			UpdatedAt: time.Now().Truncate(time.Second),
		}

		created, err := userSvc.CreateUser(ctx, 1001, user)
		require.NoError(t, err)
		assert.NotZero(t, created.Id)
		assert.Equal(t, "test@example.com", created.Email)
		assert.Equal(t, "testuser", created.Username)
	})

	t.Run("return existing user when idempotency key is duplicated", func(t *testing.T) {
		user := &model.User{
			Email:     "another@example.com",
			Username:  "anotheruser",
			Password:  "hashed_password",
			CreatedAt: time.Now().Truncate(time.Second),
			UpdatedAt: time.Now().Truncate(time.Second),
		}

		// same idempotency key 1001 as above
		replayed, err := userSvc.CreateUser(ctx, 1001, user)
		require.NoError(t, err)

		// should return the original user, not the new one
		assert.Equal(t, "test@example.com", replayed.Email)
		assert.Equal(t, "testuser", replayed.Username)
	})

	t.Run("create different user with different idempotency key", func(t *testing.T) {
		user := &model.User{
			Email:     "second@example.com",
			Username:  "seconduser",
			Password:  "hashed_password",
			CreatedAt: time.Now().Truncate(time.Second),
			UpdatedAt: time.Now().Truncate(time.Second),
		}

		created, err := userSvc.CreateUser(ctx, 2002, user)
		require.NoError(t, err)
		assert.NotZero(t, created.Id)
		assert.Equal(t, "second@example.com", created.Email)
		assert.Equal(t, "seconduser", created.Username)
	})
}
