package main

import (
	"context"
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/auth"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/database"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/handlers"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/services"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func main() {
	// 1. Load Environment Variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 2. Database Connection
	db := database.Connect()

	// 3. Initialize Core Services (Dependencies)
	llmService := services.NewLLMService()
	jobService := services.NewJobService(db)
	matcherService := services.NewMatcherService(db)

	// 4. Initialize Gmail Integration
	log.Println("Initializing Gmail Client...")

	// Use the auth helper we created to get the authenticated HTTP client
	// This handles the token.json / credentials.json logic automatically
	httpClient := auth.GetGmailClient()

	var gmailService *gmail.Service
	if httpClient != nil {
		ctx := context.Background()
		// Upgrade the HTTP client to a full Gmail Service
		gmailService, err = gmail.NewService(ctx, option.WithHTTPClient(httpClient))
		if err != nil {
			log.Printf("⚠️  Failed to create Gmail Service: %v", err)
		} else {
			log.Println("✅ Gmail Service connected successfully.")
		}
	}

	// 5. Initialize Email Watcher
	// We pass the gmailService (even if nil, the service handles it gracefully)
	emailService := services.NewEmailService(db, llmService, gmailService, matcherService)
	emailService.StartWatcher()

	// 6. Initialize Handlers
	jobHandler := handlers.NewJobHandler(llmService, jobService)

	// 7. Setup Router & CORS
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // For development only
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// 8. Define Routes
	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)

		// Job Routes
		api.POST("/jobs/extract", jobHandler.ParseJob)
		api.POST("/jobs", jobHandler.CreateJob) 
	}

	log.Println("🚀 Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
