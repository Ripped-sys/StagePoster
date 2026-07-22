package domain

import "time"

type AISessionStatus string

const (
	AISessionStatusCollectingBrief            AISessionStatus = "collecting_brief"
	AISessionStatusAwaitingPlanSelection      AISessionStatus = "awaiting_plan_selection"
	AISessionStatusGeneratingCandidates       AISessionStatus = "generating_candidates"
	AISessionStatusAwaitingCandidateSelection AISessionStatus = "awaiting_candidate_selection"
	AISessionStatusLooping                    AISessionStatus = "looping"
	AISessionStatusNeedsUserInput             AISessionStatus = "needs_user_input"
	AISessionStatusSucceeded                  AISessionStatus = "succeeded"
	AISessionStatusFailed                     AISessionStatus = "failed"
	AISessionStatusCancelled                  AISessionStatus = "cancelled"
)

func (status AISessionStatus) Terminal() bool {
	switch status {
	case AISessionStatusSucceeded,
		AISessionStatusFailed,
		AISessionStatusCancelled:
		return true

	default:
		return false
	}
}

type AISessionBrief struct {
	Event    EventBrief    `json:"event"`
	Branding BrandingBrief `json:"branding"`
	Visual   VisualBrief   `json:"visual"`
}

type AISessionRecord struct {
	ID             string          `json:"sessionId"`
	Status         AISessionStatus `json:"status"`
	Brief          AISessionBrief  `json:"brief"`
	SelectedPlanID string          `json:"selectedPlanId,omitempty"`
	PosterID       string          `json:"posterId,omitempty"`
	ErrorMessage   string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type AIMessageRole string

const (
	AIMessageRoleUser      AIMessageRole = "user"
	AIMessageRoleAssistant AIMessageRole = "assistant"
	AIMessageRoleSystem    AIMessageRole = "system"
)

type AIMessageRecord struct {
	ID        string        `json:"messageId"`
	SessionID string        `json:"sessionId"`
	Role      AIMessageRole `json:"role"`
	Content   string        `json:"content"`
	CreatedAt time.Time     `json:"createdAt"`
}

type AISessionAssetPurpose string

const (
	AISessionAssetPurposePerformer   AISessionAssetPurpose = "performer"
	AISessionAssetPurposeArtistLogo  AISessionAssetPurpose = "artist_logo"
	AISessionAssetPurposeEventLogo   AISessionAssetPurpose = "event_logo"
	AISessionAssetPurposeSponsorLogo AISessionAssetPurpose = "sponsor_logo"
	AISessionAssetPurposeReference   AISessionAssetPurpose = "reference"
)

func (purpose AISessionAssetPurpose) Valid() bool {
	switch purpose {
	case AISessionAssetPurposePerformer,
		AISessionAssetPurposeArtistLogo,
		AISessionAssetPurposeEventLogo,
		AISessionAssetPurposeSponsorLogo,
		AISessionAssetPurposeReference:
		return true

	default:
		return false
	}
}

type AISessionAssetRecord struct {
	SessionID    string                `json:"sessionId"`
	AssetID      string                `json:"assetId"`
	Purpose      AISessionAssetPurpose `json:"purpose"`
	Kind         AssetKind             `json:"kind"`
	OriginalName string                `json:"originalName"`
	MimeType     string                `json:"mimeType"`
	Width        int                   `json:"width"`
	Height       int                   `json:"height"`

	StoragePath string    `json:"-"`
	CreatedAt   time.Time `json:"createdAt"`
}

type AIDesignPlanRecord struct {
	SessionID string     `json:"sessionId"`
	PlanID    string     `json:"planId"`
	Plan      DesignPlan `json:"plan"`
	Selected  bool       `json:"selected"`
	CreatedAt time.Time  `json:"createdAt"`
}

type AIBriefAgentResult struct {
	Reply  string      `json:"reply"`
	Event  EventBrief  `json:"event"`
	Visual VisualBrief `json:"visual"`
}

type AIAssistAsset struct {
	AssetID      string                `json:"assetId"`
	Purpose      AISessionAssetPurpose `json:"purpose"`
	Kind         AssetKind             `json:"kind"`
	OriginalName string                `json:"originalName"`
	MimeType     string                `json:"mimeType"`
	Width        int                   `json:"width"`
	Height       int                   `json:"height"`

	StoragePath string `json:"-"`
}

type CreateAISessionRequest struct {
	Brief  AISessionBrief       `json:"brief"`
	Assets []BindAISessionAsset `json:"assets,omitempty"`
}

type SendAIMessageRequest struct {
	Content string `json:"content"`
}

type BindAISessionAsset struct {
	AssetID string                `json:"assetId"`
	Purpose AISessionAssetPurpose `json:"purpose"`
}

type BindAISessionAssetsRequest struct {
	Assets []BindAISessionAsset `json:"assets"`
}

type AISessionResponse struct {
	SessionID      string          `json:"sessionId"`
	Status         AISessionStatus `json:"status"`
	Brief          AISessionBrief  `json:"brief"`
	MissingFields  []string        `json:"missingFields"`
	SelectedPlanID string          `json:"selectedPlanId,omitempty"`
	PosterID       string          `json:"posterId,omitempty"`
	Error          string          `json:"error,omitempty"`

	Messages []AIMessageRecord      `json:"messages"`
	Assets   []AISessionAssetRecord `json:"assets"`
	Plans    []AIDesignPlanRecord   `json:"plans"`
	Poster   *PosterResponse        `json:"poster,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AIMessageResponse struct {
	Session AISessionResponse `json:"session"`
	Metrics AIMetricsResponse `json:"metrics"`
}
