package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ========= Build Graph ==========
func (a *CinnaReactAgent) BuildGraph(
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
		a.logger.Error("failed to compile manual cinna chat graph", "error", err)
		return nil, err
	}
	return runnable, nil
}

func (a *CinnaReactAgent) buildIntentJSONGraph(
	ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	if a.runner != nil {
		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
	}
	graph := a.graph
	a.AddIntentClassificationNodes()
	graph.AddEdge(compose.START, intentLLMNodeName)
	graph.AddEdge(intentLLMNodeName, compose.END)
	runnable, err := graph.Compile(ctx)
	if err != nil {
		a.logger.Error("failed to compile manual intent classification graph", "error", err)
		return nil, err
	}
	return runnable, nil
}
