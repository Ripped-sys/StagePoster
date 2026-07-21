package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/comfy"
	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

type PosterService struct {
	client   *comfy.Client
	template *comfy.Template
}

func NewPosterService(
	client *comfy.Client,
	template *comfy.Template,
) *PosterService {
	return &PosterService{
		client:   client,
		template: template,
	}
}

func (s *PosterService) Health(ctx context.Context) error {
	return s.client.Health(ctx)
}

func (s *PosterService) Bindings() comfy.Bindings {
	return s.template.Bindings()
}

func (s *PosterService) Generate(
	ctx context.Context,
	request domain.GenerateRequest,
) (domain.GenerateResponse, error) {
	request.Prompt = strings.TrimSpace(request.Prompt)

	if request.Prompt == "" {
		return domain.GenerateResponse{},
			errors.New("prompt is required")
	}

	seed := time.Now().UnixNano() & 0x7fffffffffffffff
	if request.Seed != nil {
		seed = *request.Seed
	}

	workflow, err := s.template.Build(
		request.Prompt,
		request.NegativePrompt,
		seed,
	)
	if err != nil {
		return domain.GenerateResponse{}, err
	}

	promptID, err := s.client.Submit(ctx, workflow)
	if err != nil {
		return domain.GenerateResponse{}, err
	}

	return domain.GenerateResponse{
		JobID:    promptID,
		PromptID: promptID,
		Status:   "queued",
		Seed:     seed,
	}, nil
}

func (s *PosterService) Status(
	ctx context.Context,
	jobID string,
) (domain.JobResponse, error) {
	status, image, messages, err :=
		s.client.Inspect(ctx, jobID)

	if err != nil {
		return domain.JobResponse{}, err
	}

	result := domain.JobResponse{
		JobID:    jobID,
		Status:   status,
		Image:    image,
		Messages: messages,
	}

	if image != nil {
		result.ResultURL = "/api/jobs/" +
			jobID +
			"/result"
	}

	return result, nil
}

func (s *PosterService) OpenResult(
	ctx context.Context,
	jobID string,
) (*http.Response, error) {
	status, image, _, err :=
		s.client.Inspect(ctx, jobID)

	if err != nil {
		return nil, err
	}

	if status == "failed" {
		return nil, errors.New("generation failed")
	}

	if status != "succeeded" || image == nil {
		return nil, errors.New("result is not ready")
	}

	return s.client.OpenImage(ctx, *image)
}
