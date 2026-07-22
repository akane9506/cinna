// Cinna Chat Node
// generates user-facing response

package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	cinnaChatNodeName = "cinna_response"
)

func (a *AgentCore) RegisterCinnaChatNode() {
	addCinnaChatNode(a.graph, a.chatModel, a.prompts.CinnaPersona)
}

// in: []*schema.Message | out: *[]schema.Message
func addCinnaChatNode(graph Graph, chatModel ChatModel, prompt string) {
	graph.AddChatModelNode(cinnaChatNodeName, chatModel,
		compose.WithStatePreHandler(prepareForChatNode(prompt)))
}

func prepareForChatNode(prompt string) compose.StatePreHandler[[]*schema.Message, *AgentState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *AgentState) ([]*schema.Message, error) {
		msgs := organizeInputMessage(input, prompt)
		state.History = msgs
		return msgs, nil
	}

}
