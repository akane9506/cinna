package agent

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ========= Build Graph ==========
func (a *CinnaReactAgent) BuildGraph(
	ctx context.Context) (compose.Runnable[*TaskInput, *schema.Message], error) {
	graph := a.graph
	a.AddProcessInputLambdaNode()
	a.AddIntentClassificationNode()
	a.AddIntentOutputLambdaNode()
	a.AddListShoppingItemsLambda()
	a.AddCinnaResponseNode()
	graph.AddEdge(compose.START, processInputLambdaNodeName)
	graph.AddEdge(processInputLambdaNodeName, intentLLMNodeName)
	graph.AddEdge(intentLLMNodeName, intentLambdaNodeName)
	a.AddIntentBranch()
	graph.AddEdge(listShoppingItemsLambdaNodeName, cinnaChatNodeName)
	graph.AddEdge(cinnaChatNodeName, compose.END)
	runnable, err := graph.Compile(ctx)
	if err != nil {
		a.logger.Error("failed to run manual cinna chat graph", "error", err)
		return nil, err
	}
	return runnable, nil
}
