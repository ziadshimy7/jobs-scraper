package repo

import (
	"database/sql"
	"fmt"

	"github.com/jobs-scraper/internal/models"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) SaveJobs(jobs []models.Job) error {
	if len(jobs) == 0 {
		return nil
	}

	// Deduplicate jobs by ID to avoid duplicate errors
	jobMap := make(map[int64]models.Job)
	for _, job := range jobs {
		jobMap[job.ID] = job
	}

	uniqueJobs := make([]models.Job, 0, len(jobMap))
	for _, job := range jobMap {
		uniqueJobs = append(uniqueJobs, job)
	}

	sqlStatement := `
        INSERT INTO jobs (id, title, company, company_link, location, job_link)
        VALUES 
    `

	// Create the value placeholders for all jobs
	vals := []interface{}{}
	for i, job := range uniqueJobs {

		n := i * 6

		if i > 0 {
			sqlStatement += ","
		}
		sqlStatement += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4, n+5, n+6)

		vals = append(vals, job.ID, job.Title, job.Company, job.CompanyLink, job.Location, job.JobLink)
	}

	sqlStatement += `
        ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title,
        company = EXCLUDED.company,
        company_link = EXCLUDED.company_link,
        location = EXCLUDED.location,
        job_link = EXCLUDED.job_link
    `

	_, err := r.db.Exec(sqlStatement, vals...)
	if err != nil {
		return fmt.Errorf("error inserting jobs: %v", err)
	}

	return nil
}

func (r *JobRepository) GetAllJobs() ([]models.Job, error) {
	rows, err := r.db.Query("SELECT id, title, company, company_link, location, job_link FROM jobs")
	if err != nil {
		return nil, fmt.Errorf("error querying jobs: %v", err)
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var job models.Job
		if err := rows.Scan(&job.ID, &job.Title, &job.Company, &job.CompanyLink, &job.Location, &job.JobLink); err != nil {
			return nil, fmt.Errorf("error scanning job row: %v", err)
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over job rows: %v", err)
	}

	return jobs, nil
}
func (r *JobRepository) GetJobByID(id int) (*models.Job, error) {
	var job models.Job

	sqlStatement := `
		SELECT id, title, company, company_link, location, job_link 
		FROM jobs 
		WHERE id = $1
	`

	err := r.db.QueryRow(sqlStatement, id).Scan(
		&job.ID,
		&job.Title,
		&job.Company,
		&job.CompanyLink,
		&job.Location,
		&job.JobLink,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job with ID %d not found", id)
	}

	if err != nil {
		return nil, fmt.Errorf("error querying job: %v", err)
	}

	return &job, nil
}
