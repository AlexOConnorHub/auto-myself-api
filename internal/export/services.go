package export

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"auto-myself-api/internal/auth"
	"auto-myself-api/internal/files"
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/maintenance"
	"auto-myself-api/internal/reports"
	"auto-myself-api/internal/users"
	"auto-myself-api/internal/vehicles"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

const SignedURLTTL = 24 * time.Hour

type Service struct {
	authRepository        *auth.Repository
	vehicleRepository     *vehicles.Repository
	maintenanceRepository *maintenance.Repository
	fileRepository        *files.Repository
	reportRepository      *reports.Repository
	storage               files.StorageRepository
}

func NewService(
	authRepository *auth.Repository,
	vehicleRepository *vehicles.Repository,
	maintenanceRepository *maintenance.Repository,
	fileRepository *files.Repository,
	reportRepository *reports.Repository,
	storage files.StorageRepository,
) *Service {
	return &Service{
		authRepository:        authRepository,
		vehicleRepository:     vehicleRepository,
		maintenanceRepository: maintenanceRepository,
		fileRepository:        fileRepository,
		reportRepository:      reportRepository,
		storage:               storage,
	}
}

func (s *Service) Build(ctx context.Context) (*Archive, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	identities, err := s.authRepository.FindIdentitiesForUser(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("finding identities for user %s: %w", user.ID, err)
	}

	archive := &Archive{
		User: UserExport{
			ID:         user.ID,
			Username:   user.Username,
			Role:       user.Role,
			Identities: make([]IdentityExport, 0, len(identities)),
			CreatedAt:  user.CreatedAt,
			UpdatedAt:  user.UpdatedAt,
		},
		Vehicles: make([]VehicleArchive, 0),
		Reports:  make([]ReportExport, 0),
	}
	for _, identity := range identities {
		if identity.Email == "" {
			continue
		}
		archive.User.Identities = append(archive.User.Identities, IdentityExport{
			Provider: identity.Provider,
			Email:    identity.Email,
		})
	}

	vehicleIDs, err := s.vehicleRepository.FindIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("finding vehicles for user %s: %w", user.ID, err)
	}

	for _, vehicleID := range vehicleIDs {
		vehicleArchive, err := s.buildVehicle(ctx, vehicleID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// The vehicle was deleted between the ID lookup and the read.
			continue
		}
		if err != nil {
			return nil, err
		}
		archive.Vehicles = append(archive.Vehicles, *vehicleArchive)
	}

	sort.Slice(archive.Vehicles, func(i, j int) bool {
		left, right := archive.Vehicles[i].Vehicle, archive.Vehicles[j].Vehicle
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID.String() < right.ID.String()
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	reportIDs, err := s.reportRepository.ListFromUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("finding reports for user %s: %w", user.ID, err)
	}
	for _, reportID := range reportIDs {
		report, err := s.reportRepository.FindByID(ctx, reportID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("finding report %s: %w", reportID, err)
		}
		archive.Reports = append(archive.Reports, ReportExport{
			ID:            report.ID,
			CreatedBy:     report.CreatedBy,
			ReportedTopic: report.ReportedTopic,
			ReportedID:    report.ReportedID,
			Reason:        report.Reason,
			PublicNotes:   report.PublicNotes,
			Resolved:      report.Resolved,
			ResolvedAt:    report.ResolvedAt,
			CreatedAt:     report.CreatedAt,
			UpdatedAt:     report.UpdatedAt,
		})
	}
	sort.Slice(archive.Reports, func(i, j int) bool {
		left, right := archive.Reports[i], archive.Reports[j]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID.String() < right.ID.String()
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	return archive, nil
}

func (s *Service) buildVehicle(ctx context.Context, vehicleID uuid.UUID) (*VehicleArchive, error) {
	vehicle, err := s.vehicleRepository.FindByID(ctx, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("finding vehicle %s: %w", vehicleID, err)
	}

	records, err := s.maintenanceRepository.FindForVehicle(ctx, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("finding maintenance records for vehicle %s: %w", vehicleID, err)
	}

	vehicleArchive := &VehicleArchive{
		Vehicle: VehicleExport{
			ID:        vehicle.ID,
			Nickname:  vehicle.Nickname,
			Make:      vehicle.Make,
			MakeID:    vehicle.MakeID,
			Model:     vehicle.Model,
			ModelID:   vehicle.ModelID,
			Year:      vehicle.Year,
			Vin:       vehicle.Vin,
			Lpn:       vehicle.Lpn,
			CreatedBy: vehicle.CreatedBy,
			CreatedAt: vehicle.CreatedAt,
			UpdatedAt: vehicle.UpdatedAt,
		},
		MaintenanceRecords: make([]MaintenanceRecordExport, 0, len(records)),
		Files:              make([]FileExport, 0),
	}

	seenFiles := make(map[uuid.UUID]struct{})
	for _, record := range records {
		vehicleArchive.MaintenanceRecords = append(vehicleArchive.MaintenanceRecords, MaintenanceRecordExport{
			ID:           record.ID,
			VehicleID:    record.VehicleID,
			Cost:         record.Cost,
			Odometer:     record.Odometer,
			Timestamp:    record.Timestamp,
			Notes:        record.Notes,
			Type:         record.Type,
			Interval:     record.Interval,
			IntervalType: record.IntervalType,
			CreatedBy:    record.CreatedBy,
			CreatedAt:    record.CreatedAt,
			UpdatedAt:    record.UpdatedAt,
		})

		storedFiles, err := s.fileRepository.FindForMaintenance(ctx, record.ID)
		if err != nil {
			return nil, fmt.Errorf("finding files for maintenance record %s: %w", record.ID, err)
		}
		for _, file := range storedFiles {
			if _, exists := seenFiles[file.ID]; exists {
				continue
			}
			signedURL, err := s.storage.SignedURL(ctx, file.StorageKey, SignedURLTTL)
			if err != nil {
				return nil, fmt.Errorf("signing URL for file %s: %w", file.ID, err)
			}
			seenFiles[file.ID] = struct{}{}
			vehicleArchive.Files = append(vehicleArchive.Files, FileExport{
				ID:                  file.ID,
				MaintenanceRecordID: record.ID,
				Sha256Hash:          file.Sha256Hash,
				FileSize:            file.FileSize,
				CreatedBy:           file.CreatedBy,
				CreatedAt:           file.CreatedAt,
				UpdatedAt:           file.UpdatedAt,
				SignedURL:           signedURL,
			})
		}
	}

	sort.Slice(vehicleArchive.Files, func(i, j int) bool {
		left, right := vehicleArchive.Files[i], vehicleArchive.Files[j]
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID.String() < right.ID.String()
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	return vehicleArchive, nil
}
