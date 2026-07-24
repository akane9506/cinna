package core

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/app/agent/core/prompt"
	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type Graph = *compose.Graph[*GraphInput, *GraphOutput]
type ChatModel = *deepseek.ChatModel
type JSONModel = *deepseek.ChatModel

type AgentCore struct {
	chatModel  ChatModel
	jsonModel  JSONModel
	graph      Graph
	prompts    *prompt.Prompts
	repos      *ports.Repositories
	logger     *slog.Logger
	runtimeEnv string
}

func InitializeAgentCore(
	ctx context.Context,
	config *app.Config,
	repos *ports.Repositories,
	logger *slog.Logger) (*AgentCore, error) {
	apiKey := &APIKey{
		Deepseek: config.DeepseekAPIKey,
	}
	// load prompts
	prompts := prompt.LoadPromptFiles(logger)
	// create LLM models
	models := NewLLMModels(apiKey, logger)
	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
	}
	jsonModel, err := models.CreateDeepseekFlashJSONModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cinna react agent: %w", err)
	}
	// initialize graph with local chat history state
	graph := compose.NewGraph[*GraphInput, *GraphOutput](
		compose.WithGenLocalState(func(ctx context.Context) (state *AgentState) {
			return &AgentState{
				History: make([]*schema.Message, 0, 4),
			}
		}),
	)
	agentCore := &AgentCore{
		chatModel:  chatModel,
		jsonModel:  jsonModel,
		repos:      repos,
		graph:      graph,
		prompts:    prompts,
		logger:     logger,
		runtimeEnv: config.RuntimeEnv,
	}
	return agentCore, nil
}

func (a *AgentCore) BuildGraph(
	ctx context.Context,
) (compose.Runnable[*GraphInput, *GraphOutput], error) {
	a.RegisterInputProcessor()
	a.RegisterIntentClassifier()
	a.RegisterShoppingListWorkflow()
	a.RegisterFeedbacksWorkflow()
	a.RegisterReplyCompressionChain()
	a.RegisterOutputProcessor()

	graph := a.graph
	graph.AddEdge(compose.START, processInputLambdaNodeName)
	graph.AddEdge(processInputLambdaNodeName, intentLLMNodeName)
	graph.AddEdge(intentLLMNodeName, intentLambdaNodeName)
	graph.AddEdge(shoppingTasksPlannerLLMNodeName, shoppingTaskRunLambdaNodeName)
	graph.AddEdge(shoppingTaskRunLambdaNodeName, replyCompressionChainName)
	graph.AddEdge(feedbacksUpdatePlannerLLMNodeName, updateFeedbackItemsLambdaNodeName)
	graph.AddEdge(updateFeedbackItemsLambdaNodeName, replyCompressionChainName)
	graph.AddEdge(replyCompressionChainName, processOutputLambdaNodeName)
	graph.AddEdge(processOutputLambdaNodeName, compose.END)
	a.AddIntentBranch()
	a.AddShoppingActionBranch()
	a.AddFeedbacksActionBranch()
	runnable, err := graph.Compile(ctx)

	if err != nil {
		a.logger.Error("failed to run manual cinna chat graph", "error", err)
		return nil, err
	}
	return runnable, nil
}

func (a *AgentCore) GetGraphRuntimeOptions(userID int64) []compose.Option {
	options := []compose.Option{}
	isolationOptions := GetDeepseekIsolationOptions([]*compose.NodePath{
		compose.NewNodePath(intentLLMNodeName),
		compose.NewNodePath(shoppingTasksPlannerLLMNodeName),
		compose.NewNodePath(feedbacksUpdatePlannerLLMNodeName),
		compose.NewNodePath(replyCompressionChainName, cinnaReplyNodeName),
	}, userID)
	options = append(options, isolationOptions...)
	if a.runtimeEnv == app.RuntimeDev {
		nodePaths := []*compose.NodePath{
			compose.NewNodePath(replyCompressionChainName, cinnaReplyNodeName),
		}
		monitorOptions := MonitorLLMInputMessagesWithPath(nodePaths)
		options = append(options, monitorOptions...)
	}
	return options
}
