package reports

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"time"

	"github.com/gofrs/uuid/v5"
)

type DateRange struct {
	StartDate time.Time
	EndDate   time.Time
}

type ReportService interface {
	Create(context.Context, *Report) error
	List(context.Context) ([]uuid.UUID, error)
	ListByDate(context.Context, DateRange) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Report, error)
	UpdateByID(context.Context, uuid.UUID, *ReportResolutionPost) (*Report, error)
}

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, report *Report) error {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	report.CreatedBy = user.ID

	return s.repository.Create(ctx, report)
}

func (s *Service) List(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	if user.Role == users.RoleAdmin || user.Role == users.RoleModerator {
		return s.repository.ListUnresolved(ctx)
	} else {
		return s.repository.ListFromUserID(ctx, user.ID)
	}
}

func (s *Service) ListByDate(ctx context.Context, dateRange DateRange) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	if user.Role != users.RoleAdmin && user.Role != users.RoleModerator {
		return nil, helpers.ErrAccessDenied
	}
	return s.repository.ListAllDateRange(ctx, dateRange.StartDate, dateRange.EndDate)
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*Report, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	report, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if user.Role != users.RoleAdmin && user.Role != users.RoleModerator && report.CreatedBy != user.ID {
		return nil, helpers.ErrAccessDenied
	}

	return report, nil
}

func (s *Service) UpdateByID(ctx context.Context, id uuid.UUID, reportPost *ReportResolutionPost) (*Report, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)
	if user.Role != users.RoleAdmin && user.Role != users.RoleModerator {
		return nil, helpers.ErrAccessDenied
	}

	report, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if report.Resolved {
		if user.ID != report.ResolvedBy || user.Role != users.RoleAdmin {
			return nil, helpers.ErrAccessDenied
		}

		report.InternalNotes = reportPost.InternalNotes
	} else {
		report.Resolved = reportPost.Resolved
		report.PublicNotes = reportPost.PublicNotes
		report.InternalNotes = reportPost.InternalNotes
		if report.Resolved {
			now := time.Now()
			report.ResolvedAt = now
			report.ResolvedBy = user.ID
		}
	}

	if err := s.repository.UpdateByID(ctx, id, report); err != nil {
		return nil, err
	}
	return report, nil
}
