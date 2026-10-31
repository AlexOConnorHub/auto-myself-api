package vehicles

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type VehicleService interface {
	FindIDs(context.Context) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Vehicle, error)
	UpdateByID(context.Context, uuid.UUID, *Vehicle) error
	DeleteByID(context.Context, uuid.UUID) error
	Create(context.Context, *Vehicle) error
}

type Service struct {
	vehicleRepository *Repository
}

func NewVehicleService(vehicleRepository *Repository) *Service {
	return &Service{
		vehicleRepository: vehicleRepository,
	}
}

func (s *Service) FindIDs(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	ids, err := s.vehicleRepository.FindIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("finding vehicles for user %s: %w", user.ID, err)
	}
	return ids, nil
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	vehicle, err := s.vehicleRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding vehicle %s: %w", id, err)
	}

	canRead, err := s.vehicleRepository.CanRead(ctx, id, user.ID)
	if err != nil {
		return nil, fmt.Errorf("checking read access to vehicle %s: %w", id, err)
	}
	if !canRead {
		// A pending share lets the recipient view the vehicle before accepting it.
		canReadPending, pendingErr := s.vehicleRepository.CanReadPending(ctx, id, user.ID)
		if pendingErr != nil {
			return nil, fmt.Errorf("checking pending read access to vehicle %s: %w", id, pendingErr)
		}
		if !canReadPending {
			return nil, gorm.ErrRecordNotFound
		}
	}

	return vehicle, nil
}

func (s *Service) UpdateByID(ctx context.Context, id uuid.UUID, vehicle *Vehicle) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	if _, err := s.vehicleRepository.FindByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding vehicle %s: %w", id, err)
	}

	if err := s.authorizeWrite(ctx, id, user.ID); err != nil {
		return err
	}

	if err := s.vehicleRepository.UpdateByID(ctx, id, vehicle); err != nil {
		return fmt.Errorf("updating vehicle %s: %w", id, err)
	}
	return nil
}

func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	vehicle, err := s.vehicleRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding vehicle %s: %w", id, err)
	}

	if vehicle.CreatedBy != user.ID {
		canRead, readErr := s.vehicleRepository.CanRead(ctx, id, user.ID)
		if readErr != nil {
			return fmt.Errorf("checking read access to vehicle %s: %w", id, readErr)
		}
		if canRead {
			return helpers.ErrAccessDenied
		}
		return gorm.ErrRecordNotFound
	}

	if err := s.vehicleRepository.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("deleting vehicle %s: %w", id, err)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, vehicle *Vehicle) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	vehicle.CreatedBy = user.ID

	if err := s.vehicleRepository.Create(ctx, vehicle); err != nil {
		return fmt.Errorf("creating vehicle: %w", err)
	}
	return nil
}

func (s *Service) authorizeWrite(ctx context.Context, vehicleID, userID uuid.UUID) error {
	canRead, err := s.vehicleRepository.CanRead(ctx, vehicleID, userID)
	if err != nil {
		return fmt.Errorf("checking read access to vehicle %s: %w", vehicleID, err)
	}
	canWrite, err := s.vehicleRepository.CanWrite(ctx, vehicleID, userID)
	if err != nil {
		return fmt.Errorf("checking write access to vehicle %s: %w", vehicleID, err)
	}
	switch {
	case canWrite:
		return nil
	case canRead:
		return helpers.ErrAccessDenied
	default:
		return gorm.ErrRecordNotFound
	}
}
