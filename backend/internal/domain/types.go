package domain

type GenerateRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negativePrompt,omitempty"`
	Seed           *int64 `json:"seed,omitempty"`
}

type GenerateResponse struct {
	JobID    string `json:"jobId"`
	PromptID string `json:"promptId"`
	Status   string `json:"status"`
	Seed     int64  `json:"seed"`
}

type ImageMeta struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type JobResponse struct {
	JobID     string     `json:"jobId"`
	Status    string     `json:"status"`
	Image     *ImageMeta `json:"image,omitempty"`
	ResultURL string     `json:"resultUrl,omitempty"`
	Messages  []any      `json:"messages,omitempty"`
}
