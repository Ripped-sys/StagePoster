package domain

import "time"

type GenerateRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negativePrompt,omitempty"`
	Seed           *int64 `json:"seed,omitempty"`
}

type GenerateResponse struct {
	JobID    string    `json:"jobId"`
	PromptID string    `json:"promptId"`
	Status   JobStatus `json:"status"`
	Seed     int64     `json:"seed"`
}

type ImageMeta struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type JobResponse struct {
	JobID           string     `json:"jobId"`
	PromptID        string     `json:"promptId,omitempty"`
	Status          JobStatus  `json:"status"`
	Prompt          string     `json:"prompt"`
	NegativePrompt  string     `json:"negativePrompt,omitempty"`
	Seed            int64      `json:"seed"`
	WorkflowKey     string     `json:"workflowKey"`
	WorkflowVersion string     `json:"workflowVersion"`
	Image           *ImageMeta `json:"image,omitempty"`
	ResultURL       string     `json:"resultUrl,omitempty"`
	Error           string     `json:"error,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	StartedAt       *time.Time `json:"startedAt,omitempty"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type JobListResponse struct {
	Items []JobResponse `json:"items"`
	Count int           `json:"count"`
}
