package testutil

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

func CurrentVersion(t *testing.T, db *gorm.DB, table string, id uuid.UUID) int64 {
	t.Helper()

	var version int64
	err := db.Table(table).
		Select("version").
		Where("id = ? AND deleted_at IS NULL", id).
		Row().
		Scan(&version)
	if err != nil {
		t.Fatalf("reading version from %s %s: %v", table, id, err)
	}
	return version
}
