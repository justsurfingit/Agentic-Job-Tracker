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
	// So we have to create a Job and enter its entry into the database
	var company models.Company
	// it create an entry if it already don't exist
	err := s.DB.Where(models.Company{Name: req.CompanyName}).
		FirstOrCreate(&company).Error
	if err != nil {
		return nil, err
	}
	// creating the job
	job := &models.Job{
		CompanyID:   company.ID,
		Title:       req.Title,
		Description: req.Description,
		JobLink:     req.JobLink,
		ResumeLink:  req.ResumeLink,
	}
	err = s.DB.Create(job).Error
	if err != nil {
		return nil, err
	}
	return job, nil
}
