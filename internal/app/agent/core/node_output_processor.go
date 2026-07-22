// Output processor
// Organize cinna chat message, chat history for the graph output

package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const processOutputLambdaNodeName = "process_output"

func (a *AgentCore) RegisterOutputProcessor() {
	addProcessOutputLambdaNode(a.graph)
}

// ProcessOutputLambdaNode
// in: *schema.Message | out: *TaskOutput
func addProcessOutputLambdaNode(graph Graph) {
	graph.AddLambdaNode(
		processOutputLambdaNodeName,
		compose.InvokableLambda[*schema.Message, *GraphOutput](processTaskOutput),
	)
}

func processTaskOutput(
	ctx context.Context, input *schema.Message) (*GraphOutput, error) {
	output := &GraphOutput{}
	output.OutputMessage = input
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			history := cleanupOutputMessage(state.History)
			output.ChatHistory = make([]*schema.Message, 0, len(history)+1)
			output.ChatHistory = append(output.ChatHistory, history...)
			output.ChatHistory = append(output.ChatHistory, input)
			return nil
		})
	return output, nil
}
