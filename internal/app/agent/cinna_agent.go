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
	return agent, nil
}

// ========= Build Graph ==========

// --------- Graphs for manual tests --------
func (a *CinnaReactAgent) buildCinnaChatGraph(
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

// Lambda

// Cinna Prompt Injection
// in: []*schema.Message | out: []*schema.Message
// func (a *CinnaReactAgent) AddCinnaPromptInjectionLambda() {
// 	a.graph.AddLambdaNode(
// 		"inject_cinna_prompt",
// 		injectSystemPrompt(a.prompts.CinnaPersona),
// 	)
// }

// Chat Model Nodes

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

// ========== Helper functions ==========

// // We've prepared different prompts for different ai models
// func injectSystemPrompt(prompt string) *compose.Lambda {
// 	return compose.InvokableLambda[[]*schema.Message, []*schema.Message](
// 		func(ctx context.Context,
// 			input []*schema.Message) (output []*schema.Message, err error) {
// 			msgs := make([]*schema.Message, 0, len(input)+1)
// 			msgs = append(msgs, schema.SystemMessage(prompt))
// 			// Keep all non-system message from input
// 			for _, msg := range input {
// 				if msg == nil || msg.Role == schema.System {
// 					continue
// 				}
// 				msgs = append(msgs, msg)
// 			}
// 			return msgs, nil
// 		},
// 	)
// }
