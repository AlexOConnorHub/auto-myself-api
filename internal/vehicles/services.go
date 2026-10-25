package vehicles

import (
	"auto-myself-api/internal/helpers"
	"auto-myself-api/internal/users"
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
)

type VehicleService interface {
	FindIDs(context.Context) ([]uuid.UUID, error)
	FindByID(context.Context, uuid.UUID) (*Vehicle, error)
	UpdateByID(context.Context, uuid.UUID, *Vehicle) error
	DeleteByID(context.Context, uuid.UUID) error
	Create(context.Context, *Vehicle) error
}

type Service struct {
	Repository *Repository
}

func NewVehicleService(Repository *Repository) *Service {
	return &Service{
		Repository: Repository,
	}
}

func (s *Service) FindIDs(ctx context.Context) ([]uuid.UUID, error) {
	user := ctx.Value(helpers.UserCtxKey).(*users.User)

	ids, err := s.Repository.FindIDs(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find vehicles for user %s: %w", user.ID.String(), err)
	}
	return ids, nil
}

func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (*Vehicle, error) {
	vehicle, err := s.Repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find vehicle with ID %s: %w", id.String(), err)
	}
	return vehicle, nil
}

func (s *Service) UpdateByID(ctx context.Context, id uuid.UUID, vehicle *Vehicle) error {
	if err := s.Repository.UpdateByID(ctx, id, vehicle); err != nil {
		return fmt.Errorf("failed to update vehicle with ID %s: %w", id.String(), err)
	}
	return nil
}

func (s *Service) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if err := s.Repository.DeleteByID(ctx, id); err != nil {
		return fmt.Errorf("failed to delete vehicle with ID %s: %w", id.String(), err)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, vehicle *Vehicle) error {
	if err := s.Repository.Create(ctx, vehicle); err != nil {
		return fmt.Errorf("failed to create vehicle: %w", err)
	}
	return nil
}
