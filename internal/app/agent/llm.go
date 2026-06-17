package agent

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/callbacks"
)

var (
	deepseekFlashModel *deepseek.ChatModel
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

func (m *Models) CreateDeepseekFlashModel(ctx context.Context) *deepseek.ChatModel {
	deepseekFlashOnce.Do(func() {
		var err error
		deepseekFlashModel, err = deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
			APIKey:  m.apiKey.Deepseek,
			BaseURL: deepseekBaseURL,
			Model:   deepseekFlashModelName,
		})
		if err != nil {
			m.logger.Error("failed to start deepseek flash model", "error", err)
			os.Exit(1)
		}
	})
	return deepseekFlashModel
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
