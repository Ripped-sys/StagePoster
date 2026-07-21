package ai

import (
	"strings"
	"testing"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func TestNormalizeDesignResult(
	t *testing.T,
) {
	result := domain.DesignAgentResult{
		State: "awaiting_plan_selection",
		Plans: []domain.DesignPlan{
			{
				ID:             "Deep Gaze",
				Name:           "深渊凝视",
				Concept:        "极简深渊视觉",
				PositivePrompt: "dark abyssal eye",
			},
			{
				ID:             "Bone Carnival",
				Name:           "骸骨狂欢",
				Concept:        "高动态哥特视觉",
				PositivePrompt: "gothic skeletal carnival",
			},
			{
				ID:             "Silent Tide",
				Name:           "静默潮汐",
				Concept:        "深海超现实视觉",
				PositivePrompt: "surreal black ocean",
			},
		},
	}

	if err := NormalizeDesignResult(&result); err != nil {
		t.Fatalf(
			"NormalizeDesignResult error: %v",
			err,
		)
	}

	if len(result.Plans) != 3 {
		t.Fatalf(
			"expected 3 plans, got %d",
			len(result.Plans),
		)
	}

	if result.Plans[0].ID != "deep-gaze" {
		t.Fatalf(
			"unexpected normalized ID %q",
			result.Plans[0].ID,
		)
	}

	if !strings.Contains(
		result.Plans[0].NegativePrompt,
		"watermark",
	) {
		t.Fatalf(
			"negative prompt missing watermark",
		)
	}

	if !strings.Contains(
		result.Plans[0].PositivePrompt,
		"upper title area",
	) {
		t.Fatalf(
			"positive prompt missing title safe zone",
		)
	}
}

func TestLowScoreCannotAccept(
	t *testing.T,
) {
	result := domain.ReviewResult{
		TotalScore: domain.Score(70),
		Decision:   domain.ReviewDecisionAccept,
	}

	if err := NormalizeReview(&result); err != nil {
		t.Fatalf(
			"NormalizeReview error: %v",
			err,
		)
	}

	if result.Decision !=
		domain.ReviewDecisionRecompose {
		t.Fatalf(
			"expected RECOMPOSE, got %s",
			result.Decision,
		)
	}
}

func TestHighScoreCanAccept(
	t *testing.T,
) {
	result := domain.ReviewResult{
		TotalScore: domain.Score(86),
		Decision:   domain.ReviewDecisionRecompose,
	}

	if err := NormalizeReview(&result); err != nil {
		t.Fatalf(
			"NormalizeReview error: %v",
			err,
		)
	}

	if result.Decision !=
		domain.ReviewDecisionAccept {
		t.Fatalf(
			"expected ACCEPT, got %s",
			result.Decision,
		)
	}
}
