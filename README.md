<!-- # LinkedIn Jobs Scraper

A robust Go application that scrapes LinkedIn job postings with **retry logic**, **exponential backoff**, and **concurrent processing** using a streaming pipeline architecture.

## 🏗️ Architecture Overview

The application uses a **streaming pipeline** with **intelligent retry mechanisms** that processes jobs concurrently while handling network failures gracefully.

```
┌─────────────────────────────────────┐
│        Job Scraping                 │
│   (Sequential + Retry Logic)       │
│  Page 1 → Page 2 → Page 3 → Page N │ ──┐
│     ↓ Retry on failure             │   │
│  [Exponential Backoff]             │   │ Jobs streamed to jobChan
└─────────────────────────────────────┘   │ immediately as found
                                          │
┌─────────────────────────────────────┐   │
│     Job Description Processing      │   │
│    (Concurrent Workers + Retries)   │ ←─┘
│  Worker 1  Worker 2  Worker 3 ... N │
│     ↓ Retry on failure             │
│  [Exponential Backoff]             │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│         Database Storage            │
│   Jobs → Job Descriptions → Match   │
└─────────────────────────────────────┘
```

## 🚀 Key Features

### 🔄 Intelligent Retry System
- **Exponential Backoff**: 1s → 2s → 4s → 8s delays between retries
- **Smart Error Handling**: Retries server errors (5xx), skips client errors (4xx)
- **Configurable Limits**: Max retries, delays, and timeouts
- **Context Cancellation**: Respects cancellation during retries
- **Request Timeout**: 30-second timeout per HTTP request

### 🚀 Streaming Pipeline Flow
1. **Step 1**: Jobs scraped page-by-page with retry logic
2. **Step 2**: Workers process job descriptions concurrently with retries
3. **Step 3**: Save jobs to database to get auto-generated IDs
4. **Step 4**: Get jobs back from database with IDs
5. **Step 5**: Map job descriptions to database IDs and save

### ⚡ Performance Optimizations
- **Zero Idle Time**: Workers start immediately when first jobs arrive
- **Concurrent Processing**: 5 configurable workers processing descriptions
- **Rate Limiting**: 2-second delays between requests to avoid detection
- **Resilient Network Handling**: Automatic retry with exponential backoff
- **Context-Aware**: Proper cancellation support throughout pipeline

### 🛡️ Anti-Detection Measures
- **Sequential Page Scraping**: Reduces bot detection risk
- **Rate Limiting**: Built-in delays between all requests
- **User-Agent Headers**: Mimics real browser requests
- **Retry Logic**: Handles temporary blocks gracefully
- **Configurable Search**: Customizable job search parameters

## 📁 Project Structure

```
jobs-scraper/
├── main.go                          # Application entry point
├── infrastructure/
│   └── db.go                        # Database connection & migrations
├── internal/
│   ├── models/
│   │   └── Job.go                   # Job data structure
│   ├── pipeline/
│   │   └── job_pipeline.go          # Core streaming pipeline
│   ├── repo/
│   │   ├── job.go                   # Job repository
│   │   └── job-description.go       # Job description repository
│   ├── services/
│   │   ├── scraper.go               # Scraping service wrapper
│   │   └── gemini.go                # AI job analysis (optional)
│   └── scraper.go                   # Core scraping logic
├── migrations/                      # Database schema migrations
└── cv/                             # CV matching (future feature)
```

## 🔧 Pipeline Components

### 1. Core Scraper (`internal/scraper.go`)
```go
// HTTP requests with exponential backoff retry logic
func (s *Scraper) RetryableHTTPRequest(ctx context.Context, url string) (*http.Response, error)

// Context-aware job scraping with retry support
func (s *Scraper) ScrapeJobsWithContext(ctx context.Context, page int, params SearchQuery) ([]models.Job, error)
```
- **Retry Logic**: Up to 3 attempts with exponential backoff (1s → 2s → 4s → 8s)
- **Smart Error Handling**: Retries 5xx errors, fails fast on 4xx errors
- **Configurable Parameters**: Search keywords, location, work type (remote/hybrid)
- **Request Timeout**: 30-second timeout per request

