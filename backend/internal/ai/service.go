package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"unicode"

	"github.com/Ripped-sys/StagePoster/backend/internal/domain"
)

const designSystemPrompt = `You are StagePoster Design Agent, an expert art director for music-event posters.

Return exactly one valid JSON object.
Do not use Markdown.
Do not use code fences.
Generate exactly three clearly different visual plans.

Important generation constraints:
- The generated key visual must contain no readable text.
- Do not generate letters, words, captions, logos, signage or watermarks.
- StagePoster will add exact event text later through deterministic composition.
- Reserve a clean upper area for the title.
- Reserve a low-detail lower area for event information.
- positivePrompt and negativePrompt must be written in English.

Use exactly this schema:
{
  "reply": "brief Chinese response",
  "state": "awaiting_plan_selection",
  "missingFields": [],
  "plans": [
    {
      "id": "lowercase-kebab-id",
      "name": "Chinese plan name",
      "concept": "Chinese visual concept",
      "palette": ["#000000", "#FFFFFF", "#8B171E"],
      "composition": {
        "subject": "center",
        "symmetry": "strong",
        "titleSafeZone": "top_20_percent",
        "informationSafeZone": "bottom_22_percent"
      },
      "positivePrompt": "English generation prompt",
      "negativePrompt": "English negative prompt",
      "composerTemplate": "editorial_top"
    }
  ]
}

Allowed composerTemplate values:
- editorial_top
- cinematic_center
- gothic_frame`

const reviewSystemPrompt = `You are StagePoster Visual Critic.

Inspect the attached final poster and compare it with the supplied event brief, visual brief and selected design plan.

Return exactly one valid JSON object.
Do not use Markdown.
Do not use code fences.
Use scores from 0 to 100.

Decision policy:
- ACCEPT only when totalScore is at least 82 and no hard failure exists.
- RECOMPOSE for layout, title placement, hierarchy, spacing, font size, information panel, contrast or readability problems.
- REGENERATE when the generated key visual contains gibberish text, malformed letters, unwanted text, watermark, unwanted logo, wrong subject or severe visual artifacts.
- REWRITE_BRIEF only when the supplied creative direction is contradictory or unusable.

Use exactly this schema:
{
  "totalScore": 0,
  "scores": {
    "requirementAlignment": 0,
    "composition": 0,
    "typography": 0,
    "readability": 0,
    "visualQuality": 0,
    "brandConsistency": 0
  },
  "hardFailures": [
    {
      "code": "GENERATED_GIBBERISH_TEXT",
      "description": "Chinese description"
    }
  ],
  "issues": [
    {
      "code": "TITLE_COLLISION",
      "severity": "high",
      "layer": "composition",
      "description": "Chinese description",
      "suggestion": "Chinese actionable suggestion"
    }
  ],
  "decision": "RECOMPOSE",
  "nextInstruction": {
    "promptAdditions": [],
    "negativePromptAdditions": [],
    "composerTemplate": "cinematic_center"
  }
}`

type Service struct {
	client *Client
}

func NewService(
	client *Client,
) *Service {
	return &Service{
		client: client,
	}
}

func (service *Service) Health(
	ctx context.Context,
) error {
	return service.client.Health(ctx)
}

func (service *Service) Plan(
	ctx context.Context,
	event domain.EventBrief,
	visual domain.VisualBrief,
	userMessage string,
) (
	domain.DesignAgentResult,
	Metrics,
	error,
) {
	input := struct {
		Event       domain.EventBrief  `json:"event"`
		Visual      domain.VisualBrief `json:"visual"`
		UserMessage string             `json:"userMessage"`
	}{
		Event:       event,
		Visual:      visual,
		UserMessage: strings.TrimSpace(userMessage),
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		return domain.DesignAgentResult{},
			Metrics{},
			fmt.Errorf(
				"encode design request: %w",
				err,
			)
	}

	var result domain.DesignAgentResult

	metrics, err := service.client.CompleteJSON(
		ctx,
		[]Message{
			{
				Role:    "system",
				Content: designSystemPrompt,
			},
			{
				Role:    "user",
				Content: string(encoded),
			},
		},
		CompletionOptions{
			Temperature: 0.65,
			MaxTokens:   1400,
		},
		&result,
	)
	if err != nil {
		return domain.DesignAgentResult{},
			metrics,
			err
	}

	if err := NormalizeDesignResult(&result); err != nil {
		return domain.DesignAgentResult{},
			metrics,
			fmt.Errorf(
				"normalize design result: %w",
				err,
			)
	}

	return result, metrics, nil
}

func (service *Service) Review(
	ctx context.Context,
	imagePath string,
	event domain.EventBrief,
	visual domain.VisualBrief,
	designPlan *domain.DesignPlan,
) (
	domain.ReviewResult,
	Metrics,
	error,
) {
	imageURL, err := imageDataURL(imagePath)
	if err != nil {
		return domain.ReviewResult{},
			Metrics{},
			err
	}

	input := struct {
		Event      domain.EventBrief  `json:"event"`
		Visual     domain.VisualBrief `json:"visual"`
		DesignPlan *domain.DesignPlan `json:"designPlan,omitempty"`
		Rules      []string           `json:"rules"`
	}{
		Event:      event,
		Visual:     visual,
		DesignPlan: designPlan,
		Rules: []string{
			"Exact event text must remain readable.",
			"Generated key visual must not contain gibberish typography.",
			"Upper title zone must remain usable.",
			"Lower information zone must remain usable.",
		},
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		return domain.ReviewResult{},
			Metrics{},
			fmt.Errorf(
				"encode review request: %w",
				err,
			)
	}

	content := []map[string]any{
		{
			"type": "text",
			"text": string(encoded),
		},
		{
			"type": "image_url",
			"image_url": map[string]string{
				"url": imageURL,
			},
		},
	}

	var result domain.ReviewResult

	metrics, err := service.client.CompleteJSON(
		ctx,
		[]Message{
			{
				Role:    "system",
				Content: reviewSystemPrompt,
			},
			{
				Role:    "user",
				Content: content,
			},
		},
		CompletionOptions{
			Temperature: 0.1,
			MaxTokens:   900,
		},
		&result,
	)
	if err != nil {
		return domain.ReviewResult{},
			metrics,
			err
	}

	if err := NormalizeReview(&result); err != nil {
		return domain.ReviewResult{},
			metrics,
			fmt.Errorf(
				"normalize review result: %w",
				err,
			)
	}

	return result, metrics, nil
}

