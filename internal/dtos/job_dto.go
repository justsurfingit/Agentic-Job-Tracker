package dtos

type JobExtractionRequest struct {
	RawHTML string `json:"raw_html" binding:"required"`
	URL     string `json:"url"`
}

type JobCreationRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Title       string `json:"role_title" binding:"required"`
	JobLink     string `json:"job_link" binding:"required"`
	Description string `json:"description" binding:"required"`

	// Optional Fields
	Location    string   `json:"location"`
	SalaryRange string   `json:"salary_range"`
	TechStack   []string `json:"tech_stack"`
	ResumeLink  string   `json:"resume_link"`
	Status      string   `json:"status"` // Defaults to "APPLIED" if empty
}
