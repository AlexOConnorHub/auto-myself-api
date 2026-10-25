package users

import (
	"auto-myself-api/internal/helpers"
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
)

type UserService interface {
	UpdateUser(context.Context, UserPost) error
	GetUserByID(context.Context, uuid.UUID) (*User, error)
}

type userService struct {
	userRepository *userRepository
}

func NewUserService(userRepository *userRepository) *userService {
	return &userService{
		userRepository: userRepository,
	}
}

func (s *userService) GetUserByID(ctx context.Context, userUUID uuid.UUID) (*User, error) {
	currentUser := ctx.Value(helpers.UserCtxKey).(*User)

	if canRead, err := s.userRepository.CanRead(ctx, currentUser.ID, userUUID); err != nil {
		return nil, fmt.Errorf("userService.GetUserByID: %w", err)
	} else if !canRead {
		return nil, helpers.ErrAccessDenied
	}

	if user, err := s.userRepository.FindUserByID(ctx, userUUID); err != nil {
		return nil, fmt.Errorf("userService.GetUserByID: %w", err)
	} else {
		return user, nil
	}
}

func (s *userService) UpdateUser(ctx context.Context, input UserPost) error {
	user := ctx.Value(helpers.UserCtxKey).(*User)
	return s.userRepository.UpdateUser(ctx, user.ID, input)
}