func NormalizeDesignResult(
	result *domain.DesignAgentResult,
) error {
	if result == nil {
		return fmt.Errorf("design result is nil")
	}

	result.Reply = strings.TrimSpace(result.Reply)
	result.State = strings.TrimSpace(result.State)

	if result.State == "" {
		result.State = "awaiting_plan_selection"
	}

	if len(result.Plans) != 3 {
		return fmt.Errorf(
			"expected exactly 3 plans, got %d",
			len(result.Plans),
		)
	}

	seen := make(map[string]struct{}, 3)

	for index := range result.Plans {
		plan := &result.Plans[index]

		plan.ID = normalizePlanID(plan.ID)

		if plan.ID == "" {
			plan.ID = fmt.Sprintf(
				"plan-%d",
				index+1,
			)
		}

		if _, exists := seen[plan.ID]; exists {
			plan.ID = fmt.Sprintf(
				"%s-%d",
				plan.ID,
				index+1,
			)
		}

		seen[plan.ID] = struct{}{}

		plan.Name = strings.TrimSpace(plan.Name)
		plan.Concept = strings.TrimSpace(plan.Concept)

		plan.PositivePrompt = strings.TrimSpace(
			plan.PositivePrompt,
		)

		plan.NegativePrompt = strings.TrimSpace(
			plan.NegativePrompt,
		)

		if plan.Name == "" {
			return fmt.Errorf(
				"plan %d has empty name",
				index+1,
			)
		}

		if plan.Concept == "" {
			return fmt.Errorf(
				"plan %d has empty concept",
				index+1,
			)
		}

		if plan.PositivePrompt == "" {
			return fmt.Errorf(
				"plan %d has empty positivePrompt",
				index+1,
			)
		}

		plan.ComposerTemplate = normalizeComposerTemplate(
			plan.ComposerTemplate,
		)

		if plan.Composition.Subject == "" {
			plan.Composition.Subject = "center"
		}

		if plan.Composition.Symmetry == "" {
			plan.Composition.Symmetry = "balanced"
		}

		if plan.Composition.TitleSafeZone == "" {
			plan.Composition.TitleSafeZone =
				"top_20_percent"
		}

		if plan.Composition.InformationSafeZone == "" {
			plan.Composition.InformationSafeZone =
				"bottom_22_percent"
		}

		plan.PositivePrompt =
			appendPromptFragment(
				plan.PositivePrompt,
				"clean empty upper title area",
			)

		plan.PositivePrompt =
			appendPromptFragment(
				plan.PositivePrompt,
				"low-detail lower information area",
			)

		negativeFragments := []string{
			"text",
			"letters",
			"words",
			"typography",
			"caption",
			"logo",
			"watermark",
			"signage",
			"gibberish",
		}

		for _, fragment := range negativeFragments {
			plan.NegativePrompt =
				appendPromptFragment(
					plan.NegativePrompt,
					fragment,
				)
		}
	}

	return nil
}

func imageDataURL(
	path string,
) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf(
			"stat review image: %w",
			err,
		)
	}

	if info.IsDir() {
		return "", fmt.Errorf(
			"review image is a directory",
		)
	}

	if info.Size() > 20<<20 {
		return "", fmt.Errorf(
			"review image exceeds 20 MiB",
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf(
			"read review image: %w",
			err,
		)
	}

	if len(data) == 0 {
		return "", fmt.Errorf(
			"review image is empty",
		)
	}

	contentType := http.DetectContentType(data)

	switch contentType {
	case "image/png",
		"image/jpeg",
		"image/webp",
		"image/gif":

	default:
		return "", fmt.Errorf(
			"unsupported review image type %q",
			contentType,
		)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	return "data:" +
			contentType +
			";base64," +
			encoded,
		nil
}

func normalizePlanID(
	value string,
) string {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	var builder strings.Builder
	previousDash := false

	for _, current := range value {
		switch {
		case unicode.IsLetter(current),
			unicode.IsDigit(current):

			builder.WriteRune(current)
			previousDash = false

		case builder.Len() > 0 &&
			!previousDash:

			builder.WriteByte('-')
			previousDash = true
		}
	}

	return strings.Trim(
		builder.String(),
		"-",
	)
}

func normalizeComposerTemplate(
	value string,
) string {
	switch strings.ToLower(
		strings.TrimSpace(value),
	) {
	case "editorial_top":
		return "editorial_top"

	case "gothic_frame":
		return "gothic_frame"

	default:
		return "cinematic_center"
	}
}

func appendPromptFragment(
	prompt string,
	fragment string,
) string {
	prompt = strings.TrimSpace(prompt)
	fragment = strings.TrimSpace(fragment)

	if fragment == "" {
		return prompt
	}

	if strings.Contains(
		strings.ToLower(prompt),
		strings.ToLower(fragment),
	) {
		return prompt
	}

	if prompt == "" {
		return fragment
	}

	return strings.TrimRight(prompt, " ,;.") +
		", " +
		fragment
}
