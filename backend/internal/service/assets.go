package service

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
	"github.com/Ripped-sys/StagePoster/backend/internal/storage"
)

var ErrInvalidAssetKind = errors.New("invalid asset kind")

type AssetFile struct {
	Body         io.ReadCloser
	Filename     string
	OriginalName string
	ContentType  string
}

type AssetService struct {
	repository *repository.Repository
	store      *storage.AssetStore
}

func NewAssetService(
	repositoryInstance *repository.Repository,
	store *storage.AssetStore,
) *AssetService {
	return &AssetService{
		repository: repositoryInstance,
		store:      store,
	}
}

func (s *AssetService) Upload(
	ctx context.Context,
	kindValue string,
	originalName string,
	body io.Reader,
) (domain.AssetResponse, error) {
	kind := domain.AssetKind(
		strings.ToLower(strings.TrimSpace(kindValue)),
	)

	if !kind.Valid() {
		return domain.AssetResponse{}, ErrInvalidAssetKind
	}

	asset, err := s.store.Save(
		kind,
		originalName,
		body,
	)
	if err != nil {
		return domain.AssetResponse{}, err
	}

	if err := s.repository.CreateAsset(ctx, asset); err != nil {
		_ = s.store.Delete(asset)
		return domain.AssetResponse{}, err
	}

	return assetResponse(asset), nil
}

func (s *AssetService) Get(
	ctx context.Context,
	assetID string,
) (domain.AssetResponse, error) {
	asset, err := s.repository.GetAsset(ctx, assetID)
	if err != nil {
		return domain.AssetResponse{}, err
	}

	return assetResponse(asset), nil
}

func (s *AssetService) List(
	ctx context.Context,
	limit int,
) (domain.AssetListResponse, error) {
	assets, err := s.repository.ListAssets(ctx, limit)
	if err != nil {
		return domain.AssetListResponse{}, err
	}

	items := make(
		[]domain.AssetResponse,
		0,
		len(assets),
	)

	for _, asset := range assets {
		items = append(items, assetResponse(asset))
	}

	return domain.AssetListResponse{
		Items: items,
		Count: len(items),
	}, nil
}

func (s *AssetService) OpenContent(
	ctx context.Context,
	assetID string,
) (AssetFile, error) {
	asset, err := s.repository.GetAsset(ctx, assetID)
	if err != nil {
		return AssetFile{}, err
	}

	file, err := s.store.Open(asset)
	if err != nil {
		return AssetFile{}, err
	}

	return AssetFile{
		Body:         file,
		Filename:     asset.Filename,
		OriginalName: asset.OriginalName,
		ContentType:  asset.MimeType,
	}, nil
}

func assetResponse(
	asset domain.Asset,
) domain.AssetResponse {
	return domain.AssetResponse{
		ID:           asset.ID,
		Kind:         asset.Kind,
		OriginalName: asset.OriginalName,
		Filename:     asset.Filename,
		MimeType:     asset.MimeType,
		SizeBytes:    asset.SizeBytes,
		SHA256:       asset.SHA256,
		Width:        asset.Width,
		Height:       asset.Height,
		ContentURL:   "/api/assets/" + asset.ID + "/content",
		CreatedAt:    asset.CreatedAt,
	}
}
