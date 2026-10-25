package files

import (
	"auto-myself-api/internal/database"
	"context"
)

type FileRepository interface {
	GetIDsForRecord(context.Context, string, string) ([]string, error)
	GetByID(context.Context, string) (*File, error)
	CreateForRecord(context.Context, string, string, *File) error
	DeleteByID(context.Context, string) error
}

type Repository struct {
	database *database.Database
}

func NewFileRepository(database *database.Database) *Repository {
	return &Repository{
		database: database,
	}
}
