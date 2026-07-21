package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Score int

func (s Score) Int() int {
	return int(s)
}

func (s *Score) UnmarshalJSON(
	data []byte,
) error {

	raw :=
		strings.TrimSpace(
			string(data),
		)

	var value float64

	// 数字
	if err := json.Unmarshal(
		data,
		&value,
	); err != nil {

		// 字符串数字
		parsed, err :=
			strconv.ParseFloat(
				strings.Trim(
					raw,
					"\"",
				),
				64,
			)

		if err != nil {
			return fmt.Errorf(
				"invalid score %s",
				raw,
			)
		}

		value = parsed
	}

	// 模型习惯输出 0-10
	// 自动转百分制
	if value > 0 &&
		value <= 10 {

		value *= 10
	}

	*s =
		Score(
			int(value + 0.5),
		)

	return nil
}

type ReviewDecision string

const (
	ReviewDecisionAccept       ReviewDecision = "ACCEPT"
	ReviewDecisionRecompose    ReviewDecision = "RECOMPOSE"
	ReviewDecisionRegenerate   ReviewDecision = "REGENERATE"
	ReviewDecisionRewriteBrief ReviewDecision = "REWRITE_BRIEF"
)

func (d ReviewDecision) Valid() bool {
	switch d {
	case ReviewDecisionAccept,
		ReviewDecisionRecompose,
		ReviewDecisionRegenerate,
		ReviewDecisionRewriteBrief:
		return true
	default:
		return false
	}
}

type ReviewFailure struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

type ReviewIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Layer    string `json:"layer"`

	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type ReviewScores struct {
	RequirementAlignment Score `json:"requirementAlignment"`

	Composition Score `json:"composition"`

	Typography Score `json:"typography"`

	Readability Score `json:"readability"`

	VisualQuality Score `json:"visualQuality"`

	BrandConsistency Score `json:"brandConsistency"`
}

type ReviewNextInstruction struct {
	PromptAdditions []string `json:"promptAdditions"`

	NegativePromptAdditions []string `json:"negativePromptAdditions"`

	ComposerTemplate string `json:"composerTemplate"`
}

type ReviewResult struct {
	TotalScore Score `json:"totalScore"`

	Scores ReviewScores `json:"scores"`

	HardFailures []ReviewFailure `json:"hardFailures"`

	Issues []ReviewIssue `json:"issues"`

	Decision ReviewDecision `json:"decision"`

	NextInstruction ReviewNextInstruction `json:"nextInstruction"`
}
