package domain

import "time"

type AssetKind string

const (
	AssetKindPerson    AssetKind = "person"
	AssetKindLogo      AssetKind = "logo"
	AssetKindReference AssetKind = "reference"
)

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

type Asset struct {
	ID           string    `json:"assetId"`
	Kind         AssetKind `json:"kind"`
	OriginalName string    `json:"originalName"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	StoragePath  string    `json:"-"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AssetResponse struct {
	ID           string    `json:"assetId"`
	Kind         AssetKind `json:"kind"`
	OriginalName string    `json:"originalName"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	SHA256       string    `json:"sha256"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	ContentURL   string    `json:"contentUrl"`
	CreatedAt    time.Time `json:"createdAt"`
}

type AssetListResponse struct {
	Items []AssetResponse `json:"items"`
	Count int             `json:"count"`
}
