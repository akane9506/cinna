package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
)

var (
	deepseekFlashModel *deepseek.ChatModel
	deepseekFlashError error
	deepseekFlashOnce  sync.Once

	deepseekFlashJSONModel *deepseek.ChatModel
	deepseekFlashJSONError error
	deepseekFlashJSONOnce  sync.Once
)

const (
	deepseekBaseURL        = "https://api.deepseek.com"
	deepseekFlashModelName = "deepseek-v4-flash"
)

type APIKey struct {
	Deepseek string
}

type Models struct {
	logger *slog.Logger
	apiKey *APIKey
}

func NewLLMModels(apiKey *APIKey, logger *slog.Logger) *Models {
	return &Models{
		logger: logger,
		apiKey: apiKey,
	}
}

func (m *Models) CreateDeepseekFlashModel(ctx context.Context) (*deepseek.ChatModel, error) {
	deepseekFlashOnce.Do(func() {
		if m.apiKey.Deepseek == "" {
			deepseekFlashError = fmt.Errorf("DEEPSEEK_API_KEY is required for flash model")
			return
		}
		var err error
		deepseekFlashModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  m.apiKey.Deepseek,
			BaseURL: deepseekBaseURL,
			Model:   deepseekFlashModelName,
		})
		if err != nil {
			deepseekFlashError = fmt.Errorf("failed to start deepseek flash model: %w", err)
			return
		}
	})
	return deepseekFlashModel, deepseekFlashError
}

// This model is for the structured JSON output
// Include the word "json" in the system or user prompt,
// and provide an example of the desired JSON format to guide the model in outputting valid JSON.
// https://api-docs.deepseek.com/guides/json_mode
func (m *Models) CreateDeepseekFlashJSONModel(ctx context.Context) (*deepseek.ChatModel, error) {
	deepseekFlashJSONOnce.Do(func() {
		if m.apiKey.Deepseek == "" {
			deepseekFlashJSONError = fmt.Errorf("DEEPSEEK_API_KEY is required for flash JSON model")
			return
		}
		var err error
		deepseekFlashJSONModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:             m.apiKey.Deepseek,
			BaseURL:            deepseekBaseURL,
			Model:              deepseekFlashModelName,
			ResponseFormatType: deepseek.ResponseFormatTypeJSONObject,
		})
		if err != nil {
			deepseekFlashJSONError = fmt.Errorf("failed to start deepseek flash JSON model: %w", err)
			return
		}
	})
	return deepseekFlashJSONModel, deepseekFlashJSONError
}
