package users

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

var ErrOwnsVehicles = errors.New("user owns vehicles")

type UserRepository interface {
	FindByID(context.Context, uuid.UUID) (*User, error)
	FindByIDIncludingDeleted(context.Context, uuid.UUID) (*User, error)
	UpdateByID(context.Context, uuid.UUID, int64, *User) error
	CanRead(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	DeleteAccount(context.Context, uuid.UUID) error
}

type Repository struct {
	database *database.Database
}

func NewUserRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return helpers.FindByID[User](ctx, r.database.GetGorm(), id)
}

func (r *Repository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	if err := r.database.GetGorm().WithContext(ctx).Unscoped().Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeleteAccount(ctx context.Context, id uuid.UUID) error {
	return r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owned int64
		if err := tx.Table("vehicles").Where("created_by = ? AND deleted_at IS NULL", id).Count(&owned).Error; err != nil {
			return err
		}
		if owned > 0 {
			return ErrOwnsVehicles
		}

		result := tx.Unscoped().Model(&User{}).
			Where("id = ? AND deleted_at IS NULL", id).
			Updates(map[string]any{
				"username":   DeletedUsername,
				"email":      nil,
				"role":       RoleUser,
				"deleted_at": time.Now(),
				"version":    gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Exec("DELETE FROM identities WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM refresh_tokens WHERE user_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, expectedVersion int64, user *User) error {
	return helpers.UpdateByIDVersioned[User](ctx, r.database.GetGorm(), id, expectedVersion, user)
}

func (r *Repository) CanRead(ctx context.Context, currentUser uuid.UUID, targetUser uuid.UUID) (bool, error) {
	if currentUser == targetUser {
		return true, nil
	}

	type Result struct {
		CanRead bool `json:"can_read"`
	}
	var result Result

	err := gorm.G[User](r.database.GetGorm()).Raw(`
	SELECT true AS can_read
	FROM vehicles V
	LEFT JOIN vehicle_user_access A ON V.id = A.vehicle_id AND A.deleted_at IS NULL
	WHERE V.deleted_at IS NULL
		AND (
			( V.created_by = ? AND A.user_id = ? )
			OR
			(
				V.id IN (
					SELECT V_INNER.id
					FROM vehicles V_INNER
					LEFT JOIN vehicle_user_access A_INNER ON V_INNER.id = A_INNER.vehicle_id AND A_INNER.deleted_at IS NULL
					WHERE V_INNER.deleted_at IS NULL
						AND ( A_INNER.user_id = ? OR V_INNER.created_by = ? )
				)
				AND
				(
					( A.user_id = ? AND A.write_access = true )
					OR
					V.created_by = ?
				)
			)
		)
	LIMIT 1`, currentUser, targetUser, currentUser, currentUser, targetUser, targetUser).Scan(ctx, &result)
	if err != nil {
		return false, err
	}
	return result.CanRead, nil
}