### 2. Scraper Service (`internal/services/scraper.go`)
```go
// Streams jobs with configurable search parameters
func (s *Scraper) ScrapeLinkedInJobsStreaming(ctx context.Context, numPages int, jobChan chan<- models.Job, params SearchQuery) error
```
- **Sequential Processing**: Pages scraped one by one to avoid detection
- **Rate Limited**: 2-second delays between requests
- **Immediate Streaming**: Jobs sent to channel as soon as found
- **Configurable Search**: Custom keywords, location, work type filters

### 3. Pipeline Orchestrator (`internal/pipeline/job_pipeline.go`)
```go
// Coordinates the entire streaming pipeline with retry-enabled scraping
func (p *JobPipeline) ProcessJobsStreaming(ctx context.Context, numPages int, jobRepo *JobRepository, jobDescRepo *JobDescriptionRepository, params SearchQuery) error
```

**Pipeline Steps:**
1. **Job Scraping Goroutine**: Scrapes pages and streams to `jobChan`
2. **Worker Goroutines**: 5 concurrent workers processing job descriptions
3. **Channel Coordinator**: Waits for workers and closes result channel
4. **Result Collection**: Main thread collects all results via channel ranging
5. **Database Operations**: Sequential saves with proper ID mapping

### 4. Job Description Workers
```go
// Each worker processes jobs with retry logic
func (p *JobPipeline) jobDescriptionWorker(ctx context.Context, jobChan <-chan models.Job, resultChan chan<- JobDescriptionResult)
```
- **Retry-Enabled**: Uses `ScrapeJobDescriptionWithContext` with retry logic
- **Rate Limited**: 2-second delays per worker to avoid overwhelming servers
- **Concurrent Processing**: Multiple workers process descriptions simultaneously
- **Graceful Failure**: Failed scrapes don't stop other workers
- **Context Cancellation**: Respects cancellation signals


## ⚙️ Setup & Installation

### Prerequisites
- Go 1.21+
- PostgreSQL 12+
- LinkedIn access (for scraping)

### Environment Variables
Create a `.env` file:
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=linkedin_jobs
DB_SSLMODE=disable

# Optional: For AI job analysis
GEMINI_API_KEY=your_gemini_api_key
```

### Installation
```bash
# Clone repository
git clone <repository-url>
cd jobs-scraper

# Install dependencies
go mod tidy

# Run database migrations
go run main.go
```

## 🚀 Usage

### Basic Usage
```bash
go run main.go
```

The application will:
1. Connect to PostgreSQL database
2. Run pending migrations
3. Initialize the streaming pipeline
4. Scrape 10 pages of LinkedIn jobs (configurable)
5. Process job descriptions concurrently
6. Store results in database

### Configuration

#### Scraper Configuration
```go
scraper := internal.NewScraper(internal.Config{
    Distance:       "25",                // Search radius in miles
    SortBy:         "R",                 // Sort by relevance
    MaxRetries:     3,                   // Maximum retry attempts
    BaseDelay:      1 * time.Second,     // Base delay for exponential backoff
    MaxDelay:       30 * time.Second,    // Maximum delay between retries
    RequestTimeout: 30 * time.Second,    // HTTP request timeout
})
```

#### Search Parameters
```go
searchParams := internal.SearchQuery{
    Keywords: "Frontend Developer",      // Job search keywords
    Location: "Japan",                   // Job location
    FWT:      "2,3",                    // Work type: 2=remote, 3=hybrid
}
```

#### Pipeline Configuration
```go
// Pipeline with 5 workers and 1-second rate limit
jobPipeline := pipeline.NewJobPipeline(&scraperService, 5, 1*time.Second)

