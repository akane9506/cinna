package prompt

import (
	"log/slog"
	"testing"

	"github.com/akane9506/cinna/internal/utils"
)

func TestLoadPrompts(t *testing.T) {
	utils.EnforceManualTest(t)
	logger := slog.Default()
	prompts := LoadPromptFiles(logger)
	t.Log("Cinna Persona Prompt: ", prompts.CinnaPersona)
	t.Log("Classification Prompt: ", prompts.IntentClassification)
}
