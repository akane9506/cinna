package prompt

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadPrompts(t *testing.T) {
	logger := slog.Default()
	prompts := LoadPromptFiles(logger)
	assert.NotZero(t, len(prompts.CinnaPersona))
	assert.Contains(t, prompts.CinnaPersona, "Cinna")
	assert.NotZero(t, len(prompts.IntentClassification))
	assert.Contains(t, prompts.IntentClassification, "SHOPPING")
	assert.NotZero(t, len(prompts.IntentFailedRecovery))
	assert.Contains(t, prompts.IntentFailedRecovery, "意图分类失败")
}
