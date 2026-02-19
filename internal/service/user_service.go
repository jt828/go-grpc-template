package service

import (
	"context"
	"time"

	"github.com/jt828/go-grpc-template/internal/constant"
	"github.com/jt828/go-grpc-template/internal/repository"
	"github.com/jt828/go-grpc-template/pkg/idempotency"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/pkg/snowflake"
)

type UserService interface {
	GetUser(ctx context.Context, id int64) (*model.User, error)
	CreateUser(ctx context.Context, idempotencyId int64, user *model.User) (*model.User, error)
}

type userService struct {
	uowFactory  repository.UnitOfWorkFactory
	idempotency idempotency.Idempotency
	snowflake   snowflake.Snowflake
}

func NewUserService(uowFactory repository.UnitOfWorkFactory, idempotency idempotency.Idempotency, snowflake snowflake.Snowflake) UserService {
	return &userService{uowFactory: uowFactory, idempotency: idempotency, snowflake: snowflake}
}

func (s *userService) GetUser(ctx context.Context, id int64) (*model.User, error) {
	uow, err := s.uowFactory.New()
	if err != nil {
		return nil, err
	}

	user, err := uow.UserRepository().Get(ctx, id)
	if err != nil {
		_ = uow.Abort(ctx)
		return nil, err
	}

	if err := uow.Commit(ctx); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) CreateUser(ctx context.Context, idempotencyId int64, user *model.User) (*model.User, error) {
	uow, err := s.uowFactory.New()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user.Id = s.snowflake.Generate()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := s.idempotency.Execute(ctx, uow.IdempotencyRecordRepository(), idempotencyId, constant.RequestTypeCreateUser, user.Id, func() any { return &model.User{} }, func() (any, error) {
		if err := uow.UserRepository().Insert(ctx, user); err != nil {
			return nil, err
		}

		createdUser, err := uow.UserRepository().Get(ctx, user.Id)
		if err != nil {
			return nil, err
		}
		return createdUser, nil
	})
	if err != nil {
		_ = uow.Abort(ctx)
		return nil, err
	}

	if err := uow.Commit(ctx); err != nil {
		return nil, err
	}

	return result.(*model.User), nil
}
