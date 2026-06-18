package prompt

import (
	"log/slog"
	"os"
	"strings"
)

const (
	cinnaPersonaPath         = "./cinna_persona.md"
	intentClassificationPath = "./intent_classification.md"
)

type Prompts struct {
	CinnaPersona         string
	IntentClassification string
	logger               *slog.Logger
}

func LoadPromptFiles(logger *slog.Logger) *Prompts {
	prompts := &Prompts{logger: logger}
	cinnaPersonaPrompt := prompts.loadFile(cinnaPersonaPath)
	intentClassificationPrompt := prompts.loadFile(intentClassificationPath)
	prompts.CinnaPersona = cinnaPersonaPrompt
	prompts.IntentClassification = intentClassificationPrompt
	/*
		maybe add fallback prompts here
	*/
	return prompts
}

// load prompt from a single file
func (p *Prompts) loadFile(filepath string) string {
	logger := p.logger.With("path", "internal/app/agent/prompt/prompts/LoadFile")
	content, err := os.ReadFile(filepath)
	if err != nil {
		logger.Error("failed to load prompt file", "file", filepath, "error", err)
		return ""
	}
	trimmedPrompt := strings.TrimSpace(string(content))
	if trimmedPrompt == "" {
		logger.Error("failed to prompt file is empty", "file", filepath)
	}
	return trimmedPrompt
}
