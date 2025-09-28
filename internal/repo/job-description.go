package repo

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jobs-scraper/internal/models"
)

type JobDescriptionRepository struct {
	db *sql.DB
}

type JobDescriptionData struct {
	JobID       int
	Description string
	Criteria    map[string]string
}

func NewJobDescriptionRepository(db *sql.DB) *JobDescriptionRepository {
	return &JobDescriptionRepository{db: db}
}

func (r *JobDescriptionRepository) SaveJobDescriptions(jobDescriptions []models.JobDescription) error {
	if len(jobDescriptions) == 0 {
		return nil
	}

	// Build the VALUES clause dynamically
	valueStrings := make([]string, 0, len(jobDescriptions))
	valueArgs := make([]interface{}, 0, len(jobDescriptions)*3)

	for i, jd := range jobDescriptions {
		// Convert criteria map to JSONB
		criteriaByte, err := json.Marshal(jd.Criteria)
		if err != nil {
			return fmt.Errorf("error marshaling job criteria for job %d: %v", jd.JobID, err)
		}

		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
		valueArgs = append(valueArgs, jd.JobID, jd.Description, criteriaByte)
	}

	sqlStatement := fmt.Sprintf(`
		INSERT INTO job_descriptions (job_id, description, job_criteria)
		VALUES %s
		ON CONFLICT (job_id) DO UPDATE SET
		description = EXCLUDED.description,
		job_criteria = EXCLUDED.job_criteria,
		updated_at = CURRENT_TIMESTAMP
	`, strings.Join(valueStrings, ","))

	_, err := r.db.Exec(sqlStatement, valueArgs...)
	if err != nil {
		return fmt.Errorf("error saving job descriptions: %v", err)
	}

	return nil
}

func (r *JobDescriptionRepository) GetJobDescriptionByJobID(jobID int64) (string, map[string]string, error) {
	var (
		description  string
		criteriaByte []byte
		criteria     map[string]string
	)

	sqlStatement := `SELECT description, job_criteria FROM job_descriptions WHERE job_id = $1`
	err := r.db.QueryRow(sqlStatement, jobID).Scan(&description, &criteriaByte)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, nil // No description found
		}
		return "", nil, fmt.Errorf("error fetching job description: %v", err)
	}

	// Unmarshal the JSON criteria
	if criteriaByte != nil {
		if err := json.Unmarshal(criteriaByte, &criteria); err != nil {
			return "", nil, fmt.Errorf("error unmarshaling job criteria: %v", err)
		}
	}

	return description, criteria, nil
}
