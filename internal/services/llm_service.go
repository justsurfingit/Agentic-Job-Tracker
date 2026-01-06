package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
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

	// Fail fast if the API key isn't available so we don't create an unauthenticated client.
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

// Function for identifying which specific role is being talked about here such that we can uniquely identified for which particular application we have recieved an update
func (s *LLMService) IdentifyJobRole(titles []string, subject, body string) int {
	ctx := context.Background()

	// Create a numbered list string for the prompt
	titlesList := ""
	for i, t := range titles {
		titlesList += fmt.Sprintf("%d. %s\n", i, t)
	}

	// Truncate body for cost saving
	if len(body) > 1000 {
		body = body[:1000]
	}

	prompt := fmt.Sprintf(`
    I have multiple job applications at this company. Based on the email, identify which role is being discussed.
    
    Candidate Roles:
    %s
    
    Email Subject: %s
    Email Body Snippet: %s
    
    Task: Return ONLY the JSON object with the index of the matched role.
    If the email is generic (e.g. "Update on your application") and doesn't specify a role, return index -1.
    
    Example Output: {"index": 0} or {"index": -1}
    `, titlesList, subject, body)

	// Call LLM
	resp, err := llms.GenerateFromSinglePrompt(ctx, s.Client, prompt)
	if err != nil {
		return -1
	}

	// Parse Response
	var result struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return -1
	}

	// Validate index range
	if result.Index >= 0 && result.Index < len(titles) {
		return result.Index
	}

	return -1
}

// Analysing the Job status for the applied jobs (Ambigous) one
func (s *LLMService) AnalyzeEmailStatus(company, subject, body string) (string, error) {
	ctx := context.Background()

	// 1. Safety Truncation
	// Emails can be huge (chains of replies). We only need the latest context.
	// 3000 characters is usually ~750 tokens, well within limits and enough context.
	if len(body) > 3000 {
		body = body[:3000] + "...(truncated)"
	}

	// 2. The Prompt
	// We use "Few-Shot" formatting instructions to ensure strict JSON.
	prompt := fmt.Sprintf(`
		You are an AI Job Application Tracker. Your goal is to keep the user's database up to date.
		
		CONTEXT:
		The user applied to a job at "%s".
		Current Status in DB: "APPLIED".

		INCOMING EMAIL:
		Subject: %s
		Body: %s

		TASK:
		Analyze the email and determine if the status of the application has changed.
		
		RULES:
		1. If the email is a rejection (e.g., "unfortunately", "not moving forward"), status is "REJECTED".
		2. If the email is an invite to chat, phone screen, or interview, status is "INTERVIEW".
		3. If the email is an offer letter, status is "OFFER".
		4. If the email is just an acknowledgement ("received"), a newsletter, or asking for login details, status is "NO_CHANGE".
		5. If the email is totally unrelated (spam), status is "UNKNOWN".

		OUTPUT FORMAT:
		Return ONLY a valid JSON object. Do not write "Here is the JSON" or use Markdown blocks.
		{
			"status": "REJECTED" | "INTERVIEW" | "OFFER" | "NO_CHANGE" | "UNKNOWN",
			"summary": "A very short, 10-word summary of the email content."
		}
	`, company, subject, body)

	// 3. Call Gemini
	// We use a slightly lower temperature (0.1) to make it more deterministic and factual.
	completion, err := llms.GenerateFromSinglePrompt(ctx, s.Client, prompt, llms.WithTemperature(0.1))
	if err != nil {
		log.Printf("Error calling Gemini LLM: %v", err)
		return "", err
	}

	// 4. Cleaning (Sanitization)
	// Sometimes LLMs wrap JSON in ```json ... ```. We remove that to prevent parsing errors.
	cleaned := cleanJSONOutput(completion)

	return cleaned, nil
}

// Helper to strip Markdown formatting if the LLM adds it
func cleanJSONOutput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "```json")
	input = strings.TrimPrefix(input, "```")
	input = strings.TrimSuffix(input, "```")
	return strings.TrimSpace(input)
}
