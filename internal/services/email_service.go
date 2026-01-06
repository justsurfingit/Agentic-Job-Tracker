package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/justsurfingit/Agentic-Job-Tracker/internal/models"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
	"gorm.io/gorm"
)

type EmailService struct {
	DB             *gorm.DB
	LLMService     *LLMService
	MatcherService *MatcherService
	GmailClient    *gmail.Service
}

func NewEmailService(db *gorm.DB, llm *LLMService, gmail *gmail.Service, matcher *MatcherService) *EmailService {
	return &EmailService{
		DB:             db,
		LLMService:     llm,
		GmailClient:    gmail,
		MatcherService: matcher,
	}
}

// StartWatcher starts the background polling
func (s *EmailService) StartWatcher() {
	if s.GmailClient == nil {
		log.Println("⚠️ Gmail Watcher disabled (no client). Check credentials.")
		return
	}

	// Ticker triggers every 15 minutes
	ticker := time.NewTicker(1 * time.Minute)

	// Run immediately on startup
	go s.SyncEmails()

	go func() {
		for range ticker.C {
			s.SyncEmails()
		}
	}()
}

// SyncEmails is the main orchestrator
func (s *EmailService) SyncEmails() {
	// 1. Timeout Context: Prevent hanging forever (2 minute limit)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("📧 Email Watcher: Starting Sync Cycle...")

	// 2. Get User State
	var user models.User
	if err := s.DB.First(&user).Error; err != nil {
		// Create default user if missing
		user = models.User{Email: "default", LastHistoryID: 0}
		s.DB.Create(&user)
	}

	var messages []*gmail.Message
	var newHistoryID uint64
	var err error

	// 3. Decide Strategy: Bootstrap (Full) or Incremental
	if user.LastHistoryID == 0 {
		log.Println("🆕 First run detected. Running Full Bootstrap Sync...")
		messages, newHistoryID, err = s.performFullSync(ctx)
	} else {
		// Try Incremental Sync
		messages, newHistoryID, err = s.performIncrementalSync(ctx, user.LastHistoryID)

		// Handle "History Expired" (404) - Google deleted old history
		if err != nil && isHistoryExpiredError(err) {
			log.Println("⚠️ History ID expired (too old). Falling back to Full Sync.")
			messages, newHistoryID, err = s.performFullSync(ctx)
		} else if err != nil {
			log.Printf("❌ Sync failed: %v", err)
			return
		}
	}

	if len(messages) == 0 {
		log.Println("✅ No new relevant emails found.")
		// Update History ID even if empty, so we don't check this window again
		if newHistoryID > user.LastHistoryID {
			s.updateUserHistoryID(user.ID, newHistoryID)
		}
		return
	}

	log.Printf("📥 Processing %d candidate emails...", len(messages))

	// 4. Process Messages with Deduplication
	for _, msg := range messages {
		// A. Check Dedup Table
		var count int64
		s.DB.Model(&models.ProcessedEmail{}).Where("id = ?", msg.Id).Count(&count)
		if count > 0 {
			continue // Already processed, skip
		}

		// B. Process the Email (Core Logic)
		s.processSingleEmail(ctx, msg)

		// C. Mark as Processed
		s.DB.Create(&models.ProcessedEmail{ID: msg.Id})
	}

	// 5. Update Bookmark (Save State)
	if newHistoryID > user.LastHistoryID {
		s.updateUserHistoryID(user.ID, newHistoryID)
		log.Printf("🔖 History updated to %d", newHistoryID)
	}
}

// performFullSync scans the last 7 days and resets the History ID (Bootstrap)
func (s *EmailService) performFullSync(ctx context.Context) ([]*gmail.Message, uint64, error) {
	var resp *gmail.ListMessagesResponse

	// Query: Relevant keywords + newer than 7 days
	q := "subject:(application OR interview OR update OR offer OR rejected OR status) newer_than:7d"

	err := retry(3, 1*time.Second, func() error {
		var e error
		call := s.GmailClient.Users.Messages.List("me").Q(q).MaxResults(50)
		resp, e = call.Context(ctx).Do()
		return e
	})

	if err != nil {
		return nil, 0, err
	}

	// Get the CURRENT History ID from Profile to set our new anchor
	profile, err := s.GmailClient.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		return nil, 0, err
	}

	// We must fetch full message details for the list we found
	fullMessages := s.expandMessages(ctx, resp.Messages)

	return fullMessages, profile.HistoryId, nil
}

