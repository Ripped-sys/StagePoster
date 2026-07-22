package poster

import (
	"testing"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func TestReviewAdjustments(t *testing.T) {
	t.Parallel()

	adjustments := reviewAdjustments(
		domain.ReviewResult{
			Issues: []domain.ReviewIssue{
				{
					Code: "TITLE_COLLISION",
				},
				{
					Code: "INFORMATION_PANEL_CONTRAST",
				},
			},
			NextInstruction: domain.ReviewNextInstruction{
				ComposerTemplate: "cinematic_center",
			},
		},
	)

	if adjustments.Template != "cinematic_center" {
		t.Fatalf(
			"template = %q",
			adjustments.Template,
		)
	}

	if adjustments.TitleOffsetRatio < 0.05 {
		t.Fatalf(
			"title offset = %f",
			adjustments.TitleOffsetRatio,
		)
	}

	if adjustments.PanelTopRatio < 0.80 {
		t.Fatalf(
			"panel top = %f",
			adjustments.PanelTopRatio,
		)
	}

	if adjustments.PanelTheme != "dark" {
		t.Fatalf(
			"panel theme = %q",
			adjustments.PanelTheme,
		)
	}
}
