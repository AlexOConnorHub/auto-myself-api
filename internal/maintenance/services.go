package maintenance

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

	if _, err := s.vehicleRepository.FindByID(ctx, vehicleUUID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding vehicle %s: %w", vehicleUUID, err)
	}

	canRead, err := s.vehicleRepository.CanRead(ctx, vehicleUUID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("checking read access to vehicle %s: %w", vehicleUUID, err)
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}

	ids, err := s.maintenanceRepository.FindIDs(ctx, vehicleUUID)
	if err != nil {
		return nil, fmt.Errorf("finding maintenance records for vehicle %s: %w", vehicleUUID, err)
	}
	return ids, nil
}

func (s *Service) FindByID(ctx context.Context, maintenanceUUID uuid.UUID) (*MaintenanceRecord, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	record, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding maintenance record %s: %w", maintenanceUUID, err)
	}

	canRead, err := s.vehicleRepository.CanRead(ctx, record.VehicleID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("checking read access to vehicle %s: %w", record.VehicleID, err)
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}

	return record, nil
}

// A maintenance record never moves between vehicles, so VehicleID is always the stored one.
func (s *Service) UpdateByID(ctx context.Context, maintenanceUUID uuid.UUID, record *MaintenanceRecord) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	existing, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding maintenance record %s: %w", maintenanceUUID, err)
	}

	if err := s.authorizeWrite(ctx, existing.VehicleID, user.ID); err != nil {
		return err
	}

	record.VehicleID = existing.VehicleID

	if err := s.maintenanceRepository.UpdateByID(ctx, maintenanceUUID, record); err != nil {
		return fmt.Errorf("updating maintenance record %s: %w", maintenanceUUID, err)
	}
	return nil
}

func (s *Service) DeleteByID(ctx context.Context, maintenanceUUID uuid.UUID) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	record, err := s.maintenanceRepository.FindByID(ctx, maintenanceUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding maintenance record %s: %w", maintenanceUUID, err)
	}

	if err := s.authorizeWrite(ctx, record.VehicleID, user.ID); err != nil {
		return err
	}

	if err := s.maintenanceRepository.DeleteByID(ctx, maintenanceUUID); err != nil {
		return fmt.Errorf("deleting maintenance record %s: %w", maintenanceUUID, err)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, record *MaintenanceRecord) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	if _, err := s.vehicleRepository.FindByID(ctx, record.VehicleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding vehicle %s: %w", record.VehicleID, err)
	}

	if err := s.authorizeWrite(ctx, record.VehicleID, user.ID); err != nil {
		return err
	}

	record.CreatedBy = user.ID

	if err := s.maintenanceRepository.Create(ctx, record); err != nil {
		return fmt.Errorf("creating maintenance record: %w", err)
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
