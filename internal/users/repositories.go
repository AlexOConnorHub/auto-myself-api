package users

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindByID(context.Context, uuid.UUID) (*User, error)
	UpdateByID(context.Context, uuid.UUID, *User) error
	CanRead(context.Context, uuid.UUID, uuid.UUID) (bool, error)
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

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, user *User) error {
	return helpers.UpdateByID[User](ctx, r.database.GetGorm(), id, user)
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
