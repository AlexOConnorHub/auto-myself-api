package export

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// UserExport is the top-level user.json document. It deliberately contains only account data;
// credentials (refresh tokens, identity subjects) never enter an export.
type UserExport struct {
	ID         uuid.UUID        `json:"id"`
	Username   string           `json:"username"`
	Role       string           `json:"role"`
	Identities []IdentityExport `json:"identities"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// IdentityExport carries the OAuth provider and email without the external subject identifier.
type IdentityExport struct {
	Provider string `json:"provider"`
	Email    string `json:"email"`
}

type VehicleExport struct {
	ID        uuid.UUID `json:"id"`
	Nickname  string    `json:"nickname"`
	Make      string    `json:"make"`
	MakeID    int       `json:"make_id"`
	Model     string    `json:"model"`
	ModelID   int       `json:"model_id"`
	Year      int       `json:"year"`
	Vin       string    `json:"vin"`
	Lpn       string    `json:"lpn"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MaintenanceRecordExport struct {
	ID           uuid.UUID `json:"id"`
	VehicleID    uuid.UUID `json:"vehicle_id"`
	Cost         string    `json:"cost"`
	Odometer     int       `json:"odometer"`
	Timestamp    time.Time `json:"timestamp"`
	Notes        string    `json:"notes"`
	Type         string    `json:"type"`
	Interval     int       `json:"interval"`
	IntervalType string    `json:"interval_type"`
	CreatedBy    uuid.UUID `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type FileExport struct {
	ID                  uuid.UUID `json:"id"`
	MaintenanceRecordID uuid.UUID `json:"maintenance_record_id"`
	Sha256Hash          string    `json:"sha256_hash"`
	FileSize            int64     `json:"file_size"`
	CreatedBy           uuid.UUID `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	SignedURL           string    `json:"signed_url"`
}

type ReportExport struct {
	ID            uuid.UUID  `json:"id"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	ReportedTopic string     `json:"reported_topic"`
	ReportedID    uuid.UUID  `json:"reported_id"`
	Reason        string     `json:"reason"`
	PublicNotes   string     `json:"public_notes"`
	Resolved      bool       `json:"resolved"`
	ResolvedAt    *time.Time `json:"resolved_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type VehicleArchive struct {
	Vehicle            VehicleExport
	MaintenanceRecords []MaintenanceRecordExport
	Files              []FileExport
}

type Archive struct {
	User     UserExport
	Vehicles []VehicleArchive
	Reports  []ReportExport
}
