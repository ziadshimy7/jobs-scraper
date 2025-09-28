-- Add unique constraint to job_id column in job_descriptions table
ALTER TABLE job_descriptions ADD CONSTRAINT unique_job_id UNIQUE (job_id);
