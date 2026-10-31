package files

import (
	"auto-myself-api/internal/helpers"
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

const (
	// StorageKeyPrefix is where file bytes live in the bucket.
	StorageKeyPrefix = "maintenance_record/"

	// SignedURLTTL is how long list responses' download links stay valid.
	SignedURLTTL = 5 * time.Minute
)

// StorageRepository abstracts Cloud Storage so services can be tested with an in-memory fake.
type StorageRepository interface {
	Upload(ctx context.Context, key string, data []byte) error
	Delete(ctx context.Context, key string) error
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type FileWeb struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;not null"`
	Sha256Hash string    `json:"sha256_hash" gorm:"type:text;not null"`
	FileSize   int64     `json:"file_size" gorm:"type:bigint;not null"`
}

type File struct {
	FileWeb
	StorageKey string    `json:"-" gorm:"type:text;not null"`
	CreatedBy  uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (File) TableName() string {
	return "files"
}

func (f *File) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, f)
}

type MaintenanceRecordFile struct {
	ID                  uuid.UUID      `json:"id" gorm:"type:uuid;primaryKey;not null"`
	FileID              uuid.UUID      `json:"file_id" gorm:"type:uuid;not null"`
	MaintenanceRecordID uuid.UUID      `json:"maintenance_record_id" gorm:"type:uuid;not null"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (MaintenanceRecordFile) TableName() string {
	return "maintenance_record_files"
}

func (m *MaintenanceRecordFile) BeforeCreate(tx *gorm.DB) (err error) {
	return helpers.BeforeCreateID(tx, m)
}

// FileListEntry is the released list-response shape. It intentionally omits the storage key.
type FileListEntry struct {
	ID         uuid.UUID `json:"id"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
	Sha256Hash string    `json:"sha256_hash"`
	FileSize   int64     `json:"file_size"`
	SignedURL  string    `json:"signed_url"`
}
