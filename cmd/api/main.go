package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/database"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/handlers"
	"github.com/justsurfingit/Agentic-Job-Tracker/internal/services"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	//database connection
	db := database.Connect()
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true // For development only.
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))
	// Dependency Ingestion
	llmService := services.NewLLMService()
	jobService := services.NewJobService(db)
	jobHandler := handlers.NewJobHandler(llmService, jobService)

	api := r.Group("/api/v1")
	{
		api.GET("/health", handlers.HealthCheck)
		api.POST("/jobs/extract", jobHandler.ParseJob)
		api.POST("job", jobHandler.CreateJob)
	}
	log.Println("Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
