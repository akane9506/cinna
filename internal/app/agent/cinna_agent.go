package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent/memory"
	"github.com/akane9506/cinna/internal/app/agent/prompt"
	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type CinnaReactAgent struct {
	chatModel *deepseek.ChatModel
	jsonModel *deepseek.ChatModel
	graph     *compose.Graph[*TaskInput, *schema.Message]
	prompts   *prompt.Prompts
	runner    compose.Runnable[*TaskInput, *schema.Message]
	store     *memory.InMemoryStore
	repos     *ports.Repositories
	logger    *slog.Logger
}

// Create a new agent with graph and runner built
func NewCinnaReactAgent(
	ctx context.Context,
	config *app.Config,
	repos *ports.Repositories,
	logger *slog.Logger) (*CinnaReactAgent, error) {
	agent, err := initializeBaseAgent(ctx, config, logger)
	agent.repos = repos
	if err != nil {
		return nil, err
	}
	// compile graph and create a runner
	runner, err := agent.BuildGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna agent runner: %w", err)
	}
	agent.runner = runner
	return agent, nil
}

// initialize a raw agent
func initializeBaseAgent(
	ctx context.Context,
	config *app.Config,
	logger *slog.Logger,
) (*CinnaReactAgent, error) {
	apiKey := &APIKey{
		Deepseek: config.DeepseekAPIKey,
	}
	// load prompts
	prompts := prompt.LoadPromptFiles(logger)
	// create LLM models
	models := NewLLMModels(apiKey, logger)
	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
	}
	jsonModel, err := models.CreateDeepseekFlashJSONModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
	}
	// initialize in-memory store (state)
	store := memory.NewImMemoryStore()
	// initialize graph with local chat history state
	graph := compose.NewGraph[*TaskInput, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) (state *CinnaAgentState) {
			return &CinnaAgentState{
				History: make([]*schema.Message, 0, 4),
			}
		}),
	)
	// create the agent
	agent := &CinnaReactAgent{
		chatModel: chatModel,
		jsonModel: jsonModel,
		graph:     graph,
		prompts:   prompts,
		store:     store,
		logger:    logger,
	}
	return agent, nil
}
