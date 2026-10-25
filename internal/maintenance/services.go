package maintenance

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type MaintenanceService interface {
	FindIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*MaintenanceRecord, error)
	UpdateByID(context.Context, uuid.UUID, *MaintenanceRecord) error
	DeleteByID(context.Context, uuid.UUID) error
	Create(context.Context, *MaintenanceRecord) error
}

type Service struct {
	maintenanceRepository *Repository
	vehicleRepository     *vehicles.Repository
}

func NewMaintenanceService(maintenanceRepository *Repository, vehicleRepository *vehicles.Repository) *Service {
	return &Service{
		maintenanceRepository: maintenanceRepository,
		vehicleRepository:     vehicleRepository,
	}
}

func (s *Service) FindIDs(ctx context.Context, vehicleUUID uuid.UUID) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	can_read, err := s.vehicleRepository.CanRead(ctx, vehicleUUID, user.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to check read permission for vehicle %s and user %s: %w", vehicleUUID, user.ID, err)
	}
	if !can_read {
		return nil, helpers.ErrAccessDenied
	}

	return s.maintenanceRepository.FindIDs(ctx, vehicleUUID)
}

func (s *Service) FindByID(ctx context.Context, maintenanceUUID uuid.UUID) (*MaintenanceRecord, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	maintenance, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		} else {
			return nil, fmt.Errorf("Failed to get maintenance record %s: %w", maintenanceUUID, err)
		}
	}

	can_read, err := s.vehicleRepository.CanRead(ctx, maintenance.VehicleID, user.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to check read permission for vehicle %s and user %s: %w", maintenance.VehicleID, user.ID, err)
	}
	if !can_read {
		return nil, helpers.ErrAccessDenied
	}

	return maintenance, nil
}

func (s *Service) UpdateByID(ctx context.Context, maintenanceUUID uuid.UUID, record *MaintenanceRecord) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	maintenance, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		} else {
			return fmt.Errorf("Failed to get maintenance record %s: %w", maintenanceUUID, err)
		}
	}

	can_read, err := s.vehicleRepository.CanRead(ctx, maintenance.VehicleID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check read permission for vehicle %s and user %s: %w", maintenance.VehicleID, user.ID, err)
	}
	can_write, err := s.vehicleRepository.CanWrite(ctx, maintenance.VehicleID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check write permission for vehicle %s and user %s: %w", maintenance.VehicleID, user.ID, err)
	}
	if !can_write {
		if can_read {
			return helpers.ErrAccessDenied
		} else {
			return gorm.ErrRecordNotFound
		}
	}

	return s.maintenanceRepository.UpdateByID(ctx, maintenanceUUID, record)
}

func (s *Service) DeleteByID(ctx context.Context, maintenanceUUID uuid.UUID) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	maintenance, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return err
		} else {
			return fmt.Errorf("Failed to get maintenance record %s: %w", maintenanceUUID, err)
		}
	}

	can_read, err := s.vehicleRepository.CanRead(ctx, maintenance.VehicleID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check read permission for vehicle %s and user %s: %w", maintenance.VehicleID, user.ID, err)
	}
	can_write, err := s.vehicleRepository.CanWrite(ctx, maintenance.VehicleID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check write permission for vehicle %s and user %s: %w", maintenance.VehicleID, user.ID, err)
	}
	if !can_write {
		if can_read {
			return helpers.ErrAccessDenied
		} else {
			return gorm.ErrRecordNotFound
		}
	}

	return s.maintenanceRepository.DeleteByID(ctx, maintenanceUUID)
}

func (s *Service) Create(ctx context.Context, record *MaintenanceRecord) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	record.MaintenanceRecordWeb.CreatedBy = user.ID
	return s.maintenanceRepository.Create(ctx, record)
}
