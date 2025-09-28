package main

import (
	"context"
	"log"
	"time"

	"github.com/jobs-scraper/infrastructure"
	"github.com/jobs-scraper/internal/models"
	"github.com/jobs-scraper/internal/pipeline"
	"github.com/jobs-scraper/internal/repo"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	dbConfig := infrastructure.LoadConfigFromEnv()
	db, err := infrastructure.NewConnection(dbConfig)
	if err != nil {
		log.Fatal("Error connecting to db")
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging db")
	}

	log.Println("Successfully connected to db")

	// Run database migrations
	if err := infrastructure.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	scraper := pipeline.NewScraper(pipeline.Config{
		Distance:       "25",
		SortBy:         "R",
		MaxRetries:     3,
		BaseDelay:      1 * time.Second,
		MaxDelay:       30 * time.Second,
		RequestTimeout: 30 * time.Second,
	})

	jobRepo := repo.NewJobRepository(db)
	jobDescriptionRepo := repo.NewJobDescriptionRepository(db)

	jobPipeline := pipeline.NewJobPipeline(scraper, 5, 1*time.Second) // 5 workers, 1 second rate limit

	ctx := context.Background()

	searchParams := models.SearchQuery{
		Keywords: "Frontend Developer",
		Location: "Japan",
		FWT:      "2,3",
	}

	err = jobPipeline.ProcessJobsStreaming(ctx, 10, jobRepo, jobDescriptionRepo, searchParams)
	if err != nil {
		log.Fatalf("Pipeline processing failed: %v", err)
	}

	log.Println("Jobs inserted successfully")
}
