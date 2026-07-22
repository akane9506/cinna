// Input Processor
// This processor prepares input data, including user id, history messages for the agent's state

package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	processInputLambdaNodeName = "process_input"
)

type Intention struct {
	Intent string `json:"intent"`
	Action string `json:"action"`
}

func (a *AgentCore) RegisterInputProcessor() {
	addProcessInputLambdaNode(a.graph)
}

// ProcessInputLambdaNode
// in: *GraphInput | out: []*schema.Message
func addProcessInputLambdaNode(graph Graph) {
	graph.AddLambdaNode(
		processInputLambdaNodeName,
		compose.InvokableLambda[*GraphInput, []*schema.Message](processTaskInput),
	)
}

func processTaskInput(
	ctx context.Context, input *GraphInput) ([]*schema.Message, error) {
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			state.TelegramUserID = input.TelegramUserID
			return nil
		},
	)
	return input.ChatHistory, nil
}
