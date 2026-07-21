package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/akane9506/cinna/internal/utils"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
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

// KV Cache isolation
func GetDeepseekIsolationOptions(nodeNames []string, telegramUserID int64) []compose.Option {
	opts := []compose.Option{}
	for _, name := range nodeNames {
		isolationID := fmt.Sprintf("cinna:%d:node:%s", telegramUserID, name)
		opt := compose.WithChatModelOption(deepseek.WithExtraFields(map[string]interface{}{
			"user_id": isolationID,
		})).DesignateNode(name)
		opts = append(opts, opt)
	}
	return opts
}

// Callbacks
func MonitorLLMInputMessages(nodeNames []string) []compose.Option {
	monitors := []compose.Option{}
	logMsgsOnStartFn := callbacks.NewHandlerBuilder().OnStartFn(
		func(ctx context.Context,
			runInfo *callbacks.RunInfo,
			input callbacks.CallbackInput) context.Context {
			modelInput := model.ConvCallbackInput(input)
			organizedMsgs := []string{}
			for _, msg := range modelInput.Messages {
				content := fmt.Sprintf(
					"role: %s, content: %s",
					msg.Role, utils.TruncateString(msg.Content, 20))
				organizedMsgs = append(organizedMsgs, content)
			}
			fmt.Println("==========New Conversation==========")
			fmt.Printf("%s\n", strings.Join(organizedMsgs, "\n"))
			return ctx
		})
	for _, name := range nodeNames {
		monitor := compose.WithCallbacks(logMsgsOnStartFn.Build()).DesignateNode(name)
		monitors = append(monitors, monitor)
	}
	return monitors
}
