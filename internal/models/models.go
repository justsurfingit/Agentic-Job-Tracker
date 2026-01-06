package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Email         string `gorm:"uniqueIndex;not null" json:"email"`
	LastHistoryID uint64 `json:"last_history_id"`
}

type Company struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Ensure Name matches the JSON key you want
	Name string `gorm:"uniqueIndex;not null" json:"company_name"`

	// 'omitempty' prevents infinite loops when fetching a Job -> Company -> Jobs -> ...
	Jobs []Job `json:"jobs,omitempty"`
}

type Job struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Foreign Key
	CompanyID uint `json:"company_id"`
	// Association: GORM needs Preload() to fill this
	Company Company `json:"company"`

	Title       string `gorm:"not null" json:"title"`
	Description string `gorm:"type:text" json:"description"`
	JobLink     string `json:"job_link"`
	Status      string `gorm:"default:'APPLIED'" json:"status"`
	ResumeLink  string `json:"resume_link"`
}

type JobEvent struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	JobID     uint      `json:"job_id"`
	EventType string    `json:"event_type"`
	Details   string    `gorm:"type:text" json:"details"`
}

type ProcessedEmail struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
}
