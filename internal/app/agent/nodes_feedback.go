package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	listFeedbackItemsLambdaNodeName   = "list_feedback_items"
	feedbacksUpdatePlannerNodeName    = "plan_feedback_updates"
	updateFeedbackItemsLambdaNodeName = "update_feedback_items"
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

// ========= Feedbacks Planner ==========
// Plan the task based the current feedbacks from the database and the user's message
// in: []*schema.Message | out: *schema.Message
func (a *CinnaReactAgent) AddFeedbacksPlanningNode() {
	a.graph.AddChatModelNode(
		feedbacksUpdatePlannerNodeName,
		a.jsonModel,
		compose.WithStatePreHandler(a.prepareForFeedbacksPlanning),
	)
}

func (a *CinnaReactAgent) prepareForFeedbacksPlanning(
	ctx context.Context,
	input []*schema.Message,
	state *CinnaAgentState,
) ([]*schema.Message, error) {
	state.SystemMessage = a.prompts.FeedbacksPlanner
	msgs := organizeInputMessage(input, state.SystemMessage)
	state.History = msgs
	return msgs, nil
}

// ========= Update Feedback Items Lambda ==========

type FeedbackItems struct {
	Items []string `json:"items"`
}

func (a *CinnaReactAgent) executeUpdateFeedbacks(
	ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
	logger := a.logger.With("Node", updateFeedbackItemsLambdaNodeName)
	newFeedbacks := a.parseFeedbackItems(ctx, msg.Content)
	var telegramUserID int64
	var msgs []*schema.Message
	compose.ProcessState[*CinnaAgentState](ctx,
		func(ctx context.Context, state *CinnaAgentState) error {
			telegramUserID = state.TelegramUserID
			msgs = state.History
			return nil
		})
	if len(newFeedbacks.Items) > 0 {
		result, err := a.repos.Feedback.CreateFeedbackItems(
			ctx, telegramUserID, newFeedbacks.Items)
		if err != nil {
			logger.Error("failed to update the feedbacks to db", "error", err)
			msgs = append(msgs, &schema.Message{
				Role:    schema.Assistant,
				Content: "failed to add the feedbacks, please let user know and make apology",
			})
			return msgs, nil
		}
		addedFeedbacks := []string{}
		for _, feedback := range result {
			addedFeedbacks = append(addedFeedbacks, feedback.Content)
		}
		if len(addedFeedbacks) > 0 {
			msgs = append(msgs, &schema.Message{
				Role: schema.Assistant,
				Content: fmt.Sprintf(
					"added the following feedbacks to the database: [%s].",
					strings.Join(addedFeedbacks, ", ")),
			})
			// add additional instruction to avoid unveiling details of the feedbacks to the user
			msgs = append(msgs, &schema.Message{
				Role:    schema.Assistant,
				Content: "let user know the added feedbacks but don't unveil the conversation summary and possible reasons",
			})
		}
	}
	return msgs, nil
}

func (a *CinnaReactAgent) parseFeedbackItems(
	ctx context.Context, rawMessage string) *FeedbackItems {
	logger := a.logger.With("Node", updateFeedbackItemsLambdaNodeName)
	var feedbackItems FeedbackItems
	if err := json.Unmarshal([]byte(rawMessage), &feedbackItems); err != nil {
		logger.Error("Invalid feedback tasks json", "error", err)
		return &FeedbackItems{Items: []string{}} // return empty object
	}
	return &feedbackItems
}
