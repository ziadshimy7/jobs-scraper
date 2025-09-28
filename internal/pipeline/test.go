package pipeline

import (
	"context"
	"log"
	"sync"

	"github.com/jobs-scraper/internal/models"
)

func GetJobs(context context.Context, scraperService *Scraper) <-chan models.Job {
	jobChan := make(chan models.Job, 100)

	go func() {
		defer close(jobChan)
		if err := scraperService.ScrapeLinkedInJobsStreaming(context, 10, jobChan, models.SearchQuery{
			Keywords: "Frontend Developer",
			Location: "Japan",
			FWT:      "2,3",
		}); err != nil {
			log.Printf("Error scraping jobs: %v", err)
		}
	}()

	return jobChan
}

func GetJobDescription(context context.Context, scraperService *Scraper, jobChan <-chan models.Job) <-chan models.JobWithDescription {
	jobDescriptionChan := make(chan models.JobWithDescription, 100)
	const numWorkers = 3

	var wg sync.WaitGroup

	// Start 3 worker goroutines
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				select {
				case <-context.Done():
					return
				default:
				}

				// Scrape job description
				if jd, jc, err := scraperService.ScrapeJobDescriptionWithContext(context, job); err != nil {
					log.Printf("Error scraping jobs: %v", err)
				} else {
					jobDescriptionChan <- models.JobWithDescription{
						Job: job,
						JobDescription: models.JobDescription{
							JobID:       job.ID,
							Description: jd,
							Criteria:    jc,
						},
					}
				}
			}
		}()
	}

	// Close the output channel when all workers are done
	go func() {
		wg.Wait()
		close(jobDescriptionChan)
	}()

	return jobDescriptionChan
}
