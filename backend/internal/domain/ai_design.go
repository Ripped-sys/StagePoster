package domain

type DesignComposition struct {
	Subject             string `json:"subject"`
	Symmetry            string `json:"symmetry"`
	TitleSafeZone       string `json:"titleSafeZone"`
	InformationSafeZone string `json:"informationSafeZone"`
}

type DesignPlan struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Concept          string            `json:"concept"`
	Palette          []string          `json:"palette"`
	Composition      DesignComposition `json:"composition"`
	PositivePrompt   string            `json:"positivePrompt"`
	NegativePrompt   string            `json:"negativePrompt"`
	ComposerTemplate string            `json:"composerTemplate"`
}

type DesignAgentResult struct {
	Reply         string       `json:"reply"`
	State         string       `json:"state"`
	MissingFields []string     `json:"missingFields"`
	Plans         []DesignPlan `json:"plans"`
}
