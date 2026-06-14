package models

import (
	"auto-myself-api/app"
	"auto-myself-api/helpers"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type MaintenanceRecordFileBase struct {
	FileID              uuid.UUID `json:"file_id,omitempty" gorm:"type:uuid;not null;"`
	MaintenanceRecordID uuid.UUID `json:"maintenance_record_id,omitempty" gorm:"type:uuid;not null;"`
}

type MaintenanceRecordFile struct {
	helpers.DatabaseMetadata
	MaintenanceRecordFileBase
	File              File              `gorm:"foreignKey:FileID;references:ID;" json:"file"`
	MaintenanceRecord MaintenanceRecord `gorm:"foreignKey:MaintenanceRecordID;references:ID;" json:"maintenance_record"`
}

var RelationString = "maintenance_record"

func (mrf *MaintenanceRecordFile) BeforeCreate(tx *gorm.DB) (err error) {
	if mrf.ID.IsNil() {
		mrf.ID, err = uuid.NewV7()
	}
	return err
}

func (f *MaintenanceRecordFile) DeleteWithFile(a *app.App) error {
	a.Gorm.Model(&f).Association("File").Find(&f.File)

	if err := f.File.DeleteWithFile(a); err != nil {
		return err
	}

	if err := a.Gorm.Delete(&f).Error; err != nil {
		return err
	}

	return nil
}

func (f *MaintenanceRecordFile) UploadFile(a *app.App, data *[]byte) error {
	a.Gorm.Model(&f).Association("File").Find(&f.File)
	a.Gorm.Model(&f).Association("MaintenanceRecord").Find(&f.MaintenanceRecord)
	a.Gorm.Model(&f.MaintenanceRecord).Association("Vehicle").Find(&f.MaintenanceRecord.Vehicle)

	if !f.MaintenanceRecord.Vehicle.CanWrite(a, nil) {
		return fmt.Errorf("forbidden")
	}

	if err := helpers.CreateFile(RelationString+f.FileID.String(), data); err != nil {
		return err
	}

	if err := a.Gorm.Save(&f.File).Error; err != nil {
		return err
	}

	if err := a.Gorm.Save(&f).Error; err != nil {
		return err
	}

	return nil
}
