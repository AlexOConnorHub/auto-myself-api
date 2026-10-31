package maintenance

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type MaintenanceRepository interface {
	FindIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	FindForVehicle(context.Context, uuid.UUID) ([]MaintenanceRecord, error)
	FindByID(context.Context, uuid.UUID) (*MaintenanceRecord, error)
	UpdateByID(context.Context, uuid.UUID, int64, *MaintenanceRecord) error
	DeleteByID(context.Context, uuid.UUID) error
	Create(context.Context, *MaintenanceRecord) error
}

type Repository struct {
	database *database.Database
}

func NewMaintenanceRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindIDs(ctx context.Context, vehicleID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := gorm.G[MaintenanceRecord](r.database.GetGorm()).Select("id").Where("vehicle_id = ?", vehicleID).Find(ctx)
	if err != nil {
		return nil, err
	}

	idsUUID := make([]uuid.UUID, len(ids))
	for i, id := range ids {
		idsUUID[i] = id.ID
	}

	return idsUUID, nil
}

func (r *Repository) FindForVehicle(ctx context.Context, vehicleID uuid.UUID) ([]MaintenanceRecord, error) {
	return gorm.G[MaintenanceRecord](r.database.GetGorm()).
		Where("vehicle_id = ?", vehicleID).
		Order(`"timestamp" ASC, id ASC`).
		Find(ctx)
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*MaintenanceRecord, error) {
	return helpers.FindByID[MaintenanceRecord](ctx, r.database.GetGorm(), id)
}

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, expectedVersion int64, record *MaintenanceRecord) error {
	return helpers.UpdateByIDVersioned[MaintenanceRecord](ctx, r.database.GetGorm(), id, expectedVersion, record)
}

func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return helpers.DeleteByID[MaintenanceRecord](ctx, r.database.GetGorm(), id)
}

func (r *Repository) Create(ctx context.Context, record *MaintenanceRecord) error {
	return helpers.Create[MaintenanceRecord](ctx, r.database.GetGorm(), record)
}
