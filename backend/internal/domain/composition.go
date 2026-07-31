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

	// UsedAssetIDs 是真正画进画布的素材 ID。ai_session_assets.actually_used
	// 之前恒为 false，对确实用过的素材主动断言"没用过"；证据只能由合成器给出，
	// 因为只有它知道哪个素材因为 StoragePath 为空被跳过了。
	UsedAssetIDs []string
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
