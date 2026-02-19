package integration

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/internal/repository"
	cbImpl "github.com/jt828/go-grpc-template/pkg/circuitbreaker/implementation"
	"github.com/jt828/go-grpc-template/pkg/retry"
	retryImpl "github.com/jt828/go-grpc-template/pkg/retry/implementation"
	"github.com/sony/gobreaker/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testDB struct {
	db        *gorm.DB
	container *tcpostgres.PostgresContainer
}

func setupTestDB(t *testing.T) *testDB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts("testdata/init_schema.sql"),
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

	return &testDB{db: db, container: pgContainer}
}

func seedUser(t *testing.T, db *gorm.DB, user *model.UserDataEntity) {
	t.Helper()
	err := db.Create(user).Error
	require.NoError(t, err)
}

func TestUserRepository_Get(t *testing.T) {
	tdb := setupTestDB(t)
	cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{Name: "test"})

	now := time.Now().Truncate(time.Second)
	seedUser(t, tdb.db, &model.UserDataEntity{
		Id:        1,
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashed_password",
		CreatedAt: now,
		UpdatedAt: now,
	})

	r := retryImpl.NewRetry(3, retry.WithInterval(100*time.Millisecond), retry.WithRetryable(func(err error) bool {
		return false
	}))
	repo := repository.NewUserRepository(tdb.db, cb, r, false)

	t.Run("existing user", func(t *testing.T) {
		user, err := repo.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), user.Id)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "hashed_password", user.Password)
	})

	t.Run("non-existing user", func(t *testing.T) {
		user, err := repo.Get(context.Background(), 999)
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("non-existing user with notFoundAsError", func(t *testing.T) {
		repoWithError := repository.NewUserRepository(tdb.db, cb, r, true)
		user, err := repoWithError.Get(context.Background(), 999)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserRepository_RetryOnDBRestart(t *testing.T) {
	tdb := setupTestDB(t)
	ctx := context.Background()

	cb := cbImpl.NewCircuitBreaker(gobreaker.Settings{
		Name: "test",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return false // disable circuit breaker tripping for this test
		},
	})

	r := retryImpl.NewRetry(5, retry.WithInterval(500*time.Millisecond), retry.WithRetryable(func(err error) bool {
		return true // retry all errors
	}))

	now := time.Now().Truncate(time.Second)
	seedUser(t, tdb.db, &model.UserDataEntity{
		Id:        1,
		Email:     "test@example.com",
		Username:  "testuser",
		Password:  "hashed_password",
		CreatedAt: now,
		UpdatedAt: now,
	})

	repo := repository.NewUserRepository(tdb.db, cb, r, false)

	// get docker client and container ID for pause/unpause
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(t, err)

	containerID := tdb.container.GetContainerID()

	// pause DB container to simulate connection loss
	err = dockerClient.ContainerPause(ctx, containerID)
	require.NoError(t, err)

	// unpause after a short delay in background
	go func() {
		time.Sleep(2 * time.Second)
		_ = dockerClient.ContainerUnpause(ctx, containerID)
	}()

	// query should fail initially but succeed after unpause via retry
	user, err := repo.Get(ctx, 1)
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, int64(1), user.Id)
	assert.Equal(t, "testuser", user.Username)
}
