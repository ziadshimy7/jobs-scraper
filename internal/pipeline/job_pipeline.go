package pipeline

import (
	"context"
	"fmt"

	// "log"
	"sync"

	"time"

	"github.com/jobs-scraper/internal/models"
	"github.com/jobs-scraper/internal/repo"
	"golang.org/x/time/rate"
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
	// var mu sync.Mutex
	var jbMu sync.Mutex
	jobsChan := GetJobs(ctx, p.scraperService)
	jobWithDescriptionChan := GetJobDescription(ctx, p.scraperService, jobsChan)

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

// jobDescriptionWorker processes jobs from jobChan and sends results to jobDescriptionChan
func (p *JobPipeline) jobDescriptionWorker(ctx context.Context, jobChan <-chan models.Job, resultChan chan<- JobDescriptionResult) {

	limiter := rate.NewLimiter(rate.Every(2*time.Second), 1) // 1 request per second

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobChan:
			if !ok {
				return
			}

			if err := limiter.Wait(ctx); err != nil {
				return
			}

			fmt.Printf("Processing job: %s\n", job.Title)

			description, criteria, err := p.scraperService.ScrapeJobDescriptionWithContext(ctx, job)

			result := JobDescriptionResult{
				Job:         job,
				Description: description,
				Criteria:    criteria,
				Error:       err,
			}

			select {
			case resultChan <- result:
			case <-ctx.Done():
				return
			}

		}
	}
}
