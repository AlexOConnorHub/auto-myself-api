package files

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type FileRepository interface {
	FindByID(context.Context, uuid.UUID) (*File, error)
	FindForMaintenance(context.Context, uuid.UUID) ([]File, error)
	FindLinkByFileID(context.Context, uuid.UUID) (*MaintenanceRecordFile, error)
	CreateForMaintenance(context.Context, uuid.UUID, *File) (*MaintenanceRecordFile, error)
	DeleteWithLink(context.Context, uuid.UUID) (string, error)
}

type Repository struct {
	database *database.Database
}

func NewFileRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*File, error) {
	return helpers.FindByID[File](ctx, r.database.GetGorm(), id)
}

func (r *Repository) FindForMaintenance(ctx context.Context, maintenanceID uuid.UUID) ([]File, error) {
	links, err := gorm.G[MaintenanceRecordFile](r.database.GetGorm()).
		Where("maintenance_record_id = ?", maintenanceID).
		Order("created_at ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []File{}, nil
	}

	fileIDs := make([]uuid.UUID, len(links))
	for i, link := range links {
		fileIDs[i] = link.FileID
	}

	return gorm.G[File](r.database.GetGorm()).
		Where("id IN ?", fileIDs).
		Order("created_at ASC").
		Find(ctx)
}

func (r *Repository) FindLinkByFileID(ctx context.Context, fileID uuid.UUID) (*MaintenanceRecordFile, error) {
	link, err := gorm.G[MaintenanceRecordFile](r.database.GetGorm()).
		Where("file_id = ?", fileID).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

// CreateForMaintenance inserts the file row and its maintenance-record link in one transaction.
// The file ID must already be allocated so its storage key matches where the bytes were uploaded.
func (r *Repository) CreateForMaintenance(ctx context.Context, maintenanceID uuid.UUID, file *File) (*MaintenanceRecordFile, error) {
	if err := helpers.BeforeCreateID(r.database.GetGorm(), file); err != nil {
		return nil, err
	}

	file.StorageKey = StorageKeyPrefix + file.ID.String()

	var link *MaintenanceRecordFile
	err := r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := gorm.G[File](tx).Create(ctx, file); err != nil {
			return err
		}

		link = &MaintenanceRecordFile{
			FileID:              file.ID,
			MaintenanceRecordID: maintenanceID,
		}
		return gorm.G[MaintenanceRecordFile](tx).Create(ctx, link)
	})
	if err != nil {
		return nil, err
	}
	return link, nil
}

// DeleteWithLink soft-deletes the link and the file row, returning the storage key for cleanup.
func (r *Repository) DeleteWithLink(ctx context.Context, fileID uuid.UUID) (string, error) {
	var storageKey string

	err := r.database.GetGorm().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		file, err := helpers.FindByID[File](ctx, tx, fileID)
		if err != nil {
			return err
		}
		storageKey = file.StorageKey

		link, err := gorm.G[MaintenanceRecordFile](tx).Where("file_id = ?", fileID).First(ctx)
		if err != nil {
			return err
		}

		rows, err := gorm.G[MaintenanceRecordFile](tx).Where("id = ?", link.ID).Delete(ctx)
		if err != nil {
			return err
		}
		if rows != 1 {
			return helpers.ErrRowCountOff
		}

		rows, err = gorm.G[File](tx).Where("id = ?", fileID).Delete(ctx)
		if err != nil {
			return err
		}
		if rows != 1 {
			return helpers.ErrRowCountOff
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	return storageKey, nil
}
