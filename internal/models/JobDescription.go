package models

type JobDescription struct {
	JobID       int64
	Description string
	Criteria    map[string]string
}
