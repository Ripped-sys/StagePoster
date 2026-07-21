package domain

import "time"

type PosterStatus string

const (
	PosterStatusPlanning          PosterStatus = "planning_candidates"
	PosterStatusGenerating        PosterStatus = "generating_candidates"
	PosterStatusValidating        PosterStatus = "validating_candidates"
	PosterStatusAwaitingSelection PosterStatus = "awaiting_selection"
	PosterStatusSelected          PosterStatus = "selected"
	PosterStatusComposing         PosterStatus = "composing"
	PosterStatusSucceeded         PosterStatus = "succeeded"
	PosterStatusFailed            PosterStatus = "failed"
)

type CandidateStatus string

const (
	CandidateStatusGenerating CandidateStatus = "generating"
	CandidateStatusValidating CandidateStatus = "validating"
	CandidateStatusRetrying   CandidateStatus = "retrying"
	CandidateStatusReady      CandidateStatus = "ready"
	CandidateStatusFailed     CandidateStatus = "failed"
)

type EventBrief struct {
	Title        string `json:"title"`
	Artist       string `json:"artist,omitempty"`
	Date         string `json:"date"`
	Time         string `json:"time"`
	Venue        string `json:"venue"`
	PresalePrice string `json:"presalePrice,omitempty"`
	DoorPrice    string `json:"doorPrice,omitempty"`
}

type BrandingBrief struct {
	ArtistLogoAssetID   string   `json:"artistLogoAssetId,omitempty"`
	EventLogoAssetID    string   `json:"eventLogoAssetId,omitempty"`
	SponsorLogoAssetIDs []string `json:"sponsorLogoAssetIds,omitempty"`
}

type VisualBrief struct {
	Style           string   `json:"style"`
	Theme           string   `json:"theme"`
	MusicGenre      string   `json:"musicGenre,omitempty"`
	Mood            []string `json:"mood,omitempty"`
	PreferredColors []string `json:"preferredColors,omitempty"`
}

type CreatePosterRequest struct {
	Event    EventBrief    `json:"event"`
	Branding BrandingBrief `json:"branding"`
	Visual   VisualBrief   `json:"visual"`
}

type GoalContract struct {
	Width               int  `json:"width"`
	Height              int  `json:"height"`
	AllowPeople         bool `json:"allowPeople"`
	AllowReadableText   bool `json:"allowReadableText"`
	RequireCentralMotif bool `json:"requireCentralMotif"`
	MaxAttempts         int  `json:"maxAttempts"`
}

type CandidateSpec struct {
	VariantKey  string   `json:"variantKey"`
	VariantName string   `json:"variantName"`
	Motif       string   `json:"motif"`
	Composition string   `json:"composition"`
	Materials   []string `json:"materials"`
	Palette     []string `json:"palette"`
	Lighting    string   `json:"lighting"`
}

type PosterRecord struct {
	ID                  string       `json:"posterId"`
	Status              PosterStatus `json:"status"`
	StyleKey            string       `json:"styleKey"`
	EventJSON           string       `json:"-"`
	BrandingJSON        string       `json:"-"`
	VisualJSON          string       `json:"-"`
	GoalJSON            string       `json:"-"`
	SelectedCandidateID string       `json:"selectedCandidateId,omitempty"`
	ErrorMessage        string       `json:"error,omitempty"`
	CreatedAt           time.Time    `json:"createdAt"`
	UpdatedAt           time.Time    `json:"updatedAt"`
	CompletedAt         *time.Time   `json:"completedAt,omitempty"`
}

type CandidateRecord struct {
	ID             string          `json:"candidateId"`
	PosterID       string          `json:"posterId"`
	JobID          string          `json:"jobId"`
	VariantIndex   int             `json:"variantIndex"`
	VariantKey     string          `json:"variantKey"`
	VariantName    string          `json:"variantName"`
	SpecJSON       string          `json:"-"`
	CompiledPrompt string          `json:"-"`
	Seed           int64           `json:"seed"`
	Attempt        int             `json:"attempt"`
	Status         CandidateStatus `json:"status"`
	Passed         bool            `json:"passed"`
	Selected       bool            `json:"selected"`
	ErrorMessage   string          `json:"error,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type CandidateResponse struct {
	CandidateID string          `json:"candidateId"`
	VariantKey  string          `json:"variantKey"`
	VariantName string          `json:"variantName"`
	Status      CandidateStatus `json:"status"`
	Attempt     int             `json:"attempt"`
	Selected    bool            `json:"selected"`
	ImageURL    string          `json:"imageUrl,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type PosterResponse struct {
	PosterID            string              `json:"posterId"`
	Status              PosterStatus        `json:"status"`
	SelectedCandidateID string              `json:"selectedCandidateId,omitempty"`
	ResultURL           string              `json:"resultUrl,omitempty"`
	ThumbnailURL        string              `json:"thumbnailUrl,omitempty"`
	Candidates          []CandidateResponse `json:"candidates"`
	Progress            PosterProgress      `json:"progress"`
	Error               string              `json:"error,omitempty"`
	CreatedAt           time.Time           `json:"createdAt"`
	UpdatedAt           time.Time           `json:"updatedAt"`
}

type PosterProgress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

type SelectCandidateRequest struct {
	CandidateID string `json:"candidateId"`
}
