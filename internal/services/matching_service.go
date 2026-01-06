package services

import (
	"net/mail"
	"strings"

	"github.com/justsurfingit/Agentic-Job-Tracker/internal/models"
	"gorm.io/gorm"
)

type MatcherService struct {
	DB *gorm.DB
}

func NewMatcherService(db *gorm.DB) *MatcherService {
	return &MatcherService{DB: db}
}

// Approach-
// Regex for filtering
// Then put the potential mail to LLM to extract the relevant information regarding the process.

// FindCompanyFromEmail tries to match an email to a tracked Company
func (s *MatcherService) FindCompanyFromEmail(subject, rawSender string) *models.Company {
	// 1. Parse the sender header to get "Display Name" and "Address"
	// e.g. "Stripe Recruiting <jobs@stripe.com>" -> name="Stripe Recruiting", addr="jobs@stripe.com"
	parsedAddr, err := mail.ParseAddress(rawSender)
	senderName := ""
	senderAddr := ""
	if err == nil {
		senderName = strings.ToLower(parsedAddr.Name)
		senderAddr = strings.ToLower(parsedAddr.Address)
	} else {
		senderAddr = strings.ToLower(rawSender) // Fallback if parsing fails
	}

	subjectLower := strings.ToLower(subject)

	// 2. Fetch all companies
	// TODO:(Optimization: Cache this map for O(1) lookups in future)
	var companies []models.Company
	s.DB.Find(&companies)
	for _, company := range companies {
		companyName := strings.ToLower(company.Name)
		// SAFETY CHECK: Skip very short names to avoid false positives.
		// e.g. If company is named "X" or "Go", it matches everything.
		if len(companyName) < 3 {
			continue
		}

		// --- RULE 1: Subject Line Match ---
		// Does "Update on your application to Stripe" contain "Stripe"?
		if strings.Contains(subjectLower, companyName) {
			return &company
		}

		// --- RULE 2: Sender Display Name Match ---
		// Does "Stripe Recruiting" contain "Stripe"?
		if senderName != "" && strings.Contains(senderName, companyName) {
			return &company
		}

		// --- RULE 3: Sender Domain Match ---
		// Does "jobs@stripe.com" contain "stripe"?
		// We only check the part AFTER the '@' to be safe
		parts := strings.Split(senderAddr, "@")
		if len(parts) == 2 {
			domain := parts[1] // "stripe.com"
			if strings.Contains(domain, companyName) {
				return &company
			}
		}
	}

	return nil
}

// func cleanUrl(url string) string {
// 	// Simple helper: https://www.google.com/careers -> google.com
// 	url = strings.ReplaceAll(url, "https://", "")
// 	url = strings.ReplaceAll(url, "http://", "")
// 	url = strings.ReplaceAll(url, "www.", "")
// 	parts := strings.Split(url, "/")
// 	if len(parts) > 0 {
// 		return parts[0]
// 	}
// 	return ""
// }
