package domain

import "time"

const (
	PosterOutputKindFinal     = "final_poster"
	PosterOutputKindThumbnail = "thumbnail"
)

type CompositionAsset struct {
	ID          string
	MimeType    string
	StoragePath string
	Width       int
	Height      int
}

type CompositionAdjustments struct {
	Template         string
	TitleOffsetRatio float64
	PanelTopRatio    float64
	PanelTheme       string
}

type ComposeInput struct {
	PosterID    string
	CandidateID string

	Width  int
	Height int

	KeyVisualPath string

	Event       EventBrief
	Adjustments CompositionAdjustments

	ArtistLogo CompositionAsset
	EventLogo  CompositionAsset
	Sponsors   []CompositionAsset
}

type ComposeResult struct {
	FinalPath     string
	ThumbnailPath string

	Width           int
	Height          int
	ThumbnailWidth  int
	ThumbnailHeight int
}

type PosterOutput struct {
	ID          string
	PosterID    string
	CandidateID string

	Kind        string
	Filename    string
	MimeType    string
	StoragePath string

	Width  int
	Height int

	CreatedAt time.Time
}
