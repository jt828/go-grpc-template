package unit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/internal/service"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockSnowflake struct {
	id int64
}

func (m *mockSnowflake) Generate() int64 { return m.id }

type mockUserRepository struct {
	getFunc    func(ctx context.Context, id int64) (*model.User, error)
	insertFunc func(ctx context.Context, user *model.User) error
}

func (m *mockUserRepository) Get(ctx context.Context, id int64) (*model.User, error) {
	return m.getFunc(ctx, id)
}

func (m *mockUserRepository) Insert(ctx context.Context, user *model.User) error {
	return m.insertFunc(ctx, user)
}

type mockIdempotencyRecordRepository struct{}

func (m *mockIdempotencyRecordRepository) Get(ctx context.Context, id int64) (*idempotency.Record, error) {
	return nil, nil
}

func (m *mockIdempotencyRecordRepository) Insert(ctx context.Context, record *idempotency.Record) error {
	return nil
}

type mockUnitOfWork struct {
	userRepo        repository.UserRepository
	ledgerRepo      repository.LedgerRepository
	idempotencyRepo idempotency.RecordRepository
	commitFunc      func(ctx context.Context) error
	abortFunc       func(ctx context.Context) error
}

func (m *mockUnitOfWork) UserRepository() repository.UserRepository           { return m.userRepo }
func (m *mockUnitOfWork) LedgerRepository() repository.LedgerRepository       { return m.ledgerRepo }
func (m *mockUnitOfWork) IdempotencyRecordRepository() idempotency.RecordRepository {
	return m.idempotencyRepo
}
func (m *mockUnitOfWork) Commit(ctx context.Context) error { return m.commitFunc(ctx) }
func (m *mockUnitOfWork) Abort(ctx context.Context) error  { return m.abortFunc(ctx) }

type mockUnitOfWorkFactory struct {
	newFunc func() (repository.UnitOfWork, error)
}

func (m *mockUnitOfWorkFactory) New() (repository.UnitOfWork, error) { return m.newFunc() }

type mockIdempotency struct {
	executeFunc func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error)
}

func (m *mockIdempotency) Execute(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
	return m.executeFunc(ctx, repo, id, requestType, referenceId, newResult, fn)
}

// --- tests ---

func TestUserService_GetUser(t *testing.T) {
	ctx := context.Background()

	t.Run("returns user successfully", func(t *testing.T) {
		expected := &model.User{Id: 1, Email: "a@b.com", Username: "alice"}
		committed := false

		uow := &mockUnitOfWork{
			userRepo: &mockUserRepository{
				getFunc: func(ctx context.Context, id int64) (*model.User, error) {
					return expected, nil
				},
			},
			commitFunc: func(ctx context.Context) error { committed = true; return nil },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			nil, nil,
		)

		user, err := svc.GetUser(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, expected, user)
		assert.True(t, committed)
	})

	t.Run("returns nil when user not found", func(t *testing.T) {
		uow := &mockUnitOfWork{
			userRepo: &mockUserRepository{
				getFunc: func(ctx context.Context, id int64) (*model.User, error) {
					return nil, nil
				},
			},
			commitFunc: func(ctx context.Context) error { return nil },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			nil, nil,
		)

		user, err := svc.GetUser(ctx, 999)
		require.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("uow factory error is propagated", func(t *testing.T) {
		factoryErr := errors.New("cannot begin transaction")

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return nil, factoryErr }},
			nil, nil,
		)

		user, err := svc.GetUser(ctx, 1)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, factoryErr)
	})

	t.Run("repository error aborts and is propagated", func(t *testing.T) {
		repoErr := errors.New("db error")
		aborted := false

		uow := &mockUnitOfWork{
			userRepo: &mockUserRepository{
				getFunc: func(ctx context.Context, id int64) (*model.User, error) {
					return nil, repoErr
				},
			},
			commitFunc: func(ctx context.Context) error { t.Fatal("commit should not be called"); return nil },
			abortFunc:  func(ctx context.Context) error { aborted = true; return nil },
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			nil, nil,
		)

		user, err := svc.GetUser(ctx, 1)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, repoErr)
		assert.True(t, aborted)
	})

	t.Run("commit error is propagated", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		uow := &mockUnitOfWork{
			userRepo: &mockUserRepository{
				getFunc: func(ctx context.Context, id int64) (*model.User, error) {
					return &model.User{Id: 1}, nil
				},
			},
			commitFunc: func(ctx context.Context) error { return commitErr },
			abortFunc:  func(ctx context.Context) error { return nil },
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			nil, nil,
		)

		user, err := svc.GetUser(ctx, 1)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, commitErr)
	})
}