// performIncrementalSync asks Google ONLY for what changed since startID
func (s *EmailService) performIncrementalSync(ctx context.Context, startID uint64) ([]*gmail.Message, uint64, error) {
	var resp *gmail.ListHistoryResponse

	err := retry(3, 1*time.Second, func() error {
		var e error
		call := s.GmailClient.Users.History.List("me").StartHistoryId(startID)
		// We only care about added messages, not label changes
		call.HistoryTypes("messageAdded")
		resp, e = call.Context(ctx).Do()
		return e
	})

	if err != nil {
		return nil, 0, err
	}

	// Extract all added messages from the history events
	var msgHeaders []*gmail.Message
	for _, h := range resp.History {
		for _, mAdded := range h.MessagesAdded {
			if mAdded.Message != nil {
				msgHeaders = append(msgHeaders, mAdded.Message)
			}
		}
	}

	fullMessages := s.expandMessages(ctx, msgHeaders)

	return fullMessages, resp.HistoryId, nil
}

// expandMessages takes a list of IDs and fetches the full body/headers
func (s *EmailService) expandMessages(ctx context.Context, headers []*gmail.Message) []*gmail.Message {
	var fullMessages []*gmail.Message
	for _, h := range headers {
		// Retry individual message fetches
		retry(2, 500*time.Millisecond, func() error {
			msg, err := s.GmailClient.Users.Messages.Get("me", h.Id).Context(ctx).Do()
			if err == nil {
				fullMessages = append(fullMessages, msg)
			}
			return err
		})
	}
	return fullMessages
}

