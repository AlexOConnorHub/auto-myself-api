package reports

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"gorm.io/gorm"
)

type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type ReportService interface {
	Create(context.Context, *Report) error
	List(context.Context) ([]uuid.UUID, error)
	ListUnresolved(context.Context) ([]uuid.UUID, error)
	ListByDate(context.Context, DateRange) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Report, error)
	UpdateByID(context.Context, uuid.UUID, *Report) error
}

type Service struct {
	repository *Repository
}

func NewReportService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, report *Report) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	report.CreatedBy = user.ID

	if err := s.repository.Create(ctx, report); err != nil {
		return fmt.Errorf("creating report: %w", err)
	}
	return nil
}

// List returns the caller's own reports for every role. Moderators and admins use
// ListUnresolved to see the moderation queue, so their own submissions stay visible to them.
func (s *Service) List(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	ids, err := s.repository.ListFromUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("listing reports for user %s: %w", user.ID, err)
	}
	return ids, nil
}

func (s *Service) ListUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	if !isModerator(user) {
		return nil, helpers.ErrAccessDenied
	}

	ids, err := s.repository.ListUnresolved(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing unresolved reports: %w", err)
	}
	return ids, nil
}

func (s *Service) ListByDate(ctx context.Context, dateRange DateRange) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	if !isModerator(user) {
		return nil, helpers.ErrAccessDenied
	}

	ids, err := s.repository.ListAllDateRange(ctx, dateRange.StartDate, dateRange.EndDate)
	if err != nil {
		return nil, fmt.Errorf("listing reports by date: %w", err)
	}
	return ids, nil
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	report, err := s.repository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("finding report %s: %w", id, err)
	}

	if !isModerator(user) && report.CreatedBy != user.ID {
		return nil, gorm.ErrRecordNotFound
	}

	return report, nil
}

func (s *Service) UpdateByID(ctx context.Context, id uuid.UUID, input *Report) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	existing, err := s.repository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return fmt.Errorf("finding report %s: %w", id, err)
	}

	if !isModerator(user) {
		if existing.CreatedBy != user.ID {
			return gorm.ErrRecordNotFound
		}
		return helpers.ErrAccessDenied
	}

	switch {
	case existing.Resolved:
		return helpers.ErrAccessDenied
	case input.Resolved:
		now := time.Now()
		input.ResolvedAt = &now
		input.ResolvedBy = &user.ID
	}

	if err := s.repository.UpdateByID(ctx, id, input.Version, input); err != nil {
		if errors.Is(err, helpers.ErrVersionConflict) {
			return err
		}
		return fmt.Errorf("updating report %s: %w", id, err)
	}
	return nil
}

func isModerator(user *users.User) bool {
	return user.Role == users.RoleAdmin || user.Role == users.RoleModerator
}
