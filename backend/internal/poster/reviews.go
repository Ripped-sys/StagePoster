package poster

import (
	"context"
	"errors"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
	"github.com/Ripped-sys/StagePoster/backend/internal/repository"
)

type ReviewMaterial struct {
	Poster    domain.PosterRecord  `json:"poster"`
	Output    domain.PosterOutput  `json:"output"`
	Candidate *domain.CandidateRecord `json:"candidate,omitempty"`
}

func (s *Service) ReviewMaterial(
	ctx context.Context,
	posterID string,
) (ReviewMaterial, error) {
	posterRecord, err := s.repository.GetPoster(
		ctx,
		posterID,
	)
	if err != nil {
		return ReviewMaterial{}, err
	}

	output, err := s.repository.GetPosterOutput(
		ctx,
		posterID,
		domain.PosterOutputKindFinal,
	)
	if errors.Is(err, repository.ErrNotFound) {
		return ReviewMaterial{}, ErrResultNotReady
	}

	if err != nil {
		return ReviewMaterial{}, err
	}

	material := ReviewMaterial{
		Poster: posterRecord,
		Output: output,
	}

	candidate, candidateErr :=
		s.repository.GetSelectedCandidate(
			ctx,
			posterID,
		)

	if candidateErr == nil {
		material.Candidate = &candidate
	} else if !errors.Is(
		candidateErr,
		repository.ErrNotFound,
	) {
		return ReviewMaterial{}, candidateErr
	}

	return material, nil
}

func (s *Service) NextReviewRound(
	ctx context.Context,
	posterID string,
) (int, error) {
	return s.repository.NextPosterReviewRound(
		ctx,
		posterID,
	)
}

func (s *Service) SaveReview(
	ctx context.Context,
	review domain.PosterReviewRecord,
) error {
	return s.repository.CreatePosterReview(
		ctx,
		review,
	)
}

func (s *Service) ListReviews(
	ctx context.Context,
	posterID string,
	limit int,
) ([]domain.PosterReviewRecord, error) {
	// 先确认 poster 存在，避免未知 poster 返回空数组。
	if _, err := s.repository.GetPoster(
		ctx,
		posterID,
	); err != nil {
		return nil, err
	}

	return s.repository.ListPosterReviews(
		ctx,
		posterID,
		limit,
	)
}
