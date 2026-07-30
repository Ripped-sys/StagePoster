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
	AISessionStatusCompletedWithWarnings      AISessionStatus = "completed_with_warnings"
	AISessionStatusSucceeded                  AISessionStatus = "succeeded"
	AISessionStatusFailed                     AISessionStatus = "failed"
	AISessionStatusCanceled                   AISessionStatus = "canceled"
)

// AISessionStatusLegacyCanceled 是 2026-07 之前写进库的英式拼写。
// 线上值已统一成单 l 的 "canceled"（与 PosterStatusCanceled / JobStatusCanceled
// 一致），但 NFS 上的旧会话行还带着双 l。读取时归一化，否则一个已取消的会话
// 会因为常量对不上而被 Terminal() 判成仍在进行。
const AISessionStatusLegacyCanceled AISessionStatus = "cancelled"

// NormalizeAISessionStatus 把旧拼写折叠到当前线上值。
func NormalizeAISessionStatus(
	status AISessionStatus,
) AISessionStatus {
	if status == AISessionStatusLegacyCanceled {
		return AISessionStatusCanceled
	}

	return status
}

func (status AISessionStatus) Terminal() bool {
	switch NormalizeAISessionStatus(status) {
	case AISessionStatusSucceeded,
		AISessionStatusCompletedWithWarnings,
		AISessionStatusFailed,
		AISessionStatusCanceled:
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

	StoragePath string `json:"-"`

	// Usage tracking
	UsedInStage  []string `json:"usedInStage,omitempty"` // plan / candidate / logo_overlay / review
	ActuallyUsed bool     `json:"actuallyUsed"`         // 实际参与生成
	UsageNote    string   `json:"usageNote,omitempty"`   // 简要说明

	CreatedAt time.Time `json:"createdAt"`
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

type AIReviewSummary struct {
	Finalized      bool           `json:"finalized"`
	Accepted       bool           `json:"accepted"`
	Rounds         int            `json:"rounds"`
	BestRound      int            `json:"bestRound,omitempty"`
	BestScore      Score          `json:"bestScore,omitempty"`
	LatestDecision ReviewDecision `json:"latestDecision,omitempty"`
	Warning        string         `json:"warning,omitempty"`
}

type AISessionResponse struct {
	SessionID        string           `json:"sessionId"`
	Status           AISessionStatus  `json:"status"`
	AvailableActions []string         `json:"availableActions,omitempty"`
	Brief            AISessionBrief   `json:"brief"`
	MissingFields    []string         `json:"missingFields"`
	SelectedPlanID   string           `json:"selectedPlanId,omitempty"`
	PosterID         string           `json:"posterId,omitempty"`
	Error            string           `json:"error,omitempty"`
	ReviewSummary    *AIReviewSummary `json:"reviewSummary,omitempty"`

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
