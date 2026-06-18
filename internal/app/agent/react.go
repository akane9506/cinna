package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type HistoryMessage struct {
	History       []*schema.Message
	SystemMessage string
	UserMessage   string
}

func NewCinnaReactAgent(ctx context.Context, config *app.Config, logger *slog.Logger) error {
	apiKey := &APIKey{
		Deepseek: config.DeepseekAPIKey,
	}
	// create LLM models
	models := NewLLMModels(apiKey, logger)
	// load prompts
	// prompts := prompt.LoadPromptFiles(logger)

	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		return fmt.Errorf("failed to create cinna react agent: %w", err)
	}

	// build graph with local chat history state
	graph := compose.NewGraph[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) (state *HistoryMessage) {
			return &HistoryMessage{
				History:       make([]*schema.Message, 0, 4),
				UserMessage:   "",
				SystemMessage: "",
			}
		}),
	)

	// add nodes ---- still in progress ----
	graph.AddChatModelNode("cinna_response", chatModel,
		compose.WithStatePreHandler(
			func(
				ctx context.Context,
				input []*schema.Message,
				state *HistoryMessage) ([]*schema.Message, error) {
				return nil, nil
			},
		),
	)

	return nil
}
