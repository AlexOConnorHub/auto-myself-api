package files

import "context"

type FilesService interface {
	GetIDsForRecord(context.Context, string, string) ([]string, error)
	GetByID(context.Context, string) (*File, error)
	CreateForRecord(context.Context, string, string, *File) error
	DeleteByID(context.Context, string) error
}

type Service struct {
	repository Repository
}

func NewFilesService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetIDsForRecord(ctx context.Context, recordType string, recordID string) ([]string, error) {

	// type fileReturnStruct struct {
	// 	ID         string `json:"id"`
	// 	CreatedAt  string `json:"created_at"`
	// 	UpdatedAt  string `json:"updated_at"`
	// 	Sha256Hash string `json:"sha256_hash"`
	// 	FileSize   int64  `json:"file_size"`
	// 	SignedUrl  string `json:"signed_url"`
	// }

	switch fileType {
	case "maintenance":
		var maintenance MaintenanceRecord
		err := a.Gorm.
			Preload("Vehicle").
			Preload("FileLink").
			Preload("FileLink.File").
			Where("id = ?", uuid).
			First(&maintenance).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, gin.H{"error": "Maintenance record not found"})
			} else {
				c.JSON(500, gin.H{"error": "Database error"})
			}
			return
		}

		if !maintenance.Vehicle.CanRead(a, user) {
			c.JSON(404, gin.H{"error": "Maintenance record not found"})
			return
		}

		var final []fileReturnStruct

		for _, file_link := range maintenance.FileLink {
			signedUrl, err := helpers.GetFileUrl(file_link.File.StorageKey)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to generate signed URL"})
				println(err.Error())
				println("Failed to generate signed URL for file ID: " + file_link.File.StorageKey)
				return
			}

			final = append(final, fileReturnStruct{
				ID:         file_link.File.ID.String(),
				CreatedAt:  file_link.File.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				UpdatedAt:  file_link.File.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
				Sha256Hash: file_link.File.Sha256Hash,
				FileSize:   file_link.File.FileSize,
				SignedUrl:  signedUrl,
			})
		}

		c.JSON(200, final)
		return

	default:
		c.JSON(400, gin.H{"error": "Invalid file type"})
		return
	}
