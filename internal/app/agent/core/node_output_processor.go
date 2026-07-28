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
// in: *schema.Message | out: *GraphOutput
func addProcessOutputLambdaNode(graph Graph) {
	graph.AddLambdaNode(
		processOutputLambdaNodeName,
		compose.InvokableLambda[*ReplyCompressionOutput, *GraphOutput](processTaskOutput),
	)
}

func processTaskOutput(
	ctx context.Context, input *ReplyCompressionOutput) (*GraphOutput, error) {
	output := &GraphOutput{}
	output.OutputMessage = input.Reply
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			history := cleanupOutputMessage(state.History)
			output.ChatHistory = make([]*schema.Message, 0, len(history)+1)
			output.ChatHistory = append(output.ChatHistory, history...)
			output.ChatHistory = append(output.ChatHistory, input.Reply)
			return nil
		})
	if input.Compression {
		output.Compression = true
		output.Memory = input.Memory
	}
	return output, nil
}
