package pipeline

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jobs-scraper/internal/models"
	"github.com/jobs-scraper/internal/utils"
)

type Config struct {
	Timespan       string        // Time filter for job postings (e.g., "r86400" for last 24 hours)
	Distance       string        // Search radius in miles (e.g., "25")
	SortBy         string        // Sort results by (R for relevance, DD for date posted)
	MaxRetries     int           // Maximum number of retries for failed requests
	BaseDelay      time.Duration // Base delay for exponential backoff
	MaxDelay       time.Duration // Maximum delay between retries
	RequestTimeout time.Duration // Timeout for individual HTTP requests
}

type Scraper struct {
	config Config
}

func NewScraper(config Config) *Scraper {
	// r86400 last 24 hours
	// r604800 last week
	// r2592000 last month
	// r7776000 last 3 months
	// r31536000 last year
	if config.Timespan == "" {
		config.Timespan = "r604800" // Default to last week
	}

	// Set default retry configuration
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.BaseDelay == 0 {
		config.BaseDelay = 1 * time.Second
	}
	if config.MaxDelay == 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}

	return &Scraper{
		config: config,
	}
}

// ScrapeLinkedInJobsStreaming scrapes jobs page by page and sends them to channel immediately
func (s *Scraper) ScrapeLinkedInJobsStreaming(ctx context.Context, numPages int, jobChan chan<- models.Job, params models.SearchQuery) error {
	// Process pages sequentially to send jobs immediately
	for i := range numPages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			jobs, err := s.ScrapeJobsWithContext(ctx, i, params)
			if err != nil {
				return fmt.Errorf("error scraping page %d: %w", i, err)
			}

			// Send jobs to channel immediately
			for _, job := range jobs {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case jobChan <- job:
				}
			}
		}
	}
	return nil
}

func (s *Scraper) ScrapeJobsWithContext(ctx context.Context, page int, params models.SearchQuery) ([]models.Job, error) {
	jobs := make([]models.Job, 0, 10)
	url := s.buildSearchURL(params, page)

	retryableRequest := utils.NewRetryableHTTPRequest(utils.RetryConfig{
		BaseDelay:  s.config.BaseDelay,
		MaxDelay:   s.config.MaxDelay,
		MaxRetries: s.config.MaxRetries,
	})

	res, err := retryableRequest.RetryableHTTPRequest(ctx, url, "GET", nil, nil)
	if err != nil {
		fmt.Printf("Error fetching URL after retries: %v\n", err)
		return jobs, err
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return jobs, err
	}

	doc.Find("li > div.base-card").Each(func(i int, s *goquery.Selection) {
		job := models.Job{}
		title := s.Find("[class*=_title]").Text()
		job.Title = strings.TrimSpace(title)
		job.Company = strings.TrimSpace(s.Find(".hidden-nested-link").Text())
		job.CompanyLink = strings.TrimSpace(s.Find(".hidden-nested-link").AttrOr("href", ""))
		job.Location = strings.TrimSpace(s.Find(".job-search-card__location").Text())
		job.JobLink = strings.TrimSpace(s.Find("a.base-card__full-link").AttrOr("href", ""))
		jobId, err := extractJobIDFromURL(job.JobLink)
		if err != nil {
			fmt.Printf("Error extracting job ID from URL %s: %v\n", job.JobLink, err)
		}
		jobIdInt, err := strconv.ParseInt(jobId, 10, 64)

		if err != nil {
			fmt.Printf("Error extracting job ID from URL %s: %v\n", job.JobLink, err)
		}
		job.ID = jobIdInt

		if job.Title != "" && job.Company != "" && job.Location != "" && job.JobLink != "" {
			jobs = append(jobs, job)
		}
	})

	return jobs, nil
}

func (s *Scraper) ScrapeJobDescriptionWithContext(ctx context.Context, job models.Job) (string, map[string]string, error) {
	var jobDescription string
	url := s.buildJobDescriptionSearchURL(job.JobLink)

	retryableRequest := utils.NewRetryableHTTPRequest(utils.RetryConfig{
		MaxRetries: s.config.MaxRetries,
		BaseDelay:  s.config.BaseDelay,
		MaxDelay:   s.config.MaxDelay,
	})

	res, err := retryableRequest.RetryableHTTPRequest(ctx, url, "GET", nil, nil)
	if err != nil {
		fmt.Printf("Error fetching job description URL after retries: %v\n", err)
		return "", map[string]string{}, err
	}

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return "", map[string]string{}, err
	}

	// Find the job details section directly by ID and extract its text content
	jobDescription, jobCriteria, err := s.parseJobDescription(doc)
	if err != nil {
		return "", map[string]string{}, err
	}

	return strings.TrimSpace(jobDescription), jobCriteria, nil
}

