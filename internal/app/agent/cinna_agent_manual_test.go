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

func TestIntentClassification(t *testing.T) {
	utils.EnforceManualTest(t)
	ctx := context.Background()
	config, err := app.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	agent, err := initializeBaseAgent(ctx, config, slog.Default())
	runnable, err := agent.buildIntentJSONGraph(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// single message
	input := []*schema.Message{
		&schema.Message{Role: schema.User, Content: "可以帮我把五花肉和铅笔加到购物车吗?"},
	}
	result, err := runnable.Invoke(ctx, input)
	t.Log("single SHOPPING intent: ", result.Content)

	// chat history
	input = []*schema.Message{
		&schema.Message{Role: schema.Assistant, Content: "做蛋糕需要用到面粉和糖?"},
		&schema.Message{Role: schema.User, Content: "可以帮我把这些记下吗？"},
	}
	result, err = runnable.Invoke(ctx, input)
	t.Log("chat history with SHOPPING intent: ", result.Content)

	// other
	input = []*schema.Message{
		&schema.Message{Role: schema.User, Content: "拍摄模型车需要用到什么设备呀？"},
	}
	result, err = runnable.Invoke(ctx, input)
	t.Log("single OTHER intent: ", result.Content)
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
