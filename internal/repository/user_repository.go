package repository

import (
	"context"

	"errors"

	"github.com/jt828/go-grpc-template/pkg/circuitbreaker"
	"github.com/jt828/go-grpc-template/pkg/model"
	"github.com/jt828/go-grpc-template/pkg/retry"
	"gorm.io/gorm"
)

type UserRepository interface {
	Get(ctx context.Context, id int64) (*model.User, error)
	Insert(ctx context.Context, user *model.User) error
}

type UserRepositoryImpl struct {
	db              *gorm.DB
	cb              circuitbreaker.CircuitBreaker
	retry           retry.Retry
	notFoundAsError bool
}

func NewUserRepository(db *gorm.DB, cb circuitbreaker.CircuitBreaker, retry retry.Retry, notFoundAsError bool) UserRepository {
	return &UserRepositoryImpl{db: db, cb: cb, retry: retry, notFoundAsError: notFoundAsError}
}

func (r *UserRepositoryImpl) Get(ctx context.Context, id int64) (*model.User, error) {
	result, err := r.cb.Execute(func() (any, error) {
		var user *model.User
		err := r.retry.Execute(ctx, func() error {
			var entity model.UserDataEntity
			if err := r.db.WithContext(ctx).First(&entity, id).Error; err != nil {
				if !r.notFoundAsError && errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			u := entity.ToDomain()
			user = &u
			return nil
		})
		if err != nil {
			return nil, err
		}
		return user, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*model.User), nil
}

func (r *UserRepositoryImpl) Insert(ctx context.Context, user *model.User) error {
	_, err := r.cb.Execute(func() (any, error) {
		err := r.retry.Execute(ctx, func() error {
			entity := model.UserDataEntity{
				Id:        user.Id,
				Email:     user.Email,
				Username:  user.Username,
				Password:  user.Password,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			}
			return r.db.WithContext(ctx).Create(&entity).Error
		})
		return nil, err
	})
	return err
}
