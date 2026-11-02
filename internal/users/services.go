package users

import (
	"auto-myself-api/internal/helpers"
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type UserService interface {
	UpdateUser(context.Context, *User) error
	GetUserByID(context.Context, uuid.UUID) (*User, error)
	DeleteAccount(context.Context) error
}

type Service struct {
	userRepository *Repository
}

func NewUserService(userRepository *Repository) *Service {
	return &Service{
		userRepository: userRepository,
	}
}

func (s *Service) GetUserByID(ctx context.Context, userUUID uuid.UUID) (*User, error) {
	currentUser := ctx.Value(helpers.UserCtxKey).(*User)

	canRead, err := s.userRepository.CanRead(ctx, currentUser.ID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("checking read access to user %s: %w", userUUID, err)
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}

	requestedUser, err := s.userRepository.FindByIDIncludingDeleted(ctx, userUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding user %s: %w", userUUID, err)
	}

	if requestedUser.DeletedAt.Valid {
		requestedUser.Username = DeletedUsername
		requestedUser.Role = RoleUser
	}

	return requestedUser, nil
}

func (s *Service) DeleteAccount(ctx context.Context) error {
	currentUser := ctx.Value(helpers.UserCtxKey).(*User)

	err := s.userRepository.DeleteAccount(ctx, currentUser.ID)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrOwnsVehicles):
		return err
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil
	default:
		return fmt.Errorf("deleting account %s: %w", currentUser.ID, err)
	}
}

// UpdateUser applies user to the current user's row and replaces user with the persisted row.
// The caller decides which fields are set; the HTTP handler fills only the Post section.
func (s *Service) UpdateUser(ctx context.Context, user *User) error {
	currentUser := ctx.Value(helpers.UserCtxKey).(*User)

	if err := s.userRepository.UpdateByID(ctx, currentUser.ID, user.Version, user); err != nil {
		if errors.Is(err, helpers.ErrVersionConflict) {
			return err
		}
		return fmt.Errorf("updating user %s: %w", currentUser.ID, err)
	}
	return nil
}
