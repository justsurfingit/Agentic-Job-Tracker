package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/dtos"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/services"
)

// LLm service such that we can use that global llm client here
// Dependecny ingestion
type JobHandler struct {
	LLMService *services.LLMService
	JobService *services.JobService
}

// NewJobHandler creates the handler with dependencies
func NewJobHandler(llm *services.LLMService,j *services.JobService) *JobHandler {
	return &JobHandler{LLMService: llm,
		JobService: j,
	}
}

// ParseJob is the POST /jobs/extract endpoint
func (h *JobHandler) ParseJob(c *gin.Context) {
	var req dtos.JobExtractionRequest

	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format: " + err.Error()})
		return
	}
	extractedJSON, err := h.LLMService.ExtractJobDetails(req.RawHTML)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Extraction failed: " + err.Error()})
		return
	}

	// 4. Return the raw JSON string from AI directly
	// We use json.RawMessage to prevent Go from escaping the inner JSON string
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    json.RawMessage(extractedJSON),
	})
}
// creating the job
func (h *JobHandler) CreateJob(c *gin.Context) {
	var req dtos.JobCreationRequest
	if err:=c.ShouldBindJSON(&req);err!=nil{
		c.JSON(http.StatusBadRequest,gin.H{"error":"Invalid JSON format: "+err.Error()})
		return
	}
	// creating the job
	job,err:=h.JobService.CreateJob(&req)
	if err!=nil{
		c.JSON(http.StatusInternalServerError,gin.H{"error":"Failed to create job: "+err.Error()})
		return
	}
	c.JSON(http.StatusCreated,job)
}