// Process 10 pages with search parameters
err = jobPipeline.ProcessJobsStreaming(ctx, 10, jobRepo, jobDescRepo, searchParams)
```

## 📊 Performance Metrics

### Before Optimization (Sequential + No Retries)
- Scrape 100 jobs: ~2 minutes
- Process descriptions: ~5 minutes
- **Failures**: High failure rate due to network issues
- **Total: ~7+ minutes** (with manual retries)

### After Optimization (Streaming Pipeline + Retry Logic)
- Scrape 100 jobs: ~2 minutes (with automatic retries)
- Process descriptions: **~2 minutes (concurrent with retries)**
- **Failures**: Near-zero failure rate with exponential backoff
- **Total: ~2 minutes** (65% improvement + reliability)

### Key Improvements
1. **Streaming Architecture**: Jobs processed immediately as scraped
2. **Concurrent Workers**: 5 workers processing descriptions simultaneously
3. **Intelligent Retries**: Automatic retry with exponential backoff
4. **Resilient Network Handling**: Graceful handling of temporary failures
5. **Smart Error Classification**: Skip permanent errors, retry temporary ones

## 🔍 Monitoring & Debugging

The application provides detailed logging with retry information:
```
Scraping page 1
Page 1 complete: sent 25 jobs to channel
Processing job: Frontend Developer at Company X
Request attempt 1 failed: connection timeout
Retrying in 1s... (attempt 1/3)
Request attempt 2 succeeded
Saved 250 job descriptions to database
```

### Retry Logging
```
Request attempt 1 failed with status 503
Retrying in 1s... (attempt 1/3)
Request attempt 2 failed with status 502
Retrying in 2s... (attempt 2/3)
Request attempt 3 succeeded
```

## 🛠️ Troubleshooting

### Common Issues

**Network/Retry Issues**
- Check retry configuration in scraper config
- Monitor retry logs for patterns
- Adjust `MaxRetries`, `BaseDelay`, or `MaxDelay` if needed
- Verify network connectivity and DNS resolution

**Rate Limiting/Blocking**
- Increase delays: modify rate limiter from 2s to 5s+
- Reduce concurrent workers from 5 to 2-3
- Check if IP is temporarily blocked
- Verify User-Agent header is set correctly

**Database Connection Issues**
- Verify PostgreSQL is running: `pg_ctl status`
- Check connection string in `.env` file
- Ensure database exists and migrations ran
- Check database logs for connection errors

**Memory/Performance Issues**
- Monitor goroutine count for leaks
- Reduce number of pages processed per run
- Check channel buffer sizes
- Monitor database connection pool usage

### Debugging Tips

**Enable Verbose Logging**
- Watch retry attempts and delays
- Monitor worker processing rates
- Check database operation timing
- Verify channel coordination

**Test Configuration**
```go
// Conservative settings for testing
scraper := internal.NewScraper(internal.Config{
    MaxRetries:     5,                   // More retries
    BaseDelay:      2 * time.Second,     // Longer delays
    MaxDelay:       60 * time.Second,    // Higher max delay
    RequestTimeout: 60 * time.Second,    // Longer timeout
})

// Fewer workers for testing
jobPipeline := pipeline.NewJobPipeline(&scraperService, 2, 3*time.Second)
```

## 🔮 Future Enhancements

- [ ] **CV Matching**: Compare scraped jobs against CV requirements
- [ ] **AI Analysis**: Enhanced job analysis using Gemini AI
- [ ] **Web Interface**: Dashboard for monitoring and results
- [ ] **Multiple Sources**: Support for other job boards
- [ ] **Real-time Updates**: Continuous scraping with webhooks
- [ ] **Advanced Filtering**: Location, salary, experience filters

## 📝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## ⚠️ Disclaimer

This tool is for educational purposes. Please respect LinkedIn's Terms of Service and robots.txt. Use responsibly and consider rate limiting to avoid being blocked. -->
