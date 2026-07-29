package domain

import "time"

type AssetKind string

const (
	AssetKindPerson    AssetKind = "person"
	AssetKindLogo      AssetKind = "logo"
	AssetKindReference AssetKind = "reference"
)

const AssetProcessVersion = "v1"

func (kind AssetKind) Valid() bool {
	switch kind {
	case AssetKindPerson,
		AssetKindLogo,
		AssetKindReference:
		return true
	default:
		return false
	}
}

type AssetProcessStatus string

const (
	AssetProcessStatusPending    AssetProcessStatus = "pending"
	AssetProcessStatusProcessing AssetProcessStatus = "processing"
	AssetProcessStatusReady      AssetProcessStatus = "ready"
	AssetProcessStatusFailed     AssetProcessStatus = "failed"
)

type ProcessingStepStatus string

const (
	ProcessingStepStatusPending    ProcessingStepStatus = "pending"
	ProcessingStepStatusProcessing ProcessingStepStatus = "processing"
	ProcessingStepStatusCompleted  ProcessingStepStatus = "completed"
	ProcessingStepStatusFailed     ProcessingStepStatus = "failed"
	ProcessingStepStatusSkipped    ProcessingStepStatus = "skipped"
)

type ProcessingStep struct {
	Name        string               `json:"name"`
	Status      ProcessingStepStatus `json:"status"`
	Error       string               `json:"error,omitempty"`
	StartedAt   *time.Time           `json:"startedAt,omitempty"`
	CompletedAt *time.Time           `json:"completedAt,omitempty"`
}

type Asset struct {
	ID           string    `json:"assetId"`
	Kind         AssetKind `json:"kind"`
	OriginalName string    `json:"originalName"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	StoragePath  string    `json:"-"`

	Width  int `json:"width"`
	Height int `json:"height"`

	// Processing state
	ProcessStatus  AssetProcessStatus `json:"processStatus"`
	ProcessError   string             `json:"processError,omitempty"`
	ProcessedAt    *time.Time         `json:"processedAt,omitempty"`
	MaskPath       string             `json:"maskPath,omitempty"`
	AnalysisJSON   string             `json:"analysisJson,omitempty"`
	DominantColors []string           `json:"dominantColors,omitempty"`
	ProcessVersion string             `json:"processVersion,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

type AssetResponse struct {
	ID           string    `json:"assetId"`
	Kind         AssetKind `json:"kind"`
	OriginalName string    `json:"originalName"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	ContentURL   string    `json:"contentUrl"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`

	// Processing state
	ProcessStatus  AssetProcessStatus `json:"processStatus"`
	ProcessError   string             `json:"processError,omitempty"`
	ProcessedAt    *time.Time         `json:"processedAt,omitempty"`
	MaskPath       string             `json:"maskPath,omitempty"`
	AnalysisJSON   string             `json:"analysisJson,omitempty"`
	DominantColors []string           `json:"dominantColors,omitempty"`
	ProcessVersion string             `json:"processVersion,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

type AssetListResponse struct {
	Items []AssetResponse `json:"items"`
	Count int             `json:"count"`
}
