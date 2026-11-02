package shares

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ShareRepository interface {
	FindByID(context.Context, uuid.UUID) (*VehicleUserAccessPending, error)
	FindIDsForUser(context.Context, uuid.UUID) ([]uuid.UUID, error)
	Create(context.Context, *VehicleUserAccessPending) error
	DeleteByID(context.Context, uuid.UUID) error
	Accept(context.Context, *VehicleUserAccessPending) (*VehicleUserAccess, error)
	ExistsForVehicleAndUser(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	FindAccessForVehicle(context.Context, uuid.UUID) ([]VehicleUserAccess, error)
	FindAccessForVehicleAndUser(context.Context, uuid.UUID, uuid.UUID) (*VehicleUserAccess, error)
	FindAccessByID(context.Context, uuid.UUID) (*VehicleUserAccess, error)
	UpdateAccessByID(context.Context, uuid.UUID, int64, *VehicleUserAccess) error
	DeleteAccessByID(context.Context, uuid.UUID) error
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

func (r *Repository) FindAccessForVehicle(ctx context.Context, vehicleID uuid.UUID) ([]VehicleUserAccess, error) {
	return gorm.G[VehicleUserAccess](r.database.GetGorm()).
		Where("vehicle_id = ?", vehicleID).
		Order("created_at ASC, id ASC").
		Find(ctx)
}

func (r *Repository) FindAccessForVehicleAndUser(ctx context.Context, vehicleID, userID uuid.UUID) (*VehicleUserAccess, error) {
	access, err := gorm.G[VehicleUserAccess](r.database.GetGorm()).
		Where("vehicle_id = ? AND user_id = ?", vehicleID, userID).
		Order("created_at ASC, id ASC").
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &access, nil
}

func (r *Repository) FindAccessByID(ctx context.Context, id uuid.UUID) (*VehicleUserAccess, error) {
	return helpers.FindByID[VehicleUserAccess](ctx, r.database.GetGorm(), id)
}

func (r *Repository) UpdateAccessByID(ctx context.Context, id uuid.UUID, expectedVersion int64, access *VehicleUserAccess) error {
	db := r.database.GetGorm().WithContext(ctx)

	var persisted VehicleUserAccess
	result := db.Model(&persisted).
		Clauses(clause.Returning{}).
		Where("id = ? AND version = ?", id, expectedVersion).
		Updates(map[string]any{
			"write_access": access.WriteAccess,
			"version":      gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return helpers.VersionConflictFor[VehicleUserAccess](db, id)
	}

	*access = persisted
	return nil
}

func (r *Repository) DeleteAccessByID(ctx context.Context, id uuid.UUID) error {
	return helpers.DeleteByID[VehicleUserAccess](ctx, r.database.GetGorm(), id)
}

// Accept hard-deletes the pending share
func (r *Repository) Accept(ctx context.Context, share *VehicleUserAccessPending) (*VehicleUserAccess, error) {
	var access *VehicleUserAccess

	err := r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		access = &VehicleUserAccess{
			VehicleUserAccessBase: share.VehicleUserAccessBase,
			CreatedBy:             share.CreatedBy,
		}
		if err := gorm.G[VehicleUserAccess](tx).Create(ctx, access); err != nil {
			return err
		}

		result := tx.Unscoped().Where("id = ? AND deleted_at IS NULL", share.ID).Delete(&VehicleUserAccessPending{})
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
	var count int64
	err := gorm.G[VehicleUserAccess](r.database.GetGorm()).Raw(`
SELECT
	(SELECT count(*) FROM vehicle_user_access
		WHERE vehicle_id = $1 AND user_id = $2 AND deleted_at IS NULL)
	+
	(SELECT count(*) FROM vehicle_user_access_pending
		WHERE vehicle_id = $1 AND user_id = $2 AND deleted_at IS NULL)`, vehicleID, userID).Scan(ctx, &count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
