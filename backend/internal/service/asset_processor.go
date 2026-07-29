package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
)

// AssetProcessor handles asynchronous processing of uploaded assets.
type AssetProcessor struct {
	repository *repository.Repository
	queue      chan processJob
	shutdown   chan struct{}
	wg         sync.WaitGroup
	started    bool
}

type processJob struct {
	AssetID string
	Kind    domain.AssetKind
}

func NewAssetProcessor(
	repositoryInstance *repository.Repository,
	workerCount int,
) *AssetProcessor {
	if workerCount <= 0 {
		workerCount = 2
	}

	p := &AssetProcessor{
		repository: repositoryInstance,
		queue:      make(chan processJob, 256),
		shutdown:   make(chan struct{}),
	}

	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}

	p.started = true

	return p
}

func (p *AssetProcessor) Enqueue(ctx context.Context, assetID string, kind domain.AssetKind) error {
	if !p.started {
		return nil
	}

	select {
	case p.queue <- processJob{AssetID: assetID, Kind: kind}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return p.processAsset(ctx, assetID, kind)
	}
}

func (p *AssetProcessor) Shutdown(ctx context.Context) error {
	if !p.started {
		return nil
	}

	close(p.shutdown)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *AssetProcessor) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			_ = p.processAsset(context.Background(), job.AssetID, job.Kind)

		case <-p.shutdown:
			return
		}
	}
}

func (p *AssetProcessor) processAsset(
	ctx context.Context,
	assetID string,
	kind domain.AssetKind,
) error {
	now := time.Now().UTC()

	if err := p.repository.UpdateAssetProcessStatus(
		ctx,
		assetID,
		domain.AssetProcessStatusProcessing,
		"",
		&now,
		"",
		"",
	); err != nil {
		return fmt.Errorf("mark asset processing: %w", err)
	}

	var maskPath string
	var analysisJSON string
	var processErr error

	switch kind {
	case domain.AssetKindLogo:
		maskPath, processErr = p.processLogo(ctx, assetID)

	case domain.AssetKindPerson:
		maskPath, processErr = p.processPerson(ctx, assetID)

	case domain.AssetKindReference:
		_, analysisJSON, processErr = p.processReference(ctx, assetID)
	}

	status := domain.AssetProcessStatusReady
	processError := ""
	if processErr != nil {
		status = domain.AssetProcessStatusFailed
		processError = processErr.Error()
	}

	processedAt := &now
	if status == domain.AssetProcessStatusFailed {
		processedAt = nil
	}

	return p.repository.UpdateAssetProcessStatus(
		ctx,
		assetID,
		status,
		processError,
		processedAt,
		maskPath,
		analysisJSON,
	)
}

func (p *AssetProcessor) processLogo(
	ctx context.Context,
	assetID string,
) (string, error) {
	asset, err := p.repository.GetAsset(ctx, assetID)
	if err != nil {
		return "", err
	}

	return generateMask(asset)
}

func (p *AssetProcessor) processPerson(
	ctx context.Context,
	assetID string,
) (string, error) {
	asset, err := p.repository.GetAsset(ctx, assetID)
	if err != nil {
		return "", err
	}

	return generateMask(asset)
}

func (p *AssetProcessor) processReference(
	ctx context.Context,
	assetID string,
) ([]string, string, error) {
	asset, err := p.repository.GetAsset(ctx, assetID)
	if err != nil {
		return nil, "", err
	}

	colors, analysis, err := analyzeReference(asset)
	if err != nil {
		return nil, "", err
	}

	analysisJSON := ""
	if analysis != nil {
		raw, _ := json.Marshal(analysis)
		analysisJSON = string(raw)
	}

	return colors, analysisJSON, nil
}

// generateMask generates a background removal mask for person/logo assets.
// Placeholder implementation — integrate with rembg or similar for production.
func generateMask(asset domain.Asset) (string, error) {
	maskDir := filepath.Dir(asset.StoragePath)
	maskPath := filepath.Join(maskDir, "mask_"+asset.ID+".png")

	if err := writePlaceholderMask(asset, maskPath); err != nil {
		return "", fmt.Errorf("generate mask: %w", err)
	}

	return maskPath, nil
}

// analyzeReference analyzes color and composition of a reference image.
func analyzeReference(asset domain.Asset) ([]string, map[string]any, error) {
	file, err := os.Open(asset.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open reference for analysis: %w", err)
	}
	defer file.Close()

	colors := extractDominantColors(file)
	analysis := map[string]any{
		"aspectRatio":    float64(asset.Width) / float64(asset.Height),
		"dominantColors": colors,
		"width":          asset.Width,
		"height":         asset.Height,
	}

	return colors, analysis, nil
}

// writePlaceholderMask writes a copy of the source as a placeholder mask.
// Replace with actual background removal model in production.
func writePlaceholderMask(asset domain.Asset, maskPath string) error {
	source, err := os.Open(asset.StoragePath)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(maskPath), 0o755); err != nil {
		return err
	}

	dest, err := os.Create(maskPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		_ = os.Remove(maskPath)
		return err
	}

	return dest.Sync()
}

// extractDominantColors extracts dominant colors from an image file.
func extractDominantColors(r io.Reader) []string {
	// Placeholder: integrate with go-colorful or similar for production.
	return nil
}
