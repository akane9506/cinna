package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/callbacks"
)

var (
	deepseekFlashModel *deepseek.ChatModel
	deepseekFlashError error
	deepseekFlashOnce  sync.Once
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
			deepseekFlashError = fmt.Errorf("DEEPSEEK_API_KEY is required")
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

// Callbacks, used for debugging purpose
func (m *Models) GetStartCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			input callbacks.CallbackInput) context.Context {
			m.logger.Info("[INPUT]", "name", info.Name, "type", info.Type, "component", info.Component, "input", input)
			return ctx
		}).Build()
}

func (m *Models) GetEndCallback() callbacks.Handler {
	return callbacks.NewHandlerBuilder().
		OnStartFn(func(
			ctx context.Context,
			info *callbacks.RunInfo,
			input callbacks.CallbackInput) context.Context {
			m.logger.Info("[OUTPUT]", "name", info.Name, "type", info.Type, "component", info.Component, "input", input)
			return ctx
		}).Build()
}
