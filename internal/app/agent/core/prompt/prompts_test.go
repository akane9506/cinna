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
	assert.NotZero(t, len(prompts.FeedbacksPlanner))
	assert.Contains(t, prompts.FeedbacksPlanner, "反馈")
	assert.NotZero(t, prompts.ShoppingListPlanner)
	assert.Contains(t, prompts.ShoppingListPlanner, "规划")
	assert.NotZero(t, len(prompts.MemoryCompression))
	assert.Contains(t, prompts.MemoryCompression, "摘要")
}
