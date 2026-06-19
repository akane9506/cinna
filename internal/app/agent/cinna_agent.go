package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent/prompt"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type HistoryMessage struct {
	History       []*schema.Message
	SystemMessage string
	UserMessage   string
}

type CinnaReactAgent struct {
	chatModel *deepseek.ChatModel
	graph     *compose.Graph[[]*schema.Message, *schema.Message]
	prompts   *prompt.Prompts
	runner    compose.Runnable[[]*schema.Message, *schema.Message]
	logger    *slog.Logger
}

const (
	cinnaChatNodeName = "cinna_response"
)

func NewCinnaReactAgent(
	ctx context.Context,
	config *app.Config,
	logger *slog.Logger) (*CinnaReactAgent, error) {

	apiKey := &APIKey{
		Deepseek: config.DeepseekAPIKey,
	}
	// create LLM models
	models := NewLLMModels(apiKey, logger)
	// load prompts
	prompts := prompt.LoadPromptFiles(logger)
	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
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
	// create the agent
	agent := &CinnaReactAgent{
		chatModel: chatModel,
		graph:     graph,
		prompts:   prompts,
		logger:    logger,
	}
	// compile graph and create a runner
	runner, err := agent.CreateRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna agent runner: %w", err)
	}
	agent.runner = runner
	return agent, nil
}

// ========= Handler =========
func (a *CinnaReactAgent) HandleText(
	ctx context.Context,
	userID int64,
	text string,
) (string, error) {
	logger := a.logger.With(
		"path",
		"internal/app/agent/cinna_graph/HandleText",
	)
	// will need to handle the chat history here
	userMessage := schema.UserMessage(text)
	result, err := a.runner.Invoke(ctx, []*schema.Message{userMessage})
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	return result.Content, nil
}

// ========= Build Graph ==========
func (a *CinnaReactAgent) CreateRunner(
	ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	graph := a.graph
	a.AddCinnaResponseNode()
	graph.AddEdge(compose.START, cinnaChatNodeName)
	graph.AddEdge(cinnaChatNodeName, compose.END)
	runnable, err := graph.Compile(ctx)
	if err != nil {
		a.logger.Error("failed to run manual cinna chat graph", "error", err)
		return nil, err
	}
	return runnable, nil
}

// ========= Components ==========
// Cinna Chat Model
// in: []*schema.Message | out: *schema.Message
func (a *CinnaReactAgent) AddCinnaResponseNode() {
	a.graph.AddChatModelNode(cinnaChatNodeName, a.chatModel,
		compose.WithStatePreHandler(
			func(
				ctx context.Context,
				input []*schema.Message,
				state *HistoryMessage) ([]*schema.Message, error) {
				// inject cinna persona
				msgs := []*schema.Message{&schema.Message{Role: schema.System, Content: a.prompts.CinnaPersona}}
				state.SystemMessage = a.prompts.CinnaPersona
				// include Tool and Assistant chat history
				for _, msg := range input {
					if msg.Role == schema.User {
						state.UserMessage = msg.Content
					} else if msg.Role == schema.System {
						continue // we just ignore system message here
					} else {
						msgs = append(msgs, msg)
					}
				}
				// add always add user's message at the end
				msgs = append(msgs, &schema.Message{Role: schema.User, Content: state.UserMessage})
				return msgs, nil
			}),
		compose.WithStatePostHandler(
			func(
				ctx context.Context,
				output *schema.Message,
				state *HistoryMessage) (*schema.Message, error) {
				state.History = append(state.History, output)
				return output, nil
			}),
	)
}

// ========== Manual Tests [DO NOT USE IN PRODUCTION] ==========
func (a *CinnaReactAgent) buildCinnaChatGraph(
	ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	if a.runner != nil {
		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
	}
	graph := a.graph
	a.AddCinnaResponseNode()
	graph.AddEdge(compose.START, cinnaChatNodeName)
	graph.AddEdge(cinnaChatNodeName, compose.END)
	runnable, err := graph.Compile(ctx)
	if err != nil {
		a.logger.Error("failed to run manual cinna chat graph", "error", err)
		return nil, err
	}
	return runnable, nil
}
