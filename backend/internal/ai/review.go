package ai

import (
	"strings"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func NormalizeReview(
	result *domain.ReviewResult,
) error {

	if !result.Decision.Valid() {

		result.Decision =
			domain.ReviewDecisionRecompose

	}

	for _, issue := range result.Issues {

		text :=
			strings.ToLower(
				issue.Code +
					" " +
					issue.Description,
			)

		if strings.Contains(
			text,
			"gibberish",
		) ||
			strings.Contains(
				text,
				"乱码",
			) ||
			strings.Contains(
				text,
				"无意义",
			) {

			result.Decision =
				domain.ReviewDecisionRegenerate

			break
		}
	}

	return nil
}
