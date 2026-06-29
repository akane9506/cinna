package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

type TaskInput struct {
	telegramUserID int64
	chatHistory    []*schema.Message
}

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

	result, err := a.runner.Invoke(ctx, input)
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	a.store.Append(ctx, userID, userMessage)
	a.store.Append(ctx, userID, result)
	return result.Content, nil
}
