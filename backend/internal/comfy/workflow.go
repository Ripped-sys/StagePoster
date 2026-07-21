package comfy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Binding struct {
	NodeID   string `json:"nodeId"`
	InputKey string `json:"inputKey"`
}

type Bindings struct {
	Prompt         *Binding `json:"prompt"`
	NegativePrompt *Binding `json:"negativePrompt,omitempty"`
	Seed           *Binding `json:"seed,omitempty"`
}

type Template struct {
	base     map[string]any
	bindings Bindings
}

func LoadTemplate(
	path string,
	promptNodeID string,
	negativePromptNodeID string,
	seedNodeID string,
) (*Template, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open workflow: %w", err)
	}
	defer file.Close()

	var workflow map[string]any
	if err := json.NewDecoder(file).Decode(&workflow); err != nil {
		return nil, fmt.Errorf("decode workflow: %w", err)
	}

	if len(workflow) == 0 {
		return nil, errors.New("workflow is empty")
	}

	prompt, err := findTextBinding(workflow, promptNodeID, false)
	if err != nil {
		return nil, err
	}

	if prompt == nil {
		return nil, errors.New(
			"cannot detect positive prompt node; set PROMPT_NODE_ID",
		)
	}

	negative, err := findTextBinding(
		workflow,
		negativePromptNodeID,
		true,
	)
	if err != nil {
		return nil, err
	}

	seed, err := findSeedBinding(workflow, seedNodeID)
	if err != nil {
		return nil, err
	}

	return &Template{
		base: workflow,
		bindings: Bindings{
			Prompt:         prompt,
			NegativePrompt: negative,
			Seed:           seed,
		},
	}, nil
}

func (t *Template) Bindings() Bindings {
	return t.bindings
}

func (t *Template) Build(
	prompt string,
	negativePrompt string,
	seed int64,
) (map[string]any, error) {
	workflow, err := cloneWorkflow(t.base)
	if err != nil {
		return nil, err
	}

	if err := applyBinding(
		workflow,
		t.bindings.Prompt,
		prompt,
	); err != nil {
		return nil, err
	}

	if negativePrompt != "" && t.bindings.NegativePrompt != nil {
		if err := applyBinding(
			workflow,
			t.bindings.NegativePrompt,
			negativePrompt,
		); err != nil {
			return nil, err
		}
	}

	if t.bindings.Seed != nil {
		if err := applyBinding(
			workflow,
			t.bindings.Seed,
			seed,
		); err != nil {
			return nil, err
		}
	}

	return workflow, nil
}

func cloneWorkflow(source map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(source)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow clone: %w", err)
	}

	var cloned map[string]any
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return nil, fmt.Errorf("unmarshal workflow clone: %w", err)
	}

	return cloned, nil
}

func applyBinding(
	workflow map[string]any,
	binding *Binding,
	value any,
) error {
	if binding == nil {
		return errors.New("workflow binding is missing")
	}

	node, ok := workflow[binding.NodeID].(map[string]any)
	if !ok {
		return fmt.Errorf(
			"workflow node %s not found",
			binding.NodeID,
		)
	}

	inputs, ok := node["inputs"].(map[string]any)
	if !ok {
		return fmt.Errorf(
			"workflow node %s has no inputs",
			binding.NodeID,
		)
	}

	inputs[binding.InputKey] = value
	return nil
}

func findTextBinding(
	workflow map[string]any,
	explicitNodeID string,
	negative bool,
) (*Binding, error) {
	if explicitNodeID != "" {
		return explicitTextBinding(workflow, explicitNodeID)
	}

	var best *Binding
	bestScore := -1_000_000

	for _, nodeID := range sortedNodeIDs(workflow) {
		node, ok := workflow[nodeID].(map[string]any)
		if !ok {
			continue
		}

		inputs, _ := node["inputs"].(map[string]any)
		description := strings.ToLower(nodeDescription(node))

		for _, key := range []string{
			"text",
			"prompt",
			"positive",
			"negative_prompt",
		} {
			value, exists := inputs[key]
			if !exists {
				continue
			}

			if _, ok := value.(string); !ok {
				continue
			}

			score := 10

			if key == "text" {
				score += 20
			}

			if strings.Contains(description, "prompt") {
				score += 30
			}

			if strings.Contains(description, "textencode") ||
				strings.Contains(description, "text encode") {
				score += 30
			}

			if negative {
				if strings.Contains(description, "negative") {
					score += 200
				}

				if strings.Contains(description, "positive") {
					score -= 200
				}
			} else {
				if strings.Contains(description, "positive") {
					score += 200
				}

				if strings.Contains(description, "negative") {
					score -= 300
				}
			}

			if score > bestScore {
				bestScore = score
				best = &Binding{
					NodeID:   nodeID,
					InputKey: key,
				}
			}
		}
	}

	if negative && bestScore < 150 {
		return nil, nil
	}

	return best, nil
}

func explicitTextBinding(
	workflow map[string]any,
	nodeID string,
) (*Binding, error) {
	node, ok := workflow[nodeID].(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"text node %s does not exist",
			nodeID,
		)
	}

	inputs, _ := node["inputs"].(map[string]any)

	for _, key := range []string{
		"text",
		"prompt",
		"positive",
		"negative_prompt",
	} {
		if _, exists := inputs[key]; exists {
			return &Binding{
				NodeID:   nodeID,
				InputKey: key,
			}, nil
		}
	}

	return nil, fmt.Errorf(
		"node %s has no supported text input",
		nodeID,
	)
}

func findSeedBinding(
	workflow map[string]any,
	explicitNodeID string,
) (*Binding, error) {
	if explicitNodeID != "" {
		node, ok := workflow[explicitNodeID].(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"seed node %s does not exist",
				explicitNodeID,
			)
		}

		inputs, _ := node["inputs"].(map[string]any)

		for _, key := range []string{
			"seed",
			"noise_seed",
		} {
			if _, exists := inputs[key]; exists {
				return &Binding{
					NodeID:   explicitNodeID,
					InputKey: key,
				}, nil
			}
		}

		return nil, fmt.Errorf(
			"node %s has no seed input",
			explicitNodeID,
		)
	}

	var best *Binding
	bestScore := -1

	for _, nodeID := range sortedNodeIDs(workflow) {
		node, ok := workflow[nodeID].(map[string]any)
		if !ok {
			continue
		}

		inputs, _ := node["inputs"].(map[string]any)
		description := strings.ToLower(nodeDescription(node))

		for _, key := range []string{
			"seed",
			"noise_seed",
		} {
			if _, exists := inputs[key]; !exists {
				continue
			}

			score := 10

			if strings.Contains(description, "sampler") {
				score += 100
			}

			if strings.Contains(description, "noise") {
				score += 80
			}

			if score > bestScore {
				bestScore = score
				best = &Binding{
					NodeID:   nodeID,
					InputKey: key,
				}
			}
		}
	}

	return best, nil
}

func nodeDescription(node map[string]any) string {
	classType, _ := node["class_type"].(string)
	description := classType

	meta, _ := node["_meta"].(map[string]any)
	if meta != nil {
		title, _ := meta["title"].(string)
		description += " " + title
	}

	return description
}

func sortedNodeIDs(workflow map[string]any) []string {
	ids := make([]string, 0, len(workflow))

	for id := range workflow {
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids
}