// processSingleEmail contains the Business Logic (Matching -> LLM -> DB)
func (s *EmailService) processSingleEmail(ctx context.Context, msg *gmail.Message) {
	headers := parseHeaders(msg)
	subject := headers["Subject"]
	sender := headers["From"]

	// Create a short log prefix so we can track this specific email in the logs
	// e.g. "[Email: Update on application...]"
	shortSub := subject
	if len(shortSub) > 20 {
		shortSub = shortSub[:20] + "..."
	}
	logPrefix := fmt.Sprintf("[Email: %s]", shortSub)

	log.Printf("%s 📥 START processing from: %s", logPrefix, sender)

	// Extract body
	body := getEmailBody(msg)

	// --- STEP 1: MATCHING ---
	company := s.MatcherService.FindCompanyFromEmail(subject, sender)
	if company == nil {
		log.Printf("%s ❌ SKIPPED: Company match failed. Sender/Subject not in DB.", logPrefix)
		return
	}
	log.Printf("%s ✅ MATCHED Company: %s", logPrefix, company.Name)

	// --- STEP 2: FIND TARGET JOB ---
	var jobs []models.Job
	// Ignore terminal states
	s.DB.Where("company_id = ? AND status NOT IN ('REJECTED', 'OFFER')", company.ID).Find(&jobs)

	if len(jobs) == 0 {
		log.Printf("%s ❌ SKIPPED: No active 'APPLIED' jobs found for %s in DB.", logPrefix, company.Name)
		return
	}

	var targetJob *models.Job
	if len(jobs) == 1 {
		targetJob = &jobs[0]
		log.Printf("%s 🎯 Auto-linked to single active job: %s", logPrefix, targetJob.Title)
	} else {
		// Disambiguate with AI
		var jobTitles []string
		for _, j := range jobs {
			jobTitles = append(jobTitles, j.Title)
		}

		log.Printf("%s ⚠️ Ambiguous: Found %d jobs (%v). Asking LLM to pick...", logPrefix, len(jobs), jobTitles)
		bestMatchIndex := s.LLMService.IdentifyJobRole(jobTitles, subject, body)

		if bestMatchIndex != -1 {
			targetJob = &jobs[bestMatchIndex]
			log.Printf("%s 🎯 LLM selected job: %s", logPrefix, targetJob.Title)
		} else {
			log.Printf("%s ❌ SKIPPED: LLM could not determine which job this email is about.", logPrefix)
			return
		}
	}

	// --- STEP 3: ANALYZE STATUS ---
	log.Printf("%s 🤖 Analyzing content with LLM...", logPrefix)
	analysisJSON, err := s.LLMService.AnalyzeEmailStatus(company.Name, subject, body)
	if err != nil {
		log.Printf("%s ❌ SKIPPED: LLM Analysis Error: %v", logPrefix, err)
		return
	}

	var result struct {
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(analysisJSON), &result); err != nil {
		log.Printf("%s ❌ SKIPPED: JSON Parse Error: %v. Raw: %s", logPrefix, err, analysisJSON)
		return
	}

	log.Printf("%s 🧠 LLM Decision: Status=%s | Summary=%s", logPrefix, result.Status, result.Summary)

	// --- STEP 4: UPDATE DB ---
	if result.Status == "NO_CHANGE" || result.Status == "UNKNOWN" {
		log.Printf("%s ⏹️  No DB Update needed (Status is %s).", logPrefix, result.Status)
		return
	}

	if result.Status == targetJob.Status {
		log.Printf("%s ⏹️  Status is already %s. Ignoring.", logPrefix, result.Status)
		return
	}

	// EXECUTE UPDATE
	log.Printf("%s ⚡ UPDATING DB: %s -> %s", logPrefix, targetJob.Status, result.Status)
	s.DB.Model(targetJob).Updates(map[string]interface{}{
		"status": result.Status,
	})

	event := models.JobEvent{
		JobID:     targetJob.ID,
		EventType: "EMAIL_UPDATE",
		Details:   fmt.Sprintf("Status changed to %s. Summary: %s", result.Status, result.Summary),
	}
	s.DB.Create(&event)
	log.Printf("%s ✅ Success! Event logged.", logPrefix)
}

// --- HELPERS ---

// retry executes a function with exponential backoff
func retry(attempts int, sleep time.Duration, f func() error) error {
	for i := 0; i < attempts; i++ {
		err := f()
		if err == nil {
			return nil
		}
		// If 404 (History Expired), fail fast so we can switch to Full Sync
		if isHistoryExpiredError(err) {
			return err
		}

		log.Printf("⚠️ API Error: %v. Retrying in %v...", err, sleep)
		time.Sleep(sleep)
		sleep *= 2
	}
	return fmt.Errorf("failed after %d attempts", attempts)
}

func isHistoryExpiredError(err error) bool {
	if gErr, ok := err.(*googleapi.Error); ok {
		return gErr.Code == 404
	}
	return false
}

func (s *EmailService) updateUserHistoryID(userID uint, newID uint64) {
	s.DB.Model(&models.User{}).Where("id = ?", userID).Update("last_history_id", newID)
}

func parseHeaders(msg *gmail.Message) map[string]string {
	res := make(map[string]string)
	for _, h := range msg.Payload.Headers {
		res[h.Name] = h.Value
	}
	return res
}

func getEmailBody(msg *gmail.Message) string {
	if msg.Payload.Body != nil && msg.Payload.Body.Data != "" {
		d, _ := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
		return string(d)
	}
	for _, part := range msg.Payload.Parts {
		if part.MimeType == "text/plain" && part.Body.Data != "" {
			d, _ := base64.URLEncoding.DecodeString(part.Body.Data)
			return string(d)
		}
	}
	for _, part := range msg.Payload.Parts {
		if part.MimeType == "text/html" && part.Body.Data != "" {
			d, _ := base64.URLEncoding.DecodeString(part.Body.Data)
			return string(d)
		}
	}
	return ""
}
