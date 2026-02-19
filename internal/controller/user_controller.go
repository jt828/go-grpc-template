package controller

import (
	"context"
	"fmt"

	"github.com/jt828/go-grpc-template/internal/service"
	"github.com/jt828/go-grpc-template/pkg/apperror"
	"github.com/jt828/go-grpc-template/pkg/model"
	v1 "github.com/jt828/go-grpc-template/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserController struct {
	v1.UnimplementedUserServiceServer
	userService service.UserService
}

func NewUserController(userService service.UserService) *UserController {
	return &UserController{userService: userService}
}

func (ctrl *UserController) GetUserById(
	ctx context.Context,
	request *v1.GetUserByIdRequest,
) (*v1.GetUserByIdResponse, error) {
	if request.Id <= 0 {
		return nil, fmt.Errorf("id must be greater than 0: %w", apperror.ErrInvalidArgument)
	}
	user, err := ctrl.userService.GetUser(ctx, request.Id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user %d: %w", request.Id, apperror.ErrNotFound)
	}

	return &v1.GetUserByIdResponse{
		Id:        user.Id,
		Email:     user.Email,
		Username:  user.Username,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}, nil
}

func (ctrl *UserController) CreateUser(
	ctx context.Context,
	request *v1.CreateUserRequest,
) (*v1.CreateUserResponse, error) {
	if request.IdempotencyId <= 0 {
		return nil, fmt.Errorf("idempotency_id must be greater than 0: %w", apperror.ErrInvalidArgument)
	}
	if request.Email == "" {
		return nil, fmt.Errorf("email is required: %w", apperror.ErrInvalidArgument)
	}
	if request.Username == "" {
		return nil, fmt.Errorf("username is required: %w", apperror.ErrInvalidArgument)
	}
	if request.Password == "" {
		return nil, fmt.Errorf("password is required: %w", apperror.ErrInvalidArgument)
	}

	user := &model.User{
		Email:    request.Email,
		Username: request.Username,
		Password: request.Password,
	}

	createdUser, err := ctrl.userService.CreateUser(ctx, request.IdempotencyId, user)
	if err != nil {
		return nil, err
	}

	return &v1.CreateUserResponse{
		Id:        createdUser.Id,
		Email:     createdUser.Email,
		Username:  createdUser.Username,
		CreatedAt: timestamppb.New(createdUser.CreatedAt),
		UpdatedAt: timestamppb.New(createdUser.UpdatedAt),
	}, nil
}
