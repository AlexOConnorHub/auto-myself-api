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
	ErrAccessDenied = errors.New("access denied")
	ErrRowCountOff  = errors.New("unexpected number of rows")
)

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

// UpdateByID applies the non-zero fields of item and replaces item with the persisted row.
// PostgreSQL's UPDATE ... RETURNING provides the authoritative state (trigger-maintained columns
// included) in the same statement, so callers do not need a follow-up SELECT. On error, item may
// be partially modified and must not be used.
func UpdateByID[T any](ctx context.Context, db *gorm.DB, id uuid.UUID, item *T) error {
	var updated T
	result := db.WithContext(ctx).
		Model(&updated).
		Clauses(clause.Returning{}).
		Where("id = ?", id).
		Updates(*item)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrRowCountOff
	}
	*item = updated
	return nil
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
