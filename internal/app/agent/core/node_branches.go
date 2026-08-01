package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const preChatPassthroughNodeName = "pre_chat_passthrough"

var branchExitNodeName string = preChatPassthroughNodeName

func (a *AgentCore) AddIntentBranch(exitNodeName string) {
	branchExitNodeName = exitNodeName
	endNodes := map[string]bool{}
	endNodes[listShoppingItemsLambdaNodeName] = true
	endNodes[listFeedbackItemsLambdaNodeName] = true
	endNodes[branchExitNodeName] = true
	a.graph.AddBranch(intentLambdaNodeName, compose.NewGraphBranch(intentBranch, endNodes))
}

func (a *AgentCore) RegisterPreChatPassthroughNode() {
	a.graph.AddPassthroughNode(preChatPassthroughNodeName)
}

func (a *AgentCore) AddReplyCompressionBranch() {
	endNodes := map[string]bool{}
	endNodes[replyOnlyChainName] = true
	endNodes[replyCompressionChainName] = true
	a.graph.AddBranch(preChatPassthroughNodeName, compose.NewGraphBranch(replyCompressionBranch, endNodes))
}

func intentBranch(ctx context.Context, in []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			switch state.ChatIntent {
			case IntentShopping:
				next = listShoppingItemsLambdaNodeName
			case IntentFeedback:
				next = listFeedbackItemsLambdaNodeName
			default:
				next = branchExitNodeName
			}
			return nil
		})
	return next, nil
}

func replyCompressionBranch(ctx context.Context, in []*schema.Message) (string, error) {
	if len(in) > COMPRESSION_THRESHOLD {
		return replyCompressionChainName, nil
	}
	return replyOnlyChainName, nil
}
