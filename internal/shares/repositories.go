package shares

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type ShareRepository interface {
	FindByID(context.Context, uuid.UUID) (*VehicleUserAccessPending, error)
	FindIDsForUser(context.Context, uuid.UUID) ([]uuid.UUID, error)
	Create(context.Context, *VehicleUserAccessPending) error
	DeleteByID(context.Context, uuid.UUID) error
	Accept(context.Context, *VehicleUserAccessPending) (*VehicleUserAccess, error)
	ExistsForVehicleAndUser(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type Repository struct {
	database *database.Database
}

func NewShareRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*VehicleUserAccessPending, error) {
	return helpers.FindByID[VehicleUserAccessPending](ctx, r.database.GetGorm(), id)
}

func (r *Repository) FindIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	shares, err := gorm.G[VehicleUserAccessPending](r.database.GetGorm()).
		Select("id").
		Where("user_id = ? OR created_by = ?", userID, userID).
		Order("created_at ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(shares))
	for i, share := range shares {
		ids[i] = share.ID
	}
	return ids, nil
}

func (r *Repository) Create(ctx context.Context, share *VehicleUserAccessPending) error {
	return helpers.Create[VehicleUserAccessPending](ctx, r.database.GetGorm(), share)
}

func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return helpers.DeleteByID[VehicleUserAccessPending](ctx, r.database.GetGorm(), id)
}

// Accept hard-deletes the pending share
func (r *Repository) Accept(ctx context.Context, share *VehicleUserAccessPending) (*VehicleUserAccess, error) {
	var access *VehicleUserAccess

	err := r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access = &VehicleUserAccess{
			VehicleUserAccessBase: share.VehicleUserAccessBase,
			CreatedBy:             share.UserID,
		}
		if err := gorm.G[VehicleUserAccess](tx).Create(ctx, access); err != nil {
			return err
		}

		result := tx.Unscoped().Where("id = ?", share.ID).Delete(&VehicleUserAccessPending{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return helpers.ErrRowCountOff
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return access, nil
}

func (r *Repository) ExistsForVehicleAndUser(ctx context.Context, vehicleID, userID uuid.UUID) (bool, error) {
	accessCount, err := gorm.G[VehicleUserAccess](r.database.GetGorm()).
		Where("vehicle_id = ? AND user_id = ?", vehicleID, userID).
		Count(ctx, "*")
	if err != nil {
		return false, err
	}
	if accessCount > 0 {
		return true, nil
	}

	pendingCount, err := gorm.G[VehicleUserAccessPending](r.database.GetGorm()).
		Where("vehicle_id = ? AND user_id = ?", vehicleID, userID).
		Count(ctx, "*")
	if err != nil {
		return false, err
	}
	return pendingCount > 0, nil
}
