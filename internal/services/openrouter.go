package services

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/eduardolat/openroutergo"
	"github.com/jobs-scraper/internal/models"
)

// JobAnalysisResult represents the structured response from job analysis
type JobAnalysisResult struct {
	Recommendation         string   `json:"recommendation"`
	ConfidenceScore        int      `json:"confidence_score"`
	MatchingSkills         []string `json:"matching_skills"`
	MissingSkills          []string `json:"missing_skills"`
	ExperienceMatch        string   `json:"experience_match"`
	Summary                string   `json:"summary"`
	ImprovementSuggestions []string `json:"improvement_suggestions"`
}

// ShouldApply returns true if the recommendation is to apply for the job
func (r *JobAnalysisResult) ShouldApply() bool {
	return r.Recommendation == "apply"
}

// IsHighConfidence returns true if the confidence score is 70 or above
func (r *JobAnalysisResult) IsHighConfidence() bool {
	return r.ConfidenceScore >= 70
}

type OpenRouterService struct {
	model  string
	apiKey string
}

func NewOpenRouterService(model string, apiKey string) OpenRouterService {
	return OpenRouterService{
		model:  model,
		apiKey: apiKey,
	}
}

func (s *OpenRouterService) AnalyzeJobDescription(cv string, jobDesc models.JobDescription) (*JobAnalysisResult, error) {
	client, err := openroutergo.
		NewClient().
		WithAPIKey(s.apiKey).
		Create()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Build the user message with all the provided data
	userMessage := fmt.Sprintf(`Analyze the following CV against the job description and criteria, then provide a recommendation following the schema below.
		1) If there are missing skills, try to guess if they still match based on similar skills or experience in the cv.
		for example: Javascript is mentioned in the cv, but the job requires Vanilla js, since they are the same thing, it should be included in the matching skills.

		2) The job shouldn't require any language skills, preferbly only english.

		3) The job should be remote, or provide relocation to the country.
	CV:
	%s
	
	Job Description:
	%s
	
	Job Criteria (key-value):
	%v
	
	OUTPUT REQUIREMENTS:
	- Return ONLY a single valid JSON object, no markdown, no backticks, no superflous characters so i can parse it.
	- Do NOT include any markdown, code fences, backticks, or any additional text.
	- Use this exact schema and key names:
	{
	  "recommendation": "apply" | "do_not_apply",
	  "confidence_score": number,  // integer 0-100
	  "matching_skills": [string],
	  "missing_skills": [string],
	  "experience_match": "excellent" | "good" | "fair" | "poor",
	  "summary": string,
	  "improvement_suggestions": [string]
	}`, cv, jobDesc.Description, jobDesc.Criteria)

	// Build and execute your request with a fluent API
	_, resp, err := client.
		NewChatCompletion().
		WithModel(s.model).
		WithSystemMessage("You are an expert HR assistant specializing in job application analysis. You help candidates determine if they should apply for specific positions based on their CV and the job requirements. Always respond in valid JSON format.").
		WithUserMessage(userMessage).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to execute completion: %v", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices received from API")
	}

	// Extract JSON from the response (handle markdown code blocks)
	jsonContent := resp.Choices[0].Message.Content

	// Parse the JSON response into our struct
	var result JobAnalysisResult
	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return &result, nil

}

// func (s *OpenRouterService) CreateCV(cv string, jobDesc models.JobDescription) (*JobAnalysisResult, error) {
// }
