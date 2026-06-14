package models

import (
	"auto-myself-api/app"
	"auto-myself-api/helpers"

	_ "github.com/joho/godotenv/autoload"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type FileBase struct {
	StorageKey string `json:"storage_key" gorm:"type:text;not null;"`
	Sha256Hash string `json:"sha256_hash" gorm:"type:text;not null;"`
	FileSize   int64  `json:"file_size" gorm:"type:bigint;not null;"`
}

type File struct {
	helpers.DatabaseMetadata
	FileBase
	CreatedBy     uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedByUser User      `gorm:"foreignKey:CreatedBy;references:ID;"`
}

func (File) TableName() string {
	return "files"
}

func (f *File) BeforeCreate(tx *gorm.DB) (err error) {
	if f.ID.IsNil() {
		f.ID, err = uuid.NewV7()
	}
	return err
}

func (f *File) DeleteWithFile(a *app.App) error {
	if err := helpers.DeleteFile(f.StorageKey); err != nil {
		return err
	}

	if err := a.Gorm.Delete(&f).Error; err != nil {
		return err
	}

	return nil
}

func (f *File) UploadFile(a *app.App, data *[]byte) error {
	if err := helpers.CreateFile(f.StorageKey, data); err != nil {
		return err
	}

	return nil
}
