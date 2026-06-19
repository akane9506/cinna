package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent/memory"
	"github.com/akane9506/cinna/internal/app/agent/prompt"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// should find a better way to use it
type HistoryMessage struct {
	SystemMessage string
	History       []*schema.Message
	// maybe add token usage-related fields here
}

type CinnaReactAgent struct {
	chatModel *deepseek.ChatModel
	graph     *compose.Graph[[]*schema.Message, *schema.Message]
	prompts   *prompt.Prompts
	runner    compose.Runnable[[]*schema.Message, *schema.Message]
	store     *memory.InMemoryStore
	logger    *slog.Logger
}

const (
	cinnaChatNodeName = "cinna_response"
)

// Create a new agent with graph and runner built
func NewCinnaReactAgent(
	ctx context.Context,
	config *app.Config,
	logger *slog.Logger) (*CinnaReactAgent, error) {

	agent, err := initializeBaseAgent(ctx, config, logger)
	if err != nil {
		return nil, err
	}
	// compile graph and create a runner
	runner, err := agent.CreateRunner(ctx)
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
	// create LLM models
	models := NewLLMModels(apiKey, logger)
	// load prompts
	prompts := prompt.LoadPromptFiles(logger)
	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
	}
	// initialize store
	store := memory.NewImMemoryStore()
	// build graph with local chat history state
	graph := compose.NewGraph[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) (state *HistoryMessage) {
			return &HistoryMessage{
				History: make([]*schema.Message, 0, 4),
			}
		}),
	)
	// create the agent
	agent := &CinnaReactAgent{
		chatModel: chatModel,
		graph:     graph,
		prompts:   prompts,
		store:     store,
		logger:    logger,
	}
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
	// concat in-memory message history and the most recent user message
	userMessage := schema.UserMessage(text)
	messages := a.store.Get(userID)
	messages = append(messages, userMessage)

	result, err := a.runner.Invoke(ctx, messages)
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	a.store.Append(userID, userMessage)
	a.store.Append(userID, result)
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
					if msg.Role == schema.System {
						continue // we just ignore system message here
					} else {
						msgs = append(msgs, msg)
					}
				}
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

// ========== Helper functions ==========
func logMessages(title string, msgs []*schema.Message) {
	fmt.Println()
	fmt.Println("Start Logging", title)
	for _, msg := range msgs {
		fmt.Println(fmt.Sprintf("[%s] %s", msg.Role, msg.Content))
	}
	fmt.Println()
}
