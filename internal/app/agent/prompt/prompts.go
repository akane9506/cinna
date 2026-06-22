package prompt

import (
	_ "embed"
	"log/slog"
	"strings"
)

const (
	cinnaPersonaPath         = "cinna_persona.md"
	intentClassificationPath = "intent_classification.md"
	intentFailedRecoveryPath = "intent_failed_recovery.md"
)

//go:embed cinna_persona.md
var cinnaPersonaPrompt string

//go:embed intent_classification.md
var intentClassificationPrompt string

//go:embed intent_failed_recovery.md
var intentFailedRecoveryPrompt string

type Prompts struct {
	CinnaPersona         string
	IntentClassification string
	IntentFailedRecovery string
	logger               *slog.Logger
}

func LoadPromptFiles(logger *slog.Logger) *Prompts {
	prompts := &Prompts{logger: logger}
	prompts.CinnaPersona = prompts.loadEmbeddedPrompt(cinnaPersonaPath, cinnaPersonaPrompt)
	prompts.IntentClassification = prompts.loadEmbeddedPrompt(intentClassificationPath, intentClassificationPrompt)
	prompts.IntentFailedRecovery = prompts.loadEmbeddedPrompt(intentFailedRecoveryPath, intentFailedRecoveryPrompt)

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
