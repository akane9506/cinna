package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/akane9506/cinna/internal/app"
	"github.com/akane9506/cinna/internal/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

const mockTelegramID = 123456

func setupModels() *Models {
	config, _ := app.LoadConfig()
	apiKey := &APIKey{Deepseek: config.DeepseekAPIKey}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	models := NewLLMModels(apiKey, logger)
	return models
}

func TestFailedModelCreations(t *testing.T) {
	ctx := context.Background()
	apiKey := &APIKey{Deepseek: ""}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	models := NewLLMModels(apiKey, logger)
	_, err := models.CreateDeepseekFlashModel(ctx)
	assert.Error(t, err)
	_, err = models.CreateDeepseekFlashJSONModel(ctx)
	assert.Error(t, err)
}

func TestGetDeepseekIsolationOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []*compose.NodePath
	}{
		{
			name: "with node names",
			input: []*compose.NodePath{
				compose.NewNodePath("node_a"),
				compose.NewNodePath("node_b", "node_c")},
		},
		{
			name:  "empty slice",
			input: []*compose.NodePath{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := GetDeepseekIsolationOptions(tt.input, mockTelegramID)
			assert.Equal(t, len(tt.input), len(options))
		})
	}
}

func TestMonitorLLMInputMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []*compose.NodePath
	}{
		{
			name: "with node names",
			input: []*compose.NodePath{
				compose.NewNodePath("node_a"),
				compose.NewNodePath("node_a", "node_b")},
		},
		{
			name:  "empty slice",
			input: []*compose.NodePath{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := MonitorLLMInputMessagesWithPath(tt.input)
			assert.Equal(t, len(tt.input), len(options))
		})
	}
}

// Manual Tests

func TestDeepSeekFlashChatModelManual(t *testing.T) {
	utils.EnforceManualTest(t)
	ctx := context.Background()
	models := setupModels()
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

func TestDeepSeekFlashJSONModelManual(t *testing.T) {
	utils.EnforceManualTest(t)
	ctx := context.Background()
	models := setupModels()
	JSONModel, err := models.CreateDeepseekFlashJSONModel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := JSONModel.Generate(ctx, []*schema.Message{
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
