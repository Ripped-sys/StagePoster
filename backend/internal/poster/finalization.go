package poster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

var ErrReviewSnapshotNotFound = errors.New(
	"review snapshot is not available",
)

func (s *Service) RecomposeFromReview(
	ctx context.Context,
	posterID string,
	result domain.ReviewResult,
) error {
	s.reconcileMu.Lock()
	defer s.reconcileMu.Unlock()

	posterRecord, err := s.repository.GetPoster(
		ctx,
		posterID,
	)
	if err != nil {
		return err
	}

	if posterRecord.Status !=
		domain.PosterStatusSucceeded {
		return fmt.Errorf(
			"cannot recompose poster while status is %s",
			posterRecord.Status,
		)
	}

	if err := s.repository.UpdatePosterStatus(
		ctx,
		posterID,
		domain.PosterStatusComposing,
		"",
	); err != nil {
		return err
	}

	posterRecord.Status =
		domain.PosterStatusComposing

	if err := s.composePoster(
		ctx,
		posterRecord,
		reviewAdjustments(result),
		"",
	); err != nil {
		return err
	}

	return s.requireSuccessfulComposition(
		ctx,
		posterID,
	)
}

func (s *Service) RegenerateFromReview(
	ctx context.Context,
	posterID string,
	result domain.ReviewResult,
) error {
	posterRecord, err := s.repository.GetPoster(
		ctx,
		posterID,
	)
	if err != nil {
		return err
	}

	if posterRecord.Status !=
		domain.PosterStatusSucceeded {
		return fmt.Errorf(
			"cannot regenerate poster while status is %s",
			posterRecord.Status,
		)
	}

	candidate, err :=
		s.repository.GetSelectedCandidate(
			ctx,
			posterID,
		)
	if err != nil {
		return err
	}

	var goal domain.GoalContract

	if err := json.Unmarshal(
		[]byte(posterRecord.GoalJSON),
		&goal,
	); err != nil {
		return fmt.Errorf(
			"decode poster goal contract: %w",
			err,
		)
	}

	prompt := joinPromptFragments(
		candidate.CompiledPrompt,
		result.NextInstruction.PromptAdditions,
	)

	negativePrompt := joinPromptFragments(
		"text, letters, words, typography, caption, logo, watermark, signage, gibberish",
		result.NextInstruction.NegativePromptAdditions,
	)

	nextAttempt := candidate.Attempt + 1
	nextSeed := candidate.Seed +
		int64(nextAttempt*1000003)

	generation, err := s.core.Generate(
		ctx,
		domain.GenerateRequest{
			Prompt:         prompt,
			NegativePrompt: negativePrompt,
			Seed:           &nextSeed,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"submit regenerated key visual: %w",
			err,
		)
	}

	output, err := s.waitForGeneratedOutput(
		ctx,
		generation.JobID,
	)
	if err != nil {
		return err
	}

	if err := s.evaluator.Evaluate(
		output,
		goal,
	); err != nil {
		return fmt.Errorf(
			"evaluate regenerated key visual: %w",
			err,
		)
	}

	s.reconcileMu.Lock()
	defer s.reconcileMu.Unlock()

	if err := s.repository.AdoptCandidateJob(
		ctx,
		candidate.ID,
		generation.JobID,
		nextSeed,
		nextAttempt,
	); err != nil {
		return fmt.Errorf(
			"adopt regenerated candidate job: %w",
			err,
		)
	}

	if err := s.repository.UpdatePosterStatus(
		ctx,
		posterID,
		domain.PosterStatusComposing,
		"",
	); err != nil {
		return err
	}

	posterRecord.Status =
		domain.PosterStatusComposing

	if err := s.composePoster(
		ctx,
		posterRecord,
		reviewAdjustments(result),
		output.StoragePath,
	); err != nil {
		return err
	}

	return s.requireSuccessfulComposition(
		ctx,
		posterID,
	)
}

func (s *Service) waitForGeneratedOutput(
	ctx context.Context,
	jobID string,
) (domain.Output, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		status, err := s.core.Status(
			ctx,
			jobID,
		)
		if err != nil {
			return domain.Output{},
				fmt.Errorf(
					"read regenerated job status: %w",
					err,
				)
		}

		switch status.Status {
		case domain.JobStatusSucceeded:
			output, outputErr :=
				s.repository.GetOutput(
					ctx,
					jobID,
					"poster",
				)
			if outputErr != nil {
				return domain.Output{},
					fmt.Errorf(
						"read regenerated output: %w",
						outputErr,
					)
			}

			return output, nil

		case domain.JobStatusFailed,
			domain.JobStatusCanceled:

			return domain.Output{},
				fmt.Errorf(
					"regenerated key visual failed: %s",
					status.Error,
				)
		}

		select {
		case <-ctx.Done():
			return domain.Output{}, ctx.Err()

		case <-ticker.C:
		}
	}
}

func (s *Service) requireSuccessfulComposition(
	ctx context.Context,
	posterID string,
) error {
	posterRecord, err := s.repository.GetPoster(
		ctx,
		posterID,
	)
	if err != nil {
		return err
	}

	if posterRecord.Status !=
		domain.PosterStatusSucceeded {
		return fmt.Errorf(
			"poster composition did not succeed: %s",
			posterRecord.ErrorMessage,
		)
	}

	return nil
}

