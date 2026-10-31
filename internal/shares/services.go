package shares

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

var (
	ErrShareAlreadyExists  = errors.New("share already exists")
	ErrCannotShareWithSelf = errors.New("cannot share a vehicle with yourself")
)

type ShareService interface {
	GetAllShares(context.Context) ([]uuid.UUID, error)
	GetShareByID(context.Context, uuid.UUID) (*VehicleUserAccessPending, error)
	CreateShare(context.Context, *VehicleUserAccessPending) error
	AcceptShare(context.Context, uuid.UUID) (*VehicleUserAccess, error)
	DeleteShareByID(context.Context, uuid.UUID) error
}

type Service struct {
	shareRepository   *Repository
	vehicleRepository *vehicles.Repository
	userRepository    *users.Repository
}

func NewShareService(
	shareRepository *Repository,
	vehicleRepository *vehicles.Repository,
	userRepository *users.Repository,
) *Service {
	return &Service{
		shareRepository:   shareRepository,
		vehicleRepository: vehicleRepository,
		userRepository:    userRepository,
	}
}

func (s *Service) GetAllShares(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	ids, err := s.shareRepository.FindIDsForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("finding shares for user %s: %w", user.ID, err)
	}
	return ids, nil
}

func (s *Service) GetShareByID(ctx context.Context, id uuid.UUID) (*VehicleUserAccessPending, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	share, err := s.shareRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding share %s: %w", id, err)
	}

	if share.UserID != user.ID && share.CreatedBy != user.ID {
		return nil, gorm.ErrRecordNotFound
	}

	return share, nil
}

func (s *Service) CreateShare(ctx context.Context, share *VehicleUserAccessPending) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	vehicle, err := s.vehicleRepository.FindByID(ctx, share.VehicleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding vehicle %s: %w", share.VehicleID, err)
	}

	if vehicle.CreatedBy != user.ID {
		canRead, readErr := s.vehicleRepository.CanRead(ctx, share.VehicleID, user.ID)
		if readErr != nil {
			return fmt.Errorf("checking read access to vehicle %s: %w", share.VehicleID, readErr)
		}
		if canRead {
			return helpers.ErrAccessDenied
		}
		return gorm.ErrRecordNotFound
	}

	if share.UserID == user.ID {
		return ErrCannotShareWithSelf
	}

	exists, err := s.shareRepository.ExistsForVehicleAndUser(ctx, share.VehicleID, share.UserID)
	if err != nil {
		return fmt.Errorf("checking existing access for vehicle %s and user %s: %w", share.VehicleID, share.UserID, err)
	}
	if exists {
		return ErrShareAlreadyExists
	}

	share.CreatedBy = user.ID
	if err := s.shareRepository.Create(ctx, share); err != nil {
		return fmt.Errorf("creating share: %w", err)
	}
	return nil
}

func (s *Service) AcceptShare(ctx context.Context, id uuid.UUID) (*VehicleUserAccess, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	share, err := s.shareRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding share %s: %w", id, err)
	}

	if share.UserID != user.ID {
		if share.CreatedBy == user.ID {
			return nil, helpers.ErrAccessDenied
		}
		return nil, gorm.ErrRecordNotFound
	}

	access, err := s.shareRepository.Accept(ctx, share)
	if err != nil {
		return nil, fmt.Errorf("accepting share %s: %w", id, err)
	}
	return access, nil
}

func (s *Service) DeleteShareByID(ctx context.Context, id uuid.UUID) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	share, err := s.shareRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding share %s: %w", id, err)
	}

	if share.UserID != user.ID && share.CreatedBy != user.ID {
		return gorm.ErrRecordNotFound
	}

	if err := s.shareRepository.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("deleting share %s: %w", id, err)
	}
	return nil
}