func (s *Scraper) buildSearchURL(query models.SearchQuery, page int) string {
	baseURL := "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search"
	params := url.Values{}

	// Required search parameters
	params.Set("keywords", url.QueryEscape(query.Keywords))
	params.Set("location", url.QueryEscape(query.Location))

	// Time filter
	// params.Set("f_TPR", s.config.Timespan)

	// Work type filter
	if query.FWT != "" {
		params.Set("f_WT", query.FWT)
	}

	// Pagination
	params.Set("start", strconv.Itoa(25*page))

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func (s *Scraper) buildJobDescriptionSearchURL(jobLink string) string {
	// For job descriptions, we should use the direct job link
	return strings.ReplaceAll(jobLink, "jp.linkedin.com", "linkedin.com")
}

func (s *Scraper) parseJobDescription(doc *goquery.Document) (string, map[string]string, error) {
	var jobDescription string
	jobCriteria := make(map[string]string)

	// Find the main description section
	descriptionSection := doc.Find("section.core-section-container.description .core-section-container__content")

	if descriptionSection.Length() == 0 {
		// Alternative selector if classes are different
		descriptionSection = doc.Find("section.description .core-section-container__content")
	}

	// Parse the main job description from show-more-less-html section
	descriptionHTML := descriptionSection.Find("section.show-more-less-html .show-more-less-html__markup")

	if descriptionHTML.Length() > 0 {
		jobDescription += parseHTMLContent(descriptionHTML)
	}

	// Parse job criteria list
	criteriaList := descriptionSection.Find("ul.description__job-criteria-list")

	if criteriaList.Length() > 0 {
		criteriaList.Find("li.description__job-criteria-item").Each(func(i int, li *goquery.Selection) {
			// Get the criteria header
			header := strings.TrimSpace(li.Find("h3.description__job-criteria-subheader").Text())

			// Get the criteria value
			value := strings.TrimSpace(li.Find("span.description__job-criteria-text").Text())

			if header != "" && value != "" {
				jobCriteria[header] = value
			}
		})
	}

	// Clean up the description
	jobDescription = strings.TrimSpace(jobDescription)

	if jobDescription == "" {
		return "", jobCriteria, fmt.Errorf("job description not found")
	}

	return jobDescription, jobCriteria, nil
}

// parseHTMLContent walks through HTML elements in order and preserves formatting
func parseHTMLContent(selection *goquery.Selection) string {
	var result strings.Builder

	// Walk through all child nodes in order
	selection.Contents().Each(func(i int, s *goquery.Selection) {
		if goquery.NodeName(s) == "#text" {
			// Handle text nodes
			text := strings.TrimSpace(s.Text())
			if text != "" {
				result.WriteString(text)
			}
		} else {
			// Handle element nodes
			tagName := goquery.NodeName(s)
			text := strings.TrimSpace(s.Text())

			if text == "" {
				return
			}

			switch tagName {
			case "p":
				result.WriteString(text)
				result.WriteString("\n\n")
			case "h1", "h2", "h3", "h4", "h5", "h6":
				result.WriteString(text)
				result.WriteString("\n\n")
			case "strong", "b":
				result.WriteString("**")
				result.WriteString(text)
				result.WriteString("**")
			case "em", "i":
				result.WriteString("*")
				result.WriteString(text)
				result.WriteString("*")
			case "li":
				result.WriteString("• ")
				result.WriteString(text)
				result.WriteString("\n")
			case "ul", "ol":
				// Process list items recursively
				s.Find("li").Each(func(j int, li *goquery.Selection) {
					liText := strings.TrimSpace(li.Text())
					if liText != "" {
						result.WriteString("• ")
						result.WriteString(liText)
						result.WriteString("\n")
					}
				})
				result.WriteString("\n")
			case "br":
				result.WriteString("\n")
			case "div", "span":
				// For div and span, recursively parse content
				if s.Children().Length() > 0 {
					result.WriteString(parseHTMLContent(s))
				} else {
					result.WriteString(text)
					result.WriteString(" ")
				}
			default:
				// For other tags, just extract text content
				result.WriteString(text)
				result.WriteString(" ")
			}
		}
	})

	return result.String()
}

func extractJobIDFromURL(jobURL string) (string, error) {
	parsed, err := url.Parse(jobURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Get the last segment of the path
	lastSegment := path.Base(parsed.Path)

	// Extract job ID from the end of the segment using regex
	// Look for a sequence of digits at the end of the string
	re := regexp.MustCompile(`(\d+)$`)
	matches := re.FindStringSubmatch(lastSegment)

	if len(matches) < 2 {
		return "", fmt.Errorf("job ID not found in URL path: %s", jobURL)
	}

	return matches[1], nil
}
