-- Drop the trigger first
DROP TRIGGER IF EXISTS trigger_update_job_descriptions_timestamp ON job_descriptions;

-- Drop the trigger function
DROP FUNCTION IF EXISTS update_job_descriptions_updated_at();

-- Drop the indexes
DROP INDEX IF EXISTS idx_job_descriptions_criteria;
DROP INDEX IF EXISTS idx_job_descriptions_job_id;

-- Drop the table (this will automatically drop any remaining dependencies)
DROP TABLE IF EXISTS job_descriptions;