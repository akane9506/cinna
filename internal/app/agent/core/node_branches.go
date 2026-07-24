package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func (a *AgentCore) AddIntentBranch() {
	endNodes := map[string]bool{}
	endNodes[listShoppingItemsLambdaNodeName] = true
	endNodes[listFeedbackItemsLambdaNodeName] = true
	endNodes[replyCompressionChainName] = true
	a.graph.AddBranch(intentLambdaNodeName, compose.NewGraphBranch(intentBranch, endNodes))
}

func intentBranch(ctx context.Context, out []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			switch state.ChatIntent {
			case IntentShopping:
				next = listShoppingItemsLambdaNodeName
			case IntentFeedback:
				next = listFeedbackItemsLambdaNodeName
			default:
				next = replyCompressionChainName
			}
			return nil
		})
	return next, nil
}
