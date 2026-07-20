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
	a.store.Append(ctx, userID, userMessage)

	chatModelOptions := GetDeepseekIsolationOptions([]string{
		intentLLMNodeName,
		shoppingTasksPlannerLLMNodeName,
		feedbacksUpdatePlannerLLMNodeName,
		cinnaChatNodeName,
	}, userID)

	// PRESERVE these for future use
	// tempCallbackOnStart := compose.WithCallbacks(callbacks.NewHandlerBuilder().
	// 	OnStartFn(func(ctx context.Context, runInfo *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	// 		modelInput := model.ConvCallbackInput(input)
	// 		organizedStrings := []string{}
	// 		for _, message := range modelInput.Messages {
	// 			organizedStrings = append(organizedStrings, fmt.Sprintf("role: %s, content: %s", message.Role, message.Content[:10]))
	// 		}
	// 		logger.Info(
	// 			"Chat model info - ",
	// 			"Node name",
	// 			runInfo.Component,
	// 			"Contents",
	// 			organizedStrings)
	// 		return ctx
	// 	}).Build()).DesignateNode(intentLLMNodeName)

	// tempCallback := compose.WithCallbacks(callbacks.NewHandlerBuilder().
	// 	OnEndFn(func(ctx context.Context, runInfo *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	// 		modelOutput := model.ConvCallbackOutput(output)
	// 		logger.Info(
	// 			"Chat model info - ",
	// 			"Node name",
	// 			runInfo.Component,
	// 			"Cached Tokens",
	// 			modelOutput.Message.ResponseMeta.Usage.PromptTokenDetails.CachedTokens)
	// 		return ctx
	// 	}).Build()).DesignateNode(intentLLMNodeName)

	result, err := a.runner.Invoke(ctx, input, chatModelOptions...)
	if err != nil {
		logger.Error("failed to get cinna response", "user_id", userID, "error", err)
		return "", err
	}
	a.store.Append(ctx, userID, result)
	return result.Content, nil
}
