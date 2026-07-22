package poster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

type designPlanVariant struct {
	Spec           domain.CandidateSpec
	PromptAddition string
}

func (s *Service) CreateFromDesignPlan(
	ctx context.Context,
	request domain.CreatePosterRequest,
	plan domain.DesignPlan,
) (domain.PosterResponse, error) {
	request.Event.Title = strings.TrimSpace(
		request.Event.Title,
	)

	if request.Event.Title == "" {
		return domain.PosterResponse{},
			fmt.Errorf(
				"%w: event title is required",
				ErrInvalidPosterBrief,
			)
	}

	if strings.TrimSpace(plan.PositivePrompt) == "" {
		return domain.PosterResponse{},
			fmt.Errorf(
				"%w: design plan positive prompt is required",
				ErrInvalidPosterBrief,
			)
	}

	if strings.TrimSpace(request.Visual.Style) == "" {
		request.Visual.Style = plan.ComposerTemplate
	}

	variants := buildDesignPlanVariants(plan)

	if len(variants) != 3 {
		return domain.PosterResponse{},
			errors.New(
				"design plan must produce exactly 3 candidates",
			)
	}

	posterID, err := domain.NewID("poster_")
	if err != nil {
		return domain.PosterResponse{}, err
	}

	goal := domain.GoalContract{
		Width:               1024,
		Height:              1536,
		AllowPeople:         false,
		AllowReadableText:   false,
		RequireCentralMotif: true,
		MaxAttempts:         2,
	}

	eventJSON, err := marshalJSON(request.Event)
	if err != nil {
		return domain.PosterResponse{}, err
	}

	brandingJSON, err := marshalJSON(request.Branding)
	if err != nil {
		return domain.PosterResponse{}, err
	}

	visualJSON, err := marshalJSON(request.Visual)
	if err != nil {
		return domain.PosterResponse{}, err
	}

	goalJSON, err := marshalJSON(goal)
	if err != nil {
		return domain.PosterResponse{}, err
	}

	now := time.Now().UTC()

	posterRecord := domain.PosterRecord{
		ID:           posterID,
		Status:       domain.PosterStatusPlanning,
		StyleKey:     request.Visual.Style,
		EventJSON:    eventJSON,
		BrandingJSON: brandingJSON,
		VisualJSON:   visualJSON,
		GoalJSON:     goalJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repository.CreatePoster(
		ctx,
		posterRecord,
	); err != nil {
		return domain.PosterResponse{}, err
	}

	baseSeed := time.Now().UnixNano() &
		0x7fffffffffffffff

	for index, variant := range variants {
		compiledPrompt := buildDesignPlanPrompt(
			plan,
			variant.PromptAddition,
		)

		seed := baseSeed +
			int64((index+1)*100003)

		generation, err := s.core.Generate(
			ctx,
			domain.GenerateRequest{
				Prompt: compiledPrompt,
				NegativePrompt: buildDesignPlanNegativePrompt(
					plan,
				),
				Seed: &seed,
			},
		)
		if err != nil {
			_ = s.repository.UpdatePosterStatus(
				context.Background(),
				posterID,
				domain.PosterStatusFailed,
				err.Error(),
			)

			return domain.PosterResponse{}, err
		}

		candidateID, err := domain.NewID("candidate_")
		if err != nil {
			return domain.PosterResponse{}, err
		}

		specJSON, err := marshalJSON(variant.Spec)
		if err != nil {
			return domain.PosterResponse{}, err
		}

		candidateNow := time.Now().UTC()

		candidate := domain.CandidateRecord{
			ID:             candidateID,
			PosterID:       posterID,
			JobID:          generation.JobID,
			VariantIndex:   index,
			VariantKey:     variant.Spec.VariantKey,
			VariantName:    variant.Spec.VariantName,
			SpecJSON:       specJSON,
			CompiledPrompt: compiledPrompt,
			Seed:           seed,
			Attempt:        1,
			Status:         domain.CandidateStatusGenerating,
			Passed:         false,
			Selected:       false,
			CreatedAt:      candidateNow,
			UpdatedAt:      candidateNow,
		}

		if err := s.repository.CreateCandidate(
			ctx,
			candidate,
		); err != nil {
			_ = s.repository.UpdatePosterStatus(
				context.Background(),
				posterID,
				domain.PosterStatusFailed,
				err.Error(),
			)

			return domain.PosterResponse{}, err
		}
	}

	if err := s.repository.UpdatePosterStatus(
		ctx,
		posterID,
		domain.PosterStatusGenerating,
		"",
	); err != nil {
		return domain.PosterResponse{}, err
	}

	return s.Get(ctx, posterID)
}

func buildDesignPlanVariants(
	plan domain.DesignPlan,
) []designPlanVariant {
	palette := append(
		[]string(nil),
		plan.Palette...,
	)

	composition := strings.Join(
		[]string{
			"subject " + plan.Composition.Subject,
			"symmetry " + plan.Composition.Symmetry,
			"title safe zone " + plan.Composition.TitleSafeZone,
			"information safe zone " +
				plan.Composition.InformationSafeZone,
		},
		", ",
	)

	return []designPlanVariant{
		{
			Spec: domain.CandidateSpec{
				VariantKey: plan.ID + "-balanced",
				VariantName: plan.Name +
					" · Balanced",
				Motif:       plan.Concept,
				Composition: composition,
				Materials: []string{
					"cinematic editorial texture",
					"professional concert poster finish",
				},
				Palette:  palette,
				Lighting: "balanced cinematic lighting with controlled contrast",
			},
			PromptAddition: "balanced editorial interpretation, clear central hierarchy, controlled cinematic contrast",
		},
		{
			Spec: domain.CandidateSpec{
				VariantKey: plan.ID + "-dramatic",
				VariantName: plan.Name +
					" · Dramatic",
				Motif:       plan.Concept,
				Composition: composition,
				Materials: []string{
					"high-contrast cinematic texture",
					"rich atmospheric depth",
				},
				Palette:  palette,
				Lighting: "dramatic directional lighting with deep dimensional shadows",
			},
			PromptAddition: "more dramatic lighting, stronger depth, more monumental visual impact",
		},
		{
			Spec: domain.CandidateSpec{
				VariantKey: plan.ID + "-graphic",
				VariantName: plan.Name +
					" · Graphic",
				Motif:       plan.Concept,
				Composition: composition,
				Materials: []string{
					"refined graphic poster treatment",
					"clean silhouette separation",
				},
				Palette:  palette,
				Lighting: "graphic rim lighting with clean silhouette separation",
			},
			PromptAddition: "cleaner graphic silhouette, stronger negative space, poster-first readability",
		},
	}
}

func buildDesignPlanPrompt(
	plan domain.DesignPlan,
	addition string,
) string {
	fragments := []string{
		strings.TrimSpace(plan.PositivePrompt),
		strings.TrimSpace(addition),
		"professional vertical 2:3 music event key visual",
		"one coherent visual concept",
		"clean empty upper title area",
		"low-detail lower information area",
		"no readable text",
		"no letters",
		"no words",
		"no logos",
		"no watermarks",
	}

	return strings.Join(
		nonEmptyPromptFragments(fragments),
		", ",
	)
}

func buildDesignPlanNegativePrompt(
	plan domain.DesignPlan,
) string {
	fragments := []string{
		strings.TrimSpace(plan.NegativePrompt),
		"text",
		"letters",
		"words",
		"captions",
		"typography",
		"gibberish",
		"logos",
		"watermarks",
		"signage",
		"UI",
		"mockup",
		"low resolution",
	}

	return strings.Join(
		nonEmptyPromptFragments(fragments),
		", ",
	)
}

func nonEmptyPromptFragments(
	values []string,
) []string {
	result := make(
		[]string,
		0,
		len(values),
	)

	seen := make(map[string]struct{})

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value == "" {
			continue
		}

		key := strings.ToLower(value)

		if _, exists := seen[key]; exists {
			continue
		}

		seen[key] = struct{}{}
		result = append(result, value)
	}

	return result
}
