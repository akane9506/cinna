package core

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func setupMockGraph(ctx context.Context) (compose.Runnable[Intent, string], error) {
	graph := compose.NewGraph[Intent, string](
		compose.WithGenLocalState(
			func(ctx context.Context) *AgentState {
				return &AgentState{}
			}),
	)

	addInputLambdaNode := func(nodeName string) {
		graph.AddLambdaNode(
			nodeName,
			compose.InvokableLambda[Intent, []*schema.Message](
				func(ctx context.Context, intent Intent) ([]*schema.Message, error) {
					compose.ProcessState[*AgentState](
						ctx,
						func(ctx context.Context, state *AgentState) error {
							state.ChatIntent = intent
							return nil
						},
					)
					return []*schema.Message{}, nil
				},
			),
		)
	}
	addResultNode := func(nodeName string) {
		graph.AddLambdaNode(
			nodeName,
			compose.InvokableLambda[[]*schema.Message, string](
				func(ctx context.Context, msgs []*schema.Message) (string, error) {
					return nodeName, nil
				},
			),
		)
	}
	inputLambdaName := "input_lambda"
	addInputLambdaNode(inputLambdaName)
	addResultNode(listShoppingItemsLambdaNodeName)
	addResultNode(listFeedbackItemsLambdaNodeName)
	addResultNode(preChatPassthroughNodeName)

	graph.AddEdge(compose.START, inputLambdaName)
	graph.AddBranch(
		inputLambdaName,
		compose.NewGraphBranch(
			intentBranch,
			map[string]bool{
				listShoppingItemsLambdaNodeName: true,
				listFeedbackItemsLambdaNodeName: true,
				preChatPassthroughNodeName:      true,
			},
		),
	)
	graph.AddEdge(listFeedbackItemsLambdaNodeName, compose.END)
	graph.AddEdge(listShoppingItemsLambdaNodeName, compose.END)
	graph.AddEdge(preChatPassthroughNodeName, compose.END)
	runnable, err := graph.Compile(ctx)
	return runnable, err
}

func TestIntentBranch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	runnable, err := setupMockGraph(ctx)
	assert.NoError(t, err)
	tests := []struct {
		name           string
		intent         Intent
		expectedBranch string
	}{
		{
			name:           string(IntentShopping),
			intent:         IntentShopping,
			expectedBranch: listShoppingItemsLambdaNodeName,
		},
		{
			name:           string(IntentFeedback),
			intent:         IntentFeedback,
			expectedBranch: listFeedbackItemsLambdaNodeName,
		},
		{
			name:           string(IntentOther),
			intent:         IntentOther,
			expectedBranch: preChatPassthroughNodeName,
		},
		{
			name:           "invalid intent",
			intent:         Intent("invalid"),
			expectedBranch: preChatPassthroughNodeName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := runnable.Invoke(ctx, tt.intent)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBranch, output)
		})
	}
}
