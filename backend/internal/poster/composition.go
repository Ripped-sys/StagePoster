package poster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

var ErrResultNotReady = errors.New(
	"poster result is not ready",
)

type CompositionEngine interface {
	Compose(
		context.Context,
		domain.ComposeInput,
	) (domain.ComposeResult, error)
}

type ComposedFile struct {
	Body        *os.File
	ContentType string
	Filename    string
}

func (s *Service) reconcileComposition(
	ctx context.Context,
	posterRecord domain.PosterRecord,
) error {
	return s.composePoster(
		ctx,
		posterRecord,
		domain.CompositionAdjustments{},
		"",
	)
}

func (s *Service) composePoster(
	ctx context.Context,
	posterRecord domain.PosterRecord,
	adjustments domain.CompositionAdjustments,
	keyVisualOverride string,
) error {
	candidate, err := s.repository.GetSelectedCandidate(
		ctx,
		posterRecord.ID,
	)
	if err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			fmt.Errorf(
				"resolve selected candidate: %w",
				err,
			),
		)
	}

	if candidate.Status != domain.CandidateStatusReady ||
		!candidate.Passed {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			ErrCandidateNotReady,
		)
	}

	if posterRecord.Status == domain.PosterStatusSelected {
		if err := s.repository.UpdatePosterStatus(
			ctx,
			posterRecord.ID,
			domain.PosterStatusComposing,
			"",
		); err != nil {
			return err
		}
	}

	keyVisualPath := strings.TrimSpace(
		keyVisualOverride,
	)

	if keyVisualPath == "" {
		keyVisual, outputErr :=
			s.repository.GetOutput(
				ctx,
				candidate.JobID,
				"poster",
			)
		if outputErr != nil {
			return s.failComposition(
				ctx,
				posterRecord.ID,
				fmt.Errorf(
					"resolve candidate output: %w",
					outputErr,
				),
			)
		}

		keyVisualPath = keyVisual.StoragePath
	}

	var event domain.EventBrief
	if err := json.Unmarshal(
		[]byte(posterRecord.EventJSON),
		&event,
	); err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			fmt.Errorf(
				"decode event brief: %w",
				err,
			),
		)
	}

	var branding domain.BrandingBrief
	if err := json.Unmarshal(
		[]byte(posterRecord.BrandingJSON),
		&branding,
	); err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			fmt.Errorf(
				"decode branding brief: %w",
				err,
			),
		)
	}

	artistLogo, err := s.resolveCompositionAsset(
		ctx,
		branding.ArtistLogoAssetID,
	)
	if err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			fmt.Errorf(
				"resolve artist logo: %w",
				err,
			),
		)
	}

	eventLogo, err := s.resolveCompositionAsset(
		ctx,
		branding.EventLogoAssetID,
	)
	if err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			fmt.Errorf(
				"resolve event logo: %w",
				err,
			),
		)
	}

	sponsors := make(
		[]domain.CompositionAsset,
		0,
		len(branding.SponsorLogoAssetIDs),
	)

	for _, sponsorID := range branding.SponsorLogoAssetIDs {
		sponsor, err := s.resolveCompositionAsset(
			ctx,
			sponsorID,
		)
		if err != nil {
			return s.failComposition(
				ctx,
				posterRecord.ID,
				fmt.Errorf(
					"resolve sponsor logo %s: %w",
					sponsorID,
					err,
				),
			)
		}

		sponsors = append(sponsors, sponsor)
	}

	result, err := s.composer.Compose(
		ctx,
		domain.ComposeInput{
			PosterID:      posterRecord.ID,
			CandidateID:   candidate.ID,
			Width:         1024,
			Height:        1536,
			KeyVisualPath: keyVisualPath,
			Event:         event,
			Adjustments:   adjustments,
			ArtistLogo:    artistLogo,
			EventLogo:     eventLogo,
			Sponsors:      sponsors,
		},
	)
	if err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			err,
		)
	}

	now := time.Now().UTC()

	finalID, err := domain.NewID("poster_output_")
	if err != nil {
		return err
	}

	thumbnailID, err := domain.NewID("poster_output_")
	if err != nil {
		return err
	}

	if err := s.repository.UpsertPosterOutput(
		ctx,
		domain.PosterOutput{
			ID:          finalID,
			PosterID:    posterRecord.ID,
			CandidateID: candidate.ID,
			Kind:        domain.PosterOutputKindFinal,
			Filename:    "final-poster.png",
			MimeType:    "image/png",
			StoragePath: result.FinalPath,
			Width:       result.Width,
			Height:      result.Height,
			CreatedAt:   now,
		},
	); err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			err,
		)
	}

	if err := s.repository.UpsertPosterOutput(
		ctx,
		domain.PosterOutput{
			ID:          thumbnailID,
			PosterID:    posterRecord.ID,
			CandidateID: candidate.ID,
			Kind:        domain.PosterOutputKindThumbnail,
			Filename:    "thumbnail.png",
			MimeType:    "image/png",
			StoragePath: result.ThumbnailPath,
			Width:       result.ThumbnailWidth,
			Height:      result.ThumbnailHeight,
			CreatedAt:   now,
		},
	); err != nil {
		return s.failComposition(
			ctx,
			posterRecord.ID,
			err,
		)
	}

	return s.repository.CompletePoster(
		ctx,
		posterRecord.ID,
	)
}

func (s *Service) resolveCompositionAsset(
	ctx context.Context,
	assetID string,
) (domain.CompositionAsset, error) {
	if assetID == "" {
		return domain.CompositionAsset{}, nil
	}

	return s.repository.GetCompositionAsset(
		ctx,
		assetID,
	)
}

func (s *Service) failComposition(
	ctx context.Context,
	posterID string,
	cause error,
) error {
	if updateErr := s.repository.UpdatePosterStatus(
		ctx,
		posterID,
		domain.PosterStatusFailed,
		cause.Error(),
	); updateErr != nil {
		return errors.Join(cause, updateErr)
	}

	return nil
}

func (s *Service) OpenFinalResult(
	ctx context.Context,
	posterID string,
) (ComposedFile, error) {
	posterRecord, err := s.repository.GetPoster(
		ctx,
		posterID,
	)
	if err != nil {
		return ComposedFile{}, err
	}

	if posterRecord.Status != domain.PosterStatusSucceeded {
		return ComposedFile{}, ErrResultNotReady
	}

	output, err := s.repository.GetPosterOutput(
		ctx,
		posterID,
		domain.PosterOutputKindFinal,
	)
	if err != nil {
		return ComposedFile{}, err
	}

	file, err := os.Open(output.StoragePath)
	if err != nil {
		return ComposedFile{},
			fmt.Errorf(
				"open composed poster: %w",
				err,
			)
	}

	return ComposedFile{
		Body:        file,
		ContentType: output.MimeType,
		Filename:    output.Filename,
	}, nil
}
