package agent

import (
	"context"
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

func NewCinnaReactAgent(ctx context.Context, config *app.Config, logger *slog.Logger) {
	apiKey := &APIKey{
		Deepseek: config.DeepseekAPIKey,
	}
	models := NewLLMModels(apiKey, logger)
	chatModel := models.CreateDeepseekFlashModel(ctx)

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
}
