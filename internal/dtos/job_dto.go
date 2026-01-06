package dtos

type JobExtractionRequest struct {
	RawHTML string `json:"raw_html" binding:"required"`
	URL     string `json:"url"`
}

// JobCreationRequest is what the Frontend sends when the user clicks "Save Job"
type JobCreationRequest struct {
	CompanyName string `json:"company_name" binding:"required"`
	Title       string `json:"role_title" binding:"required"`
	JobLink     string `json:"job_link" binding:"required"`
	//Gathered by the llm
	Description string   `json:"description" binding:"required"`
	Location    string   `json:"location" `
	SalaryRange string   `json:"salary_range"`
	TechStack   []string `json:"tech_stack"` 

	ResumeLink string `json:"resume_link"`
	Status     string `json:"status"` // e.g. "APPLIED" (Optional, default in DB)
}
