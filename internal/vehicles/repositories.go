package vehicles

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type VehicleRepository interface {
	FindIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Vehicle, error)
	UpdateByID(context.Context, uuid.UUID, *Vehicle) error
	DeleteByID(context.Context, uuid.UUID) error
	Create(context.Context, *Vehicle) error
	CanRead(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	// CanReadPending(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	CanWrite(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type Repository struct {
	database *database.Database
}

func NewVehicleRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := gorm.G[Vehicle](r.database.GetGorm()).Raw(`
	SELECT id
	FROM vehicles
	WHERE created_by = $1
	UNION
	SELECT vehicle_id
	FROM vehicle_user_access
	WHERE user_id = $1`, userID).Scan(ctx, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	return helpers.FindByID[Vehicle](ctx, r.database.GetGorm(), id)
}

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, vehicle *Vehicle) error {
	return helpers.UpdateByID[Vehicle](ctx, r.database.GetGorm(), id, vehicle)
}

func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	return helpers.DeleteByID[Vehicle](ctx, r.database.GetGorm(), id)
}
func (r *Repository) Create(ctx context.Context, vehicle *Vehicle) error {
	return helpers.Create[Vehicle](ctx, r.database.GetGorm(), vehicle)
}

func (r *Repository) CanRead(ctx context.Context, vehicleID uuid.UUID, userID uuid.UUID) (bool, error) {
	type Result struct {
		CanRead bool
	}
	var result Result
	err := gorm.G[Vehicle](r.database.GetGorm()).Raw(`
	SELECT true AS can_read
	FROM vehicles V
	WHERE V.id = $1 AND V.created_by = $2
	UNION
	SELECT true AS can_read
	FROM vehicle_user_access A
	WHERE A.vehicle_id = $1 AND A.user_id = $2
	LIMIT 1`, vehicleID, userID).Scan(ctx, &result)
	if err != nil {
		return false, err
	}

	return result.CanRead, nil
}

// func (v *Vehicle) CanPendingRead(a *server.App, user *User) bool {
// 	var pendingShare VehicleUserAccessPending
// 	if err := a.Gorm.First(&pendingShare, "vehicle_id = ? AND user_id = ?", v.ID, user.ID).Error; err != nil {
// 		if err != gorm.ErrRecordNotFound {
// 			fmt.Printf("Database error: %v\n", err)
// 		}
// 		return false
// 	}

// 	return pendingShare.ID != uuid.Nil
// }

func (r *Repository) CanWrite(ctx context.Context, vehicleID uuid.UUID, userID uuid.UUID) (bool, error) {
	// if v.CreatedBy == user.ID {
	// 	return true
	// }

	type Result struct {
		WriteAccess bool
	}
	var result Result
	err := gorm.G[Vehicle](r.database.GetGorm()).Raw(`
	SELECT true AS write_access
	FROM vehicles V
	WHERE V.id = $1 AND V.created_by = $2
	UNION
	SELECT
		A.write_access
	FROM vehicle_user_access A
	WHERE A.vehicle_id = $1
		AND A.user_id = $2
	LIMIT 1`, vehicleID, userID).Scan(ctx, &result)
	if err != nil {
		return false, err
	}
	return result.WriteAccess, nil
}
