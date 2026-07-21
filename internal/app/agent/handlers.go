package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

func (a *CinnaReactAgent) HandleText(
	ctx context.Context,
	userID int64,
	text string,
) (string, error) {
	logger := a.logger.With(
		"path",
		"internal/app/agent/cinna_graph/HandleText",
	)
	// concat in-memory message history and the most recent user message
	userMessage := schema.UserMessage(text)
	messages := a.store.Get(ctx, userID)
	messages = append(messages, userMessage)
	input := &TaskInput{
		telegramUserID: userID,
		chatHistory:    messages,
	}

	// context isolation for KV Cache optimization
	chatModelOptions := GetDeepseekIsolationOptions([]string{
		intentLLMNodeName,
		shoppingTasksPlannerLLMNodeName,
		feedbacksUpdatePlannerLLMNodeName,
		cinnaChatNodeName,
	}, userID)

	// chatModelMonitors := MonitorLLMInputMessages([]string{cinnaChatNodeName})
	// chatModelOptions = append(chatModelOptions, chatModelMonitors...)

	result, err := a.runner.Invoke(ctx, input, chatModelOptions...)
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	a.store.UpdateChatHistory(ctx, userID, result.chatHistory)
	return result.outputMessage.Content, nil
}
