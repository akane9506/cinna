package prompt

import (
	_ "embed"
	"log/slog"
	"strings"
)

const (
	cinnaPersonaPath         = "cinna_persona.md"
	intentClassificationPath = "intent_classification.md"
)

//go:embed cinna_persona.md
var cinnaPersonaPrompt string

//go:embed intent_classification.md
var intentClassificationPrompt string

type Prompts struct {
	CinnaPersona         string
	IntentClassification string
	logger               *slog.Logger
}

func LoadPromptFiles(logger *slog.Logger) *Prompts {
	prompts := &Prompts{logger: logger}
	cinnaPersonaPrompt := prompts.loadEmbeddedPrompt(cinnaPersonaPath, cinnaPersonaPrompt)
	intentClassificationPrompt := prompts.loadEmbeddedPrompt(intentClassificationPath, intentClassificationPrompt)
	prompts.CinnaPersona = cinnaPersonaPrompt
	prompts.IntentClassification = intentClassificationPrompt
	return prompts
}

// load embedded prompts
func (p *Prompts) loadEmbeddedPrompt(filename string, content string) string {
	logger := p.logger.With("path", "internal/app/agent/prompt/prompts/loadEmbeddedPrompts")
	trimmedPrompt := strings.TrimSpace(string(content))
	if trimmedPrompt == "" {
		logger.Error("the prompt file is empty", "file", filename)
	}
	return trimmedPrompt
}
