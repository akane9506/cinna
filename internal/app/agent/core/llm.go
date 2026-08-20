package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/akane9506/cinna/internal/utils"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
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
	gptModelName           = "gpt-5.6-luna"
)

type APIKey struct {
	Deepseek string
	OpenAI   string
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

func (m *Models) CreateJSONModel(
	ctx context.Context,
	jsonSchema *openai.ChatCompletionResponseFormatJSONSchema) (*openai.ChatModel, error) {
	apiKey := m.apiKey.OpenAI
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required for JSON model")
	}
	jsonModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   gptModelName,
		Timeout: 1 * time.Minute,
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: jsonSchema,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create JSON model: %w", err)
	}
	return jsonModel, nil
}

// KV Cache isolation - deepseek
func GetDeepseekIsolationOptions(nodePaths []*compose.NodePath, telegramUserID int64) []compose.Option {
	opts := []compose.Option{}
	for _, path := range nodePaths {
		isolationID := fmt.Sprintf(
			"%d_node_%s",
			telegramUserID,
			strings.Join(path.GetPath(), "_"))
		opt := compose.WithChatModelOption(deepseek.WithExtraFields(map[string]interface{}{
			"user_id": isolationID,
		})).DesignateNodeWithPath(path)
		opts = append(opts, opt)
	}
	return opts
}

// KV Cache isolation - openai
func GetOpenAICacheOptions(nodePaths []*compose.NodePath, telegramUserID int64) []compose.Option {
	opts := []compose.Option{}
	for _, path := range nodePaths {
		isolationID := fmt.Sprintf(
			"%d_node_%s",
			telegramUserID,
			strings.Join(path.GetPath(), "_"))
		opt := compose.WithChatModelOption(openai.WithExtraFields(map[string]interface{}{
			"prompt_cache_key": isolationID,
		})).DesignateNodeWithPath(path)
		opts = append(opts, opt)
	}
	return opts
}

// Callbacks
func MonitorLLMInputMessagesWithPath(nodePaths []*compose.NodePath) []compose.Option {
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
	for _, path := range nodePaths {
		monitor := compose.WithCallbacks(logMsgsOnStartFn.Build()).DesignateNodeWithPath(path)
		monitors = append(monitors, monitor)
	}
	return monitors
}