func (s *Service) SnapshotReviewRound(
	ctx context.Context,
	posterID string,
	round int,
) error {
	if round <= 0 {
		return fmt.Errorf(
			"invalid review round %d",
			round,
		)
	}

	for _, kind := range []string{
		domain.PosterOutputKindFinal,
		domain.PosterOutputKindThumbnail,
	} {
		output, err := s.repository.GetPosterOutput(
			ctx,
			posterID,
			kind,
		)
		if err != nil {
			return err
		}

		target := reviewSnapshotPath(
			output.StoragePath,
			round,
			kind,
		)

		if _, err := os.Stat(target); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}

		if err := copyFileAtomic(
			output.StoragePath,
			target,
		); err != nil {
			return fmt.Errorf(
				"snapshot %s for review round %d: %w",
				kind,
				round,
				err,
			)
		}
	}

	return nil
}

func (s *Service) RestoreReviewRound(
	ctx context.Context,
	posterID string,
	round int,
) error {
	for _, kind := range []string{
		domain.PosterOutputKindFinal,
		domain.PosterOutputKindThumbnail,
	} {
		output, err := s.repository.GetPosterOutput(
			ctx,
			posterID,
			kind,
		)
		if err != nil {
			return err
		}

		source := reviewSnapshotPath(
			output.StoragePath,
			round,
			kind,
		)

		if _, err := os.Stat(source); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf(
					"%w: round %d %s",
					ErrReviewSnapshotNotFound,
					round,
					kind,
				)
			}

			return err
		}

		if err := copyFileAtomic(
			source,
			output.StoragePath,
		); err != nil {
			return fmt.Errorf(
				"restore %s from review round %d: %w",
				kind,
				round,
				err,
			)
		}
	}

	return s.repository.CompletePoster(
		ctx,
		posterID,
	)
}

func (s *Service) KeepExistingResult(
	ctx context.Context,
	posterID string,
) error {
	for _, kind := range []string{
		domain.PosterOutputKindFinal,
		domain.PosterOutputKindThumbnail,
	} {
		output, err := s.repository.GetPosterOutput(
			ctx,
			posterID,
			kind,
		)
		if err != nil {
			return err
		}

		if _, err := os.Stat(
			output.StoragePath,
		); err != nil {
			return err
		}
	}

	return s.repository.CompletePoster(
		ctx,
		posterID,
	)
}

func reviewAdjustments(
	result domain.ReviewResult,
) domain.CompositionAdjustments {
	adjustments := domain.CompositionAdjustments{
		Template: strings.TrimSpace(
			result.NextInstruction.ComposerTemplate,
		),
	}

	for _, issue := range result.Issues {
		code := strings.ToUpper(
			strings.TrimSpace(issue.Code),
		)

		switch {
		case strings.Contains(
			code,
			"TITLE",
		):
			adjustments.TitleOffsetRatio =
				maxFloat(
					adjustments.TitleOffsetRatio,
					0.055,
				)

		case strings.Contains(
			code,
			"INFORMATION_PANEL_CONTRAST",
		):
			adjustments.PanelTheme = "dark"
			adjustments.PanelTopRatio =
				maxFloat(
					adjustments.PanelTopRatio,
					0.80,
				)

		case strings.Contains(
			code,
			"INFORMATION_PANEL",
		):
			adjustments.PanelTopRatio =
				maxFloat(
					adjustments.PanelTopRatio,
					0.81,
				)

		case strings.Contains(code, "SPACING"),
			strings.Contains(code, "HIERARCHY"),
			strings.Contains(code, "LAYOUT"):

			adjustments.TitleOffsetRatio =
				maxFloat(
					adjustments.TitleOffsetRatio,
					0.04,
				)

			adjustments.PanelTopRatio =
				maxFloat(
					adjustments.PanelTopRatio,
					0.80,
				)
		}
	}

	return adjustments
}

func joinPromptFragments(
	base string,
	additions []string,
) string {
	parts := make([]string, 0, len(additions)+1)

	if value := strings.TrimSpace(base); value != "" {
		parts = append(parts, value)
	}

	for _, addition := range additions {
		if value := strings.TrimSpace(addition); value != "" {
			parts = append(parts, value)
		}
	}

	return strings.Join(parts, ", ")
}

func reviewSnapshotPath(
	outputPath string,
	round int,
	kind string,
) string {
	extension := filepath.Ext(outputPath)
	if extension == "" {
		extension = ".png"
	}

	return filepath.Join(
		filepath.Dir(outputPath),
		fmt.Sprintf(
			"review-round-%d-%s%s",
			round,
			kind,
			extension,
		),
	)
}

func copyFileAtomic(
	source string,
	target string,
) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()

	if err := os.MkdirAll(
		filepath.Dir(target),
		0o755,
	); err != nil {
		return err
	}

	temporary, err := os.CreateTemp(
		filepath.Dir(target),
		".stageposter-copy-*",
	)
	if err != nil {
		return err
	}

	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := io.Copy(
		temporary,
		input,
	); err != nil {
		temporary.Close()
		return err
	}

	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}

	if err := temporary.Close(); err != nil {
		return err
	}

	return os.Rename(
		temporaryPath,
		target,
	)
}

func maxFloat(
	left float64,
	right float64,
) float64 {
	if left > right {
		return left
	}

	return right
}
