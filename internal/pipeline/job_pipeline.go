package pipeline

import (
	"context"
	"fmt"

	// "log"
	"sync"

	"time"

	"github.com/jobs-scraper/internal/models"
	"github.com/jobs-scraper/internal/repo"
)

// JobDescriptionResult represents the result of job description scraping
type JobDescriptionResult struct {
	Job         models.Job
	Description string
	Criteria    map[string]string
	Error       error
}

// JobPipeline manages the job processing pipeline
type JobPipeline struct {
	scraperService *Scraper
	numWorkers     int
	rateLimit      time.Duration
}

// NewJobPipeline creates a new job processing pipeline
func NewJobPipeline(scraperService *Scraper, numWorkers int, rateLimit time.Duration) *JobPipeline {
	return &JobPipeline{
		scraperService: scraperService,
		numWorkers:     numWorkers,
		rateLimit:      rateLimit,
	}
}

// ProcessJobsStreaming processes jobs and job descriptions concurrently
func (p *JobPipeline) ProcessJobsStreaming(ctx context.Context, numPages int, jobRepo *repo.JobRepository, jobDescRepo *repo.JobDescriptionRepository, params models.SearchQuery) error {
	// Create channels for the pipeline
	allJobs := make([]models.Job, 0, 100)
	allJobDescriptions := make([]models.JobDescription, 0, 100)
	var jbMu sync.Mutex
	jobsChan := GetJobs(ctx, p.scraperService)
	jobWithDescriptionChan := GetJobDescription(ctx, p.scraperService, jobsChan, 3)

	for jobWithDescription := range jobWithDescriptionChan {
		fmt.Printf("Received job description for job : %d\n", jobWithDescription.Job.ID)
		jbMu.Lock()
		allJobs = append(allJobs, jobWithDescription.Job)
		allJobDescriptions = append(allJobDescriptions, jobWithDescription.JobDescription)
		jbMu.Unlock()
	}

	if err := jobRepo.SaveJobs(allJobs); err != nil {
		return fmt.Errorf("failed to save jobs to database: %w", err)
	}

	if err := jobDescRepo.SaveJobDescriptions(allJobDescriptions); err != nil {
		return fmt.Errorf("failed to save job descriptions: %w", err)
	}

	return nil
}
