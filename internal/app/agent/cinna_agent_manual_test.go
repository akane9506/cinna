package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func TestDeepseekFlashModelManual(t *testing.T) {
	models, ctx := setupDeepseekModelTesting(t)
	chatModel, err := models.CreateDeepseekFlashModel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := chatModel.Generate(ctx, []*schema.Message{
		schema.UserMessage("Say hello in 3 different languages"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.Content)
}

func TestDeepseekFlashJSONModelManual(t *testing.T) {
	models, ctx := setupDeepseekModelTesting(t)
	jsonModel, err := models.CreateDeepseekFlashJSONModel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := jsonModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(`
The user will provide some exam text. please answer the question.

EXAMPLE INPUT:
Which is the highest mountain in the world? Mount Everest.

EXAMPLE JSON OUTPUT:
{
    "question": "Which is the highest mountain in the world?",
    "answer": "Mount Everest"
}`),
		schema.UserMessage("What is a good way to create a graph agent?"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.Role, resp.Content)
	var parsedResponse struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}
	if err := json.Unmarshal([]byte(resp.Content), &parsedResponse); err != nil {
		t.Fatal(err)
	}
	t.Log(fmt.Sprintf("%+v", parsedResponse))
}

func TestCinnaChat(t *testing.T) {
	utils.EnforceManualTest(t)
	ctx := context.Background()
	config, err := app.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	agent, err := initializeBaseAgent(ctx, config, slog.Default())
	runnable, err := agent.buildCinnaChatGraph(ctx)
	if err != nil {
		t.Fatal(err)
	}
	input := []*schema.Message{
		&schema.Message{Role: schema.User, Content: "你好呀Cinna"},
	}
	result, err := runnable.Invoke(ctx, input)
	t.Log(result.Content)
	t.Log("completion tokens: ",
		result.ResponseMeta.Usage.CompletionTokens,
		"cache miss tokens: ",
		result.ResponseMeta.Usage.PromptTokens-result.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
		"cache hit tokens: ",
		result.ResponseMeta.Usage.PromptTokenDetails.CachedTokens,
	)
}

func TestIntentLambda(t *testing.T) {
	utils.EnforceManualTest(t)
	ctx := context.Background()
	config, err := app.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	agent, err := initializeBaseAgent(ctx, config, slog.Default())
	runnable, err := agent.buildIntentLambdaOutputNode(ctx, t)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("======== Shopping intention lambda testing 1 ========")
	input := []*schema.Message{
		&schema.Message{Role: schema.Assistant, Content: "做蛋糕需要用到面粉和糖"},
		&schema.Message{Role: schema.User, Content: "可以帮我把这些记下吗？"},
	}
	_, err = runnable.Invoke(ctx, input)
	t.Log("======== Shopping intention lambda testing 2 ========")
	input = []*schema.Message{
		&schema.Message{Role: schema.User, Content: "可以看一下我都要买什么吗？"},
	}
	_, err = runnable.Invoke(ctx, input)
	t.Log("======== Other intention lambda testing ========")
	input = []*schema.Message{
		&schema.Message{Role: schema.User, Content: ""},
	}
	_, err = runnable.Invoke(ctx, input)
	t.Log("======== Other intention lambda testing ========")
	input = []*schema.Message{
		&schema.Message{Role: schema.User, Content: "拍摄模型车需要用到什么设备呀？"},
	}
	_, err = runnable.Invoke(ctx, input)
	t.Log("======== Feedback intention lambda testing ========")
	input = []*schema.Message{
		&schema.Message{Role: schema.User, Content: "Cinna目前的回复速度好慢呀？"},
	}
	_, err = runnable.Invoke(ctx, input)
}

func setupDeepseekModelTesting(t *testing.T) (*Models, context.Context) {
	utils.EnforceManualTest(t)
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Fatal("DEEPSEEK_API_KEY s required")
	}
	ctx := context.Background()
	models := NewLLMModels(&APIKey{
		Deepseek: apiKey,
	}, slog.Default())
	return models, ctx
}

// ========== Graph-Building For Manual Tests [DO NOT USE IN PRODUCTION] ==========
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

func (a *CinnaReactAgent) buildIntentLambdaOutputNode(
	ctx context.Context, t *testing.T) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
	if a.runner != nil {
		return nil, fmt.Errorf("this is only for manual test, do not use it in production")
	}
	// add a test-purposed node to make the test graph type compatible with the
	// agent's I/O type
	graph := a.graph
	a.AddIntentClassificationNodes()
	a.AddIntentOutputLambdaNode()
	graph.AddLambdaNode("mock_final_output",
		compose.InvokableLambda[[]*schema.Message, *schema.Message](func(
			ctx context.Context, input []*schema.Message) (*schema.Message, error) {
			compose.ProcessState[*HistoryMessage](ctx, func(
				ctx context.Context,
				state *HistoryMessage,
			) error {
				t.Log("Intent: ", state.ChatIntent)
				t.Log("Action: ", state.ActionType)
				t.Log("Messages: ", state.History)
				return nil
			})
			return input[len(input)-1], nil
		}),
	)
	graph.AddEdge(compose.START, intentLLMNodeName)
	graph.AddEdge(intentLLMNodeName, intentLambdaNodeName)
	graph.AddEdge(intentLambdaNodeName, "mock_final_output")
	graph.AddEdge("mock_final_output", compose.END)
	runnable, err := graph.Compile(ctx)
	if err != nil {
		a.logger.Error("failed to compile manual intent lambda graph", "error", err)
		return nil, err
	}
	return runnable, nil
}
