package prompt

import (
	_ "embed"
	"log/slog"
	"strings"
)

const (
	cinnaPersonaPath         = "cinna_persona.md"
	intentClassificationPath = "intent_classification.md"
	shoppingTaskPlannerPath  = "shopping_task_planner.md"
	feedbacksPlannerPath     = "feedbacks_planner.md"
)

//go:embed cinna_persona.md
var cinnaPersonaPrompt string

//go:embed intent_classification.md
var intentClassificationPrompt string

//go:embed shopping_task_planner.md
var shoppingTaskPlannerPrompt string

//go:embed feedbacks_planner.md
var feedbacksPlannerPrompt string

type Prompts struct {
	CinnaPersona         string
	IntentClassification string
	ShoppingListPlanner  string
	FeedbacksPlanner     string
	logger               *slog.Logger
}

func LoadPromptFiles(logger *slog.Logger) *Prompts {
	prompts := &Prompts{logger: logger}
	prompts.CinnaPersona = prompts.loadEmbeddedPrompt(cinnaPersonaPath, cinnaPersonaPrompt)
	prompts.IntentClassification = prompts.loadEmbeddedPrompt(intentClassificationPath, intentClassificationPrompt)
	prompts.ShoppingListPlanner = prompts.loadEmbeddedPrompt(shoppingTaskPlannerPath, shoppingTaskPlannerPrompt)
	prompts.FeedbacksPlanner = prompts.loadEmbeddedPrompt(feedbacksPlannerPath, feedbacksPlannerPrompt)
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
