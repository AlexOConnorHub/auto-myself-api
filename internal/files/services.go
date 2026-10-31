package files

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/maintenance"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

var ErrUnsupportedRecordType = errors.New("unsupported record type")

type FileService interface {
	GetForRecord(context.Context, string, uuid.UUID) ([]FileListEntry, error)
	CreateForRecord(context.Context, string, uuid.UUID, []byte, *MaintenanceRecordFile) error
	DeleteByID(context.Context, uuid.UUID) error
}

type Service struct {
	fileRepository        *Repository
	storage               StorageRepository
	maintenanceRepository *maintenance.Repository
	vehicleRepository     *vehicles.Repository
}

func NewFileService(
	fileRepository *Repository,
	storage StorageRepository,
	maintenanceRepository *maintenance.Repository,
	vehicleRepository *vehicles.Repository,
) *Service {
	return &Service{
		fileRepository:        fileRepository,
		storage:               storage,
		maintenanceRepository: maintenanceRepository,
		vehicleRepository:     vehicleRepository,
	}
}

func (s *Service) GetForRecord(ctx context.Context, recordType string, recordID uuid.UUID) ([]FileListEntry, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	record, err := s.findMaintenanceRecord(ctx, recordType, recordID)
	if err != nil {
		return nil, err
	}

	canRead, err := s.vehicleRepository.CanRead(ctx, record.VehicleID, user.ID)
	if err != nil {
		return nil, fmt.Errorf("checking read access to vehicle %s: %w", record.VehicleID, err)
	}
	if !canRead {
		return nil, gorm.ErrRecordNotFound
	}

	storedFiles, err := s.fileRepository.FindForMaintenance(ctx, recordID)
	if err != nil {
		return nil, fmt.Errorf("finding files for maintenance record %s: %w", recordID, err)
	}

	entries := make([]FileListEntry, 0, len(storedFiles))
	for _, file := range storedFiles {
		signedURL, err := s.storage.SignedURL(ctx, file.StorageKey, SignedURLTTL)
		if err != nil {
			return nil, fmt.Errorf("signing URL for file %s: %w", file.ID, err)
		}

		entries = append(entries, FileListEntry{
			ID:         file.ID,
			CreatedAt:  file.CreatedAt.Format(time.RFC3339),
			UpdatedAt:  file.UpdatedAt.Format(time.RFC3339),
			Sha256Hash: file.Sha256Hash,
			FileSize:   file.FileSize,
			SignedURL:  signedURL,
		})
	}

	return entries, nil
}

// CreateForRecord uploads data and fills link with the persisted maintenance-record link.
func (s *Service) CreateForRecord(ctx context.Context, recordType string, recordID uuid.UUID, data []byte, link *MaintenanceRecordFile) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	record, err := s.findMaintenanceRecord(ctx, recordType, recordID)
	if err != nil {
		return err
	}

	if err := s.authorizeWrite(ctx, record.VehicleID, user.ID); err != nil {
		return err
	}

	hash := sha256.Sum256(data)
	file := &File{
		FileWeb: FileWeb{
			Sha256Hash: hex.EncodeToString(hash[:]),
			FileSize:   int64(len(data)),
		},
		CreatedBy: user.ID,
	}

	if err := helpers.BeforeCreateID(s.fileRepository.database.GetGorm(), file); err != nil {
		return err
	}

	storageKey := StorageKeyPrefix + file.ID.String()
	if err := s.storage.Upload(ctx, storageKey, data); err != nil {
		return fmt.Errorf("uploading file %s: %w", file.ID, err)
	}

	// Prefer orphaned storage object over failed database write.
	created, err := s.fileRepository.CreateForMaintenance(ctx, recordID, file)
	if err != nil {
		if deleteErr := s.storage.Delete(ctx, storageKey); deleteErr != nil {
			return fmt.Errorf("creating file record: %w (removing uploaded object failed: %v)", err, deleteErr)
		}
		return fmt.Errorf("creating file record: %w", err)
	}

	*link = *created
	return nil
}

func (s *Service) DeleteByID(ctx context.Context, fileID uuid.UUID) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	if _, err := s.fileRepository.FindByID(ctx, fileID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding file %s: %w", fileID, err)
	}

	link, err := s.fileRepository.FindLinkByFileID(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding link for file %s: %w", fileID, err)
	}

	record, err := s.maintenanceRepository.FindByID(ctx, link.MaintenanceRecordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding maintenance record %s: %w", link.MaintenanceRecordID, err)
	}

	if err := s.authorizeWrite(ctx, record.VehicleID, user.ID); err != nil {
		return err
	}

	storageKey, err := s.fileRepository.DeleteWithLink(ctx, fileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("deleting file %s: %w", fileID, err)
	}

	// Prefer database delete failure causing orphaned object over pointing to missing storage bytes.
	if err := s.storage.Delete(ctx, storageKey); err != nil {
		slog.ErrorContext(ctx, "failed to delete file object; leaving orphan", "storage_key", storageKey, "error", err)
	}

	return nil
}

func (s *Service) findMaintenanceRecord(ctx context.Context, recordType string, recordID uuid.UUID) (*maintenance.MaintenanceRecord, error) {
	if recordType != "maintenance" {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedRecordType, recordType)
	}

	record, err := s.maintenanceRepository.FindByID(ctx, recordID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding maintenance record %s: %w", recordID, err)
	}
	return record, nil
}

func (s *Service) authorizeWrite(ctx context.Context, vehicleID, userID uuid.UUID) error {
	canRead, err := s.vehicleRepository.CanRead(ctx, vehicleID, userID)
	if err != nil {
		return fmt.Errorf("checking read access to vehicle %s: %w", vehicleID, err)
	}
	canWrite, err := s.vehicleRepository.CanWrite(ctx, vehicleID, userID)
	if err != nil {
		return fmt.Errorf("checking write access to vehicle %s: %w", vehicleID, err)
	}
	switch {
	case canWrite:
		return nil
	case canRead:
		return helpers.ErrAccessDenied
	default:
		return gorm.ErrRecordNotFound
	}
}
