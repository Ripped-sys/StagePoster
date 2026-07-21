package ai

import (
	"fmt"
	"strings"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

func NormalizeReview(
	result *domain.ReviewResult,
) error {
	if result == nil {
		return fmt.Errorf("review result is nil")
	}

	if !result.Decision.Valid() {
		return fmt.Errorf(
			"invalid review decision %q",
			result.Decision,
		)
	}

	if len(result.Issues) > 10 {
		result.Issues = result.Issues[:10]
	}

	hasGenerationFailure := false
	hasCompositionFailure := false

	for index := range result.HardFailures {
		failure := &result.HardFailures[index]

		failure.Code = strings.ToUpper(
			strings.TrimSpace(failure.Code),
		)

		failure.Description = strings.TrimSpace(
			failure.Description,
		)

		if indicatesGenerationFailure(
			failure.Code + " " + failure.Description,
		) {
			hasGenerationFailure = true
		}
	}

	for index := range result.Issues {
		issue := &result.Issues[index]

		issue.Code = strings.ToUpper(
			strings.TrimSpace(issue.Code),
		)

		issue.Severity = normalizeSeverity(
			issue.Severity,
		)

		issue.Layer = strings.ToLower(
			strings.TrimSpace(issue.Layer),
		)

		issue.Description = strings.TrimSpace(
			issue.Description,
		)

		issue.Suggestion = strings.TrimSpace(
			issue.Suggestion,
		)

		combined := issue.Code +
			" " +
			issue.Description

		if indicatesGenerationFailure(combined) {
			hasGenerationFailure = true
		}

		if indicatesCompositionFailure(combined) {
			hasCompositionFailure = true
		}
	}

	switch {
	case hasGenerationFailure:
		result.Decision =
			domain.ReviewDecisionRegenerate

	case len(result.HardFailures) > 0 &&
		result.Decision ==
			domain.ReviewDecisionAccept:
		result.Decision =
			domain.ReviewDecisionRegenerate

	case result.TotalScore.Int() >= 82 &&
		len(result.HardFailures) == 0 &&
		!hasCompositionFailure:
		result.Decision =
			domain.ReviewDecisionAccept

	case result.Decision ==
		domain.ReviewDecisionAccept:
		result.Decision =
			domain.ReviewDecisionRecompose
	}

	return nil
}

func normalizeSeverity(
	value string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "high":
		return "high"

	case "low":
		return "low"

	default:
		return "medium"
	}
}

func indicatesGenerationFailure(
	value string,
) bool {
	value = strings.ToLower(value)

	keywords := []string{
		"generated_gibberish_text",
		"gibberish",
		"malformed_text",
		"unwanted_text",
		"unwanted_logo",
		"watermark",
		"wrong_subject",
		"severe_artifact",
		"乱码",
		"无意义文字",
		"伪文字",
		"错误文字",
		"生成文字",
		"水印",
		"畸形文字",
		"主体错误",
	}

	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}

	return false
}

func indicatesCompositionFailure(
	value string,
) bool {
	value = strings.ToLower(value)

	keywords := []string{
		"title_collision",
		"layout",
		"typography",
		"readability",
		"information_panel",
		"spacing",
		"hierarchy",
		"contrast",
		"标题重叠",
		"排版",
		"字号",
		"可读性",
		"信息栏",
		"层级",
		"间距",
	}

	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}

	return false
}
