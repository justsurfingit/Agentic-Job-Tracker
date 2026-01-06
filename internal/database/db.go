package database

import (
	"log"

	"github.com/justsurfingit/Agentic-Job-Tracker/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() *gorm.DB {
	// In a real company, these values come from os.Getenv() (Environment Variables)
	// For now, we hardcode for local dev, but we will fix this in Sprint 2.
	dsn := "host=localhost user=postgres password=password dbname=jobtracker port=5432 sslmode=disable"

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connection established")

	// Migration: This creates the tables in Postgres automatically
	log.Println("Running Migrations...")
	DB.AutoMigrate(&models.Company{}, &models.Job{}, &models.JobEvent{}, &models.User{}, &models.ProcessedEmail{})
	return DB
}
