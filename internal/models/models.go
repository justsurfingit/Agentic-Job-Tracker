package models

import (
	"time"

	"gorm.io/gorm"
)

// Company represents the organization you are applying to.

type Company struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                   // Soft delete: keeps data but marks as deleted
	Name      string         `gorm:"uniqueIndex;not null" json:"name"` // TODO: Have to implement the name normalization
	Jobs      []Job          `json:"jobs,omitempty"`
}

// Job represents the specific role.
type Job struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Foreign Key: Links this job to a specific Company
	CompanyID uint    `json:"company_id"`
	Company   Company `json:"company"`

	Title       string `gorm:"not null" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	JobLink     string `json:"job_link"`

	// Status tracking
	Status string `gorm:"default:'APPLIED'" json:"status"` // E.g., APPLIED, INTERVIEW, REJECTED

	// Resume tracking: Which version of your resume did you use?
	ResumeLink string `json:"resume_link"`
}

// JobEvent tracks the history of what happened (Audit Log).
type JobEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	JobID     uint   `json:"job_id"`
	Job       Job    `json:"-"`                        // We don't need to return the full Job JSON here
	EventType string `json:"event_type"`               // e.g., "STATUS_CHANGE", "EMAIL_RECEIVED"
	Details   string `gorm:"type:text" json:"details"` // JSON blob or text summary
}
