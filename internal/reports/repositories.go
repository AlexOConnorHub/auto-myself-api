package reports

import (
	"auto-myself-api/internal/database"
	"auto-myself-api/internal/helpers"
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type ReportRepository interface {
	Create(context.Context, *Report) error
	ListUnresolved(context.Context) ([]uuid.UUID, error)
	ListAllDateRange(context.Context, time.Time, time.Time) ([]uuid.UUID, error)
	ListFromUserID(context.Context, uuid.UUID) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Report, error)
	UpdateByID(context.Context, uuid.UUID, int64, *Report) error
}

type Repository struct {
	database *database.Database
}

func NewReportRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}

func (r *Repository) Create(ctx context.Context, report *Report) error {
	return helpers.Create[Report](ctx, r.database.GetGorm(), report)
}

func (r *Repository) ListUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	reports, err := gorm.G[Report](r.database.GetGorm()).
		Select("id").
		Where("resolved = ?", false).
		Order("created_at ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return reportIDs(reports), nil
}

func (r *Repository) ListAllDateRange(ctx context.Context, startDate time.Time, endDate time.Time) ([]uuid.UUID, error) {
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	reports, err := gorm.G[Report](r.database.GetGorm()).
		Select("id").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Order("created_at ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return reportIDs(reports), nil
}

func (r *Repository) ListFromUserID(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	reports, err := gorm.G[Report](r.database.GetGorm()).
		Select("id").
		Where("created_by = ?", userID).
		Order("created_at ASC").
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return reportIDs(reports), nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	return helpers.FindByID[Report](ctx, r.database.GetGorm(), id)
}

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, expectedVersion int64, report *Report) error {
	return helpers.UpdateByIDVersioned[Report](ctx, r.database.GetGorm(), id, expectedVersion, report)
}

func reportIDs(reports []Report) []uuid.UUID {
	ids := make([]uuid.UUID, len(reports))
	for i, report := range reports {
		ids[i] = report.ID
	}
	return ids
}
