package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	// Import LangChainGo packages (you'll need to find the specific imports for Gemini)
	// Hint: look for "github.com/tmc/langchaingo/llms/googleai"
)

type LLMService struct {
	// You might want to hold the LLM client here so you don't recreate it every time
	Client llms.Model
}

// NewLLMService initializes the client
func NewLLMService() *LLMService {
	// 1. Initialize the Gemini client here using the API key from os.Getenv()
	// 2. Return the service instance
	ctx := context.Background()

	// Debugging: Print this to see if it's actually working (Remove this line later!)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("CRITICAL ERROR: GEMINI_API_KEY is empty. Did you load the .env file?")
	}

	// Initialize the client with the Key and the Model
	llm, err := googleai.New(ctx,
		googleai.WithAPIKey(apiKey), // Explicitly pass the key
		googleai.WithDefaultModel("gemini-2.5-flash"),
	)

	if err != nil {
		log.Fatal("Failed to create Gemini client:", err)
	}

	return &LLMService{
		Client: llm,
	}
}
func (s *LLMService) GetJobInformation(prompt string) {
	ctx := context.Background()
	// Reuse s.Client here!
	resp, _ := llms.GenerateFromSinglePrompt(ctx, s.Client, prompt)
	log.Println("Response:", resp)
}

// ExtractJobDetails takes raw HTML and returns a structured object
func (s *LLMService) ExtractJobDetails(rawHTML string) (string, error) {

	ctx := context.Background()
	if len(rawHTML) > 20000 {
		rawHTML = rawHTML[:20000]

	}
	const JobExtractionPrompt = `
You are an expert Job Data Extraction Agent. Your task is to analyze the provided raw HTML/Text from a job posting and extract structured data.

### INSTRUCTIONS:
1. **Analyze** the text to identify the core job details.
2. **Ignore** navigation menus, footers, "similar jobs" lists, and site advertisements.
3. **Extract** the following fields strictly.
4. **Format** the output as valid JSON only. Do not wrap the output in markdown code blocks.

### OUTPUT SCHEMA:
{
    "company_name": "Name of the company (e.g., Google, StartupInc)",
    "role_title": "Job title (e.g., Senior Backend Engineer)",
    "location": "Job location or 'Remote'",
    "description": "A clean summary of the job. Focus on Responsibilities and Requirements. Remove HTML tags.",
    "tech_stack": ["Array", "of", "technologies", "mentioned", "e.g., Go, React, AWS"],
    "salary_range": "The salary string if explicitly mentioned (e.g., '$100k - $150k'), otherwise null",
    
}

### CONSTRAINT:
If a piece of information is missing, set the value to null. Do not hallucinate or guess.

### RAW CONTENT:
%s
`
	prompt := fmt.Sprintf(JobExtractionPrompt, rawHTML)
	resp, err := llms.GenerateFromSinglePrompt(ctx, s.Client, prompt)
	if err != nil {
		return "", err
	}
	return resp, nil
}
