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
	FindByID(context.Context, uuid.UUID) error
	UpdateByID(context.Context, uuid.UUID, *Report) error
}

type Repository struct {
	database *database.Database
}

func NewRepository(db *database.Database) *Repository {
	return &Repository{
		database: db,
	}
}

func (r *Repository) Create(ctx context.Context, report *Report) error {
	return gorm.G[Report](r.database.GetGorm()).Create(ctx, report)
}

func (r *Repository) ListUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	reports, err := gorm.G[Report](r.database.GetGorm()).Select("id").Where("resolved = ?", false).Find(ctx)
	if err != nil {
		return nil, err
	}
	for _, report := range reports {
		ids = append(ids, report.ID)
	}
	return ids, nil
}

func (r *Repository) ListAllDateRange(ctx context.Context, startDate time.Time, endDate time.Time) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if startDate.IsZero() {
		startDate = time.Now().AddDate(-1, 0, 0)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}
	reports, err := gorm.G[Report](r.database.GetGorm()).Select("id").Where("created_at BETWEEN ? AND ?", startDate, endDate).Find(ctx)
	if err != nil {
		return nil, err
	}
	for _, report := range reports {
		ids = append(ids, report.ID)
	}
	return ids, nil
}

func (r *Repository) ListFromUserID(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	reports, err := gorm.G[Report](r.database.GetGorm()).Select("id").Where("created_by = ?", userID).Find(ctx)
	if err != nil {
		return nil, err
	}
	for _, report := range reports {
		ids = append(ids, report.ID)
	}
	return ids, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	report, err := gorm.G[Report](r.database.GetGorm()).Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *Repository) UpdateByID(ctx context.Context, id uuid.UUID, reportPost *Report) error {
	affectedCount, err := gorm.G[Report](r.database.GetGorm()).Where("id = ?", id).Updates(ctx, *reportPost)
	if err != nil {
		return err
	}
	if affectedCount == 0 {
		return gorm.ErrRecordNotFound
	} else if affectedCount > 1 {
		return helpers.ErrRowCountOff
	}
	return nil
}
