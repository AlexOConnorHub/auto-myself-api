package helpers

import (
	"context"
	"errors"
	"reflect"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	UserCtxKey   = "user"
	UUIDParamKey = "uuid_param"
)

var (
	ErrAccessDenied    = errors.New("access denied")
	ErrRowCountOff     = errors.New("unexpected number of rows")
	ErrVersionConflict = errors.New("version conflict")
)

type VersionConflictError struct {
	CurrentVersion int64
}

func (e *VersionConflictError) Error() string {
	return "version conflict"
}

func (e *VersionConflictError) Is(target error) bool {
	return target == ErrVersionConflict
}

func BeforeCreateID[T any](tx *gorm.DB, t *T) (err error) {
	if idField := reflect.ValueOf(t).Elem().FieldByName("ID"); idField.IsValid() && idField.CanSet() {
		if idField.Interface().(uuid.UUID).IsNil() {
			var err error
			newID, err := uuid.NewV7()
			if err != nil {
				return err
			}
			idField.Set(reflect.ValueOf(newID))
			return nil
		}
	}
	return nil
}

func FindByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID) (*T, error) {
	item, err := gorm.G[T](db).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func UpdateByIDVersioned[T any](ctx context.Context, db *gorm.DB, id uuid.UUID, expectedVersion int64, item *T) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current T
		applied := tx.Model(&current).
			Where("id = ? AND version = ?", id, expectedVersion).
			Omit("Version", "ID").
			Updates(*item)
		if applied.Error != nil {
			return applied.Error
		}
		if applied.RowsAffected == 0 {
			return versionConflict[T](tx, id)
		}

		var persisted T
		bumped := tx.Model(&persisted).
			Clauses(clause.Returning{}).
			Where("id = ?", id).
			Updates(map[string]any{"version": gorm.Expr("version + 1")})
		if bumped.Error != nil {
			return bumped.Error
		}
		if bumped.RowsAffected != 1 {
			return ErrRowCountOff
		}
		*item = persisted
		return nil
	})
}

func versionConflict[T any](tx *gorm.DB, id uuid.UUID) error {
	var row struct {
		Version int64
	}
	result := tx.Model(new(T)).
		Select("version").
		Where("id = ?", id).
		Scan(&row)
	if result.Error != nil || result.RowsAffected != 1 {
		return ErrVersionConflict
	}
	return &VersionConflictError{CurrentVersion: row.Version}
}

func DeleteByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID) error {
	rows, err := gorm.G[T](db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrRowCountOff
	}
	return nil
}

func Create[T any](ctx context.Context, db *gorm.DB, item *T) error {
	return gorm.G[T](db).Create(ctx, item)
}
