package services

import (
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/dtos"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/models"
	"gorm.io/gorm"
)

type JobService struct {
	DB *gorm.DB
}

func NewJobService(db *gorm.DB) *JobService {
	return &JobService{
		DB: db,
	}
}
func (s *JobService) CreateJob(req *dtos.JobCreationRequest) (*models.Job, error) {
	// 1. Find or Create the Company
	// We search by Name. If it doesn't exist, GORM creates it.
	var company models.Company
	// Using Where(...) with FirstOrCreate is safer to ensure the Name is set on creation
	err := s.DB.Where(models.Company{Name: req.CompanyName}).
		Attrs(models.Company{Name: req.CompanyName}). // Ensure Name is set if creating new
		FirstOrCreate(&company).Error
	if err != nil {
		return nil, err
	}

	// 2. Prepare the Job Object
	// Set default status if the request didn't provide one
	status := req.Status
	if status == "" {
		status = "APPLIED"
	}

	job := &models.Job{
		CompanyID:   company.ID,
		Title:       req.Title,
		Description: req.Description,
		JobLink:     req.JobLink,
		ResumeLink:  req.ResumeLink,
		Status:      status,
		// Note: If you added Location/Salary to your Job Model, map them here too.
		// e.g. Location: req.Location,
	}

	// 3. Save Job to Database
	err = s.DB.Create(job).Error
	if err != nil {
		return nil, err
	}

	// 4. THE FIX: Manually populate the Association
	// GORM's Create() sets the ID but leaves the 'Company' struct empty.
	// We plug the company we found earlier back into the job object
	// so the frontend gets the full data immediately.
	job.Company = company

	return job, nil
}
