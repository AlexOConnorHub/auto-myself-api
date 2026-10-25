package files

import (
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type FileWeb struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;not null;"`
	StorageKey string    `json:"storage_key" gorm:"type:text;not null;"`
	Sha256Hash string    `json:"sha256_hash" gorm:"type:text;not null;"`
	FileSize   int64     `json:"file_size" gorm:"type:bigint;not null;"`
}

type File struct {
	FileWeb
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
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

// func (f *File) DeleteWithFile(a *server.App) error {
// 	if err := helpers.DeleteFile(f.StorageKey); err != nil {
// 		return err
// 	}

// 	if err := a.Gorm.Delete(&f).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (f *File) UploadFile(a *server.App, data *[]byte) error {
// 	if err := helpers.CreateFile(f.StorageKey, data); err != nil {
// 		return err
// 	}

// 	return nil
// }

type MaintenanceRecordFile struct {
	ID                  uuid.UUID `json:"id" gorm:"type:uuid;not null;"`
	FileID              uuid.UUID `json:"file_id,omitempty" gorm:"type:uuid;not null;"`
	MaintenanceRecordID uuid.UUID `json:"maintenance_record_id,omitempty" gorm:"type:uuid;not null;"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

func (mrf *MaintenanceRecordFile) BeforeCreate(tx *gorm.DB) (err error) {
	if mrf.ID.IsNil() {
		mrf.ID, err = uuid.NewV7()
	}
	return err
}

// func (f *MaintenanceRecordFile) DeleteWithFile(a *server.App) error {
// 	a.Gorm.Model(&f).Association("File").Find(&f.File)

// 	if err := f.File.DeleteWithFile(a); err != nil {
// 		return err
// 	}

// 	if err := a.Gorm.Delete(&f).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (f *MaintenanceRecordFile) UploadFile(a *server.App, data *[]byte) error {
// 	a.Gorm.Model(&f).Association("File").Find(&f.File)
// 	a.Gorm.Model(&f).Association("MaintenanceRecord").Find(&f.MaintenanceRecord)
// 	a.Gorm.Model(&f.MaintenanceRecord).Association("Vehicle").Find(&f.MaintenanceRecord.Vehicle)

// 	if !f.MaintenanceRecord.Vehicle.CanWrite(a, nil) {
// 		return fmt.Errorf("forbidden")
// 	}

// 	if err := helpers.CreateFile(fmt.Sprintf("maintenance_record/%s", f.FileID.String()), data); err != nil {
// 		return err
// 	}

// 	if err := a.Gorm.Save(&f.File).Error; err != nil {
// 		return err
// 	}

// 	if err := a.Gorm.Save(&f).Error; err != nil {
// 		return err
// 	}

// 	return nil
// }
