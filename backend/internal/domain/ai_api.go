package domain

import "time"

type AIDesignRequest struct {
	Event   EventBrief  `json:"event"`
	Visual  VisualBrief `json:"visual"`
	Message string      `json:"message"`
}

type AIMetricsResponse struct {
	LatencyMS        int64 `json:"latencyMs"`
	PromptTokens     int   `json:"promptTokens"`
	CompletionTokens int   `json:"completionTokens"`
}

type AIDesignResponse struct {
	Result  DesignAgentResult `json:"result"`
	Metrics AIMetricsResponse `json:"metrics"`
}

type PosterReviewRequest struct {
	DesignPlan *DesignPlan `json:"designPlan,omitempty"`
}

type PosterReviewRecord struct {
	ID          string `json:"reviewId"`
	PosterID    string `json:"posterId"`
	OutputID    string `json:"outputId"`
	CandidateID string `json:"candidateId,omitempty"`

	Round      int            `json:"round"`
	TotalScore Score          `json:"totalScore"`
	Decision   ReviewDecision `json:"decision"`
	Result     ReviewResult   `json:"result"`

	Model            string `json:"model"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	LatencyMS        int64  `json:"latencyMs"`

	CreatedAt time.Time `json:"createdAt"`
}

type PosterReviewResponse struct {
	Review PosterReviewRecord `json:"review"`
}

type PosterReviewListResponse struct {
	PosterID string               `json:"posterId"`
	Reviews  []PosterReviewRecord `json:"reviews"`
}
