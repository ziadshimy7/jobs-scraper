-- Remove unique constraint from job_id column in job_descriptions table
ALTER TABLE job_descriptions DROP CONSTRAINT IF EXISTS unique_job_id;
