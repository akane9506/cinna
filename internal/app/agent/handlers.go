package agent

import (
	"context"

	"github.com/akane9506/cinna/internal/app/agent/core"
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
	// lock user request to avoid concurrent message collapse history
	unlockRequest := a.store.LockUserRequest(userID)
	defer unlockRequest()

	// concat in-memory message history and the most recent user message
	userMessage := schema.UserMessage(text)
	messages := a.store.Get(ctx, userID)
	messages = append(messages, userMessage)
	input := &core.GraphInput{
		TelegramUserID: userID,
		ChatHistory:    messages,
	}

	runtimeOptions := a.core.GetGraphRuntimeOptions(userID)
	result, err := a.runner.Invoke(ctx, input, runtimeOptions...)
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	a.store.UpdateChatHistory(ctx, userID, result.ChatHistory)
	return result.OutputMessage.Content, nil
}
