package main

import (
	"log"
	"os"

	"github.com/jobs-scraper/infrastructure"
	"github.com/jobs-scraper/internal/models"
	"github.com/jobs-scraper/internal/repo"
	"github.com/jobs-scraper/internal/services"
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

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	model := os.Getenv("CV_AI_MODEL")

	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging db")
	}

	log.Println("Successfully connected to db")

	// Run database migrations
	if err := infrastructure.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database migrations completed successfully")

	jobRepo := repo.NewJobRepository(db)
	jobDescriptionRepo := repo.NewJobDescriptionRepository(db)
	openRouterService := services.NewOpenRouterService(model, apiKey)

	cv, err := os.ReadFile("../cv.txt")

	if err != nil {
		log.Fatalf("Failed to get cv: %v", err)
	}

	job, err := jobRepo.GetJobByID(4306471753)

	if err != nil {
		log.Fatalf("Failed to get job: %v", err)
	}

	jobDescription, jobCriteria, err := jobDescriptionRepo.GetJobDescriptionByJobID(job.ID)

	if err != nil {
		log.Fatalf("Failed to get job description: %v", err)
	}

	result, err := openRouterService.AnalyzeJobDescription(string(cv), models.JobDescription{
		JobID:       job.ID,
		Description: jobDescription,
		Criteria:    jobCriteria,
	})

	if err != nil {
		log.Fatalf("Failed to get job analysis result: %v", err)
	}

	log.Println(result)
	// job,err :=
}
