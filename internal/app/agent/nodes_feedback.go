package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	listFeedbackItemsLambdaNodeName = "list_feedback_items"
)

// ========== List Feedback Items Lambda ==========

func (a *CinnaReactAgent) AddListFeedbackItemsLambda() {
	a.graph.AddLambdaNode(
		listFeedbackItemsLambdaNodeName,
		compose.InvokableLambda[[]*schema.Message, []*schema.Message](a.processFeedbackListLambda),
	)
}

func (a *CinnaReactAgent) processFeedbackListLambda(
	ctx context.Context, msgs []*schema.Message) ([]*schema.Message, error) {
	compose.ProcessState[*CinnaAgentState](ctx,
		func(ctx context.Context, state *CinnaAgentState) error {
			feedbackItemsMessage := a.listFeedbackItems(ctx, state)
			msgs = append(msgs, feedbackItemsMessage)
			state.History = append(state.History, feedbackItemsMessage)
			return nil
		},
	)
	return msgs, nil
}

func (a *CinnaReactAgent) listFeedbackItems(
	ctx context.Context, state *CinnaAgentState) *schema.Message {
	logger := a.logger.With("Node", listFeedbackItemsLambdaNodeName)
	telegramUserID := state.TelegramUserID
	// query database
	currList, err := a.repos.Feedback.ListIncompleteFeedbacks(ctx)
	if err != nil {
		logger.Error(
			"failed to get pending feedback items",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "failed to get feedbacks from database"}
	}
	feedbackContents := []string{}
	for _, row := range currList {
		feedbackContents = append(feedbackContents, row.Content)
	}
	message := fmt.Sprintf(
		"feedbacks in the current database: [%s]",
		strings.Join(feedbackContents, ", "),
	)
	return &schema.Message{
		Role:    schema.Assistant,
		Content: message,
	}
}
