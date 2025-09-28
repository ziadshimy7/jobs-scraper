CREATE TABLE IF NOT EXISTS job_descriptions (
    id SERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL,
    description TEXT NOT NULL,
    job_criteria JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

-- Create an index on job_id for faster lookups
CREATE INDEX idx_job_descriptions_job_id ON job_descriptions(job_id);

-- Create a GIN index on the job_criteria JSONB field for faster searching within the JSON data
CREATE INDEX idx_job_descriptions_criteria ON job_descriptions USING GIN (job_criteria);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_job_descriptions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trigger_update_job_descriptions_timestamp
    BEFORE UPDATE ON job_descriptions
    FOR EACH ROW
    EXECUTE FUNCTION update_job_descriptions_updated_at();