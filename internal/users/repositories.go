package users

import (
	"auto-myself-api/internal/database"
	"context"
	"errors"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindUserByID(context.Context, uuid.UUID) (*User, error)
	UpdateUser(context.Context, uuid.UUID, UserPost) error
	CanRead(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type userRepository struct {
	database *database.Database
}

func NewUserRepository(database *database.Database) *userRepository {
	return &userRepository{
		database: database,
	}
}

func (r *userRepository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := gorm.G[User](r.database.GetGorm()).Where("id = ?", id.String()).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, id uuid.UUID, input UserPost) error {
	_, err := gorm.G[User](r.database.GetGorm()).Where("id = ?", id.String()).Updates(ctx, User{UserWeb: UserWeb{UserPost: input}})
	return err
}

func (r *userRepository) CanRead(ctx context.Context, currentUser uuid.UUID, targetUser uuid.UUID) (bool, error) {
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
	LEFT JOIN vehicle_user_access A ON V.id = A.vehicle_id
	WHERE
		( V.created_by = ? AND A.user_id = ?)
		OR
		(
			V.id IN (
				SELECT V_INNER.id
				FROM vehicles V_INNER
				LEFT JOIN vehicle_user_access A_INNER ON V_INNER.id = A_INNER.vehicle_id
				WHERE A_INNER.user_id = ? OR V_INNER.created_by = ?
			)
			AND
			(
				( A.user_id = ? AND A.write_access = true )
				OR
				V.created_by = ?
			)
		)
	LIMIT 1`, currentUser, targetUser, currentUser, currentUser, targetUser, targetUser).Scan(ctx, &result)
	if err != nil {
		return false, err
	}
	return result.CanRead, nil
}
