package agent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func EnforceManualTest(t *testing.T) {
	if os.Getenv("RUN_MANUAL_TEST") != "1" {
		t.Skip("set RUN_MANUAL_TEST=1 to run manual LLM tests")
	}
}

func TestDeepseekFlashModelManual(t *testing.T) {
	EnforceManualTest(t)
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		t.Fatal("DEEPSEEK_API_KEY s required")
	}
	ctx := context.Background()
	models := NewLLMModels(&APIKey{
		Deepseek: apiKey,
	}, slog.Default())
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