func TestUserService_CreateUser(t *testing.T) {
	ctx := context.Background()
	snowflakeId := int64(12345)

	t.Run("creates user with snowflake ID and timestamps", func(t *testing.T) {
		var insertedUser *model.User
		committed := false

		userRepo := &mockUserRepository{
			insertFunc: func(ctx context.Context, user *model.User) error {
				insertedUser = &model.User{
					Id: user.Id, Email: user.Email, Username: user.Username,
					Password: user.Password, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
				}
				return nil
			},
			getFunc: func(ctx context.Context, id int64) (*model.User, error) {
				return insertedUser, nil
			},
		}

		uow := &mockUnitOfWork{
			userRepo:        userRepo,
			idempotencyRepo: &mockIdempotencyRecordRepository{},
			commitFunc:      func(ctx context.Context) error { committed = true; return nil },
			abortFunc:       func(ctx context.Context) error { return nil },
		}

		// Use a passthrough idempotency that always executes fn (cache miss)
		idem := &mockIdempotency{
			executeFunc: func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
				assert.Equal(t, int64(99), id)
				assert.Equal(t, constant.RequestTypeCreateUser, requestType)
				assert.Equal(t, snowflakeId, referenceId)
				return fn()
			},
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			idem,
			&mockSnowflake{id: snowflakeId},
		)

		before := time.Now().UTC()
		user, err := svc.CreateUser(ctx, 99, &model.User{Email: "a@b.com", Username: "alice", Password: "hash"})
		after := time.Now().UTC()

		require.NoError(t, err)
		assert.Equal(t, snowflakeId, user.Id)
		assert.Equal(t, "a@b.com", user.Email)
		assert.True(t, committed)

		// Verify timestamps are set to approximately now
		assert.False(t, insertedUser.CreatedAt.IsZero())
		assert.False(t, insertedUser.UpdatedAt.IsZero())
		assert.True(t, !insertedUser.CreatedAt.Before(before) && !insertedUser.CreatedAt.After(after))
		assert.Equal(t, insertedUser.CreatedAt, insertedUser.UpdatedAt)
	})

	t.Run("idempotency cache hit returns cached user without insert", func(t *testing.T) {
		cached := &model.User{Id: 100, Email: "cached@test.com"}
		committed := false

		uow := &mockUnitOfWork{
			userRepo: &mockUserRepository{
				insertFunc: func(ctx context.Context, user *model.User) error {
					t.Fatal("insert should not be called on cache hit")
					return nil
				},
			},
			idempotencyRepo: &mockIdempotencyRecordRepository{},
			commitFunc:      func(ctx context.Context) error { committed = true; return nil },
			abortFunc:       func(ctx context.Context) error { return nil },
		}

		idem := &mockIdempotency{
			executeFunc: func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
				return cached, nil // simulate cache hit
			},
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			idem,
			&mockSnowflake{id: snowflakeId},
		)

		user, err := svc.CreateUser(ctx, 99, &model.User{Email: "a@b.com"})
		require.NoError(t, err)
		assert.Equal(t, cached, user)
		assert.True(t, committed)
	})

	t.Run("uow factory error is propagated", func(t *testing.T) {
		factoryErr := errors.New("cannot begin transaction")

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return nil, factoryErr }},
			nil,
			&mockSnowflake{id: snowflakeId},
		)

		user, err := svc.CreateUser(ctx, 99, &model.User{})
		assert.Nil(t, user)
		assert.ErrorIs(t, err, factoryErr)
	})

	t.Run("insert error aborts and is propagated", func(t *testing.T) {
		insertErr := errors.New("insert failed")
		aborted := false

		userRepo := &mockUserRepository{
			insertFunc: func(ctx context.Context, user *model.User) error { return insertErr },
			getFunc:    func(ctx context.Context, id int64) (*model.User, error) { return nil, nil },
		}

		uow := &mockUnitOfWork{
			userRepo:        userRepo,
			idempotencyRepo: &mockIdempotencyRecordRepository{},
			commitFunc:      func(ctx context.Context) error { t.Fatal("commit should not be called"); return nil },
			abortFunc:       func(ctx context.Context) error { aborted = true; return nil },
		}

		idem := &mockIdempotency{
			executeFunc: func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
				return fn()
			},
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			idem,
			&mockSnowflake{id: snowflakeId},
		)

		user, err := svc.CreateUser(ctx, 99, &model.User{})
		assert.Nil(t, user)
		assert.ErrorIs(t, err, insertErr)
		assert.True(t, aborted)
	})

	t.Run("get after insert error aborts and is propagated", func(t *testing.T) {
		getErr := errors.New("get failed after insert")
		aborted := false

		userRepo := &mockUserRepository{
			insertFunc: func(ctx context.Context, user *model.User) error { return nil },
			getFunc:    func(ctx context.Context, id int64) (*model.User, error) { return nil, getErr },
		}

		uow := &mockUnitOfWork{
			userRepo:        userRepo,
			idempotencyRepo: &mockIdempotencyRecordRepository{},
			commitFunc:      func(ctx context.Context) error { t.Fatal("commit should not be called"); return nil },
			abortFunc:       func(ctx context.Context) error { aborted = true; return nil },
		}

		idem := &mockIdempotency{
			executeFunc: func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
				return fn()
			},
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			idem,
			&mockSnowflake{id: snowflakeId},
		)

		user, err := svc.CreateUser(ctx, 99, &model.User{})
		assert.Nil(t, user)
		assert.ErrorIs(t, err, getErr)
		assert.True(t, aborted)
	})

	t.Run("commit error is propagated", func(t *testing.T) {
		commitErr := errors.New("commit failed")

		userRepo := &mockUserRepository{
			insertFunc: func(ctx context.Context, user *model.User) error { return nil },
			getFunc: func(ctx context.Context, id int64) (*model.User, error) {
				return &model.User{Id: id}, nil
			},
		}

		uow := &mockUnitOfWork{
			userRepo:        userRepo,
			idempotencyRepo: &mockIdempotencyRecordRepository{},
			commitFunc:      func(ctx context.Context) error { return commitErr },
			abortFunc:       func(ctx context.Context) error { return nil },
		}

		idem := &mockIdempotency{
			executeFunc: func(ctx context.Context, repo idempotency.RecordRepository, id int64, requestType constant.RequestType, referenceId int64, newResult func() any, fn func() (any, error)) (any, error) {
				return fn()
			},
		}

		svc := service.NewUserService(
			&mockUnitOfWorkFactory{newFunc: func() (repository.UnitOfWork, error) { return uow, nil }},
			idem,
			&mockSnowflake{id: snowflakeId},
		)

		user, err := svc.CreateUser(ctx, 99, &model.User{})
		assert.Nil(t, user)
		assert.ErrorIs(t, err, commitErr)
	})
}
