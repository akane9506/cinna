// Feedbacks workflow

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	listFeedbackItemsLambdaNodeName   = "list_feedback_items"
	feedbacksUpdatePlannerLLMNodeName = "plan_feedback_updates"
	updateFeedbackItemsLambdaNodeName = "update_feedback_items"
)

func (a *AgentCore) RegisterFeedbacksWorkflow() {
	addListFeedbackItemsLambda(a.graph, a.repos.Feedback, a.logger)
	addFeedbacksPlanningNode(a.graph, a.jsonModel, a.prompts.FeedbacksPlanner)
	addUpdateFeedbacksLambdaNode(a.graph, a.repos.Feedback, a.logger)
}

func (a *AgentCore) AddFeedbacksActionBranch() {
	addFeedbacksActionBranch(a.graph)
}

// ========== List Feedback Items Lambda ==========
// List feedbacks in the database
// in: []*schema.Message | out: []*schema.Message
func addListFeedbackItemsLambda(
	graph Graph,
	repo ports.FeedbacksRepository,
	logger *slog.Logger) {
	graph.AddLambdaNode(
		listFeedbackItemsLambdaNodeName,
		compose.InvokableLambda[[]*schema.Message, []*schema.Message](
			processFeedbackList(repo, logger)),
	)
}

func processFeedbackList(
	repo ports.FeedbacksRepository,
	logger *slog.Logger) compose.InvokeWOOpt[[]*schema.Message, []*schema.Message] {
	return func(
		ctx context.Context,
		msgs []*schema.Message) ([]*schema.Message, error) {
		compose.ProcessState[*AgentState](ctx,
			func(ctx context.Context, state *AgentState) error {
				// we only add DB feedback items to messages when the action is "UPDATE"
				// other wise feedback will be inspected from the user's chat history
				if state.ActionType == ActionUpdate {
					feedbackItemsMessage := listFeedbackItems(ctx, state, repo, logger)
					msgs = append(msgs, feedbackItemsMessage)
				}
				return nil
			},
		)
		return msgs, nil
	}
}

func listFeedbackItems(
	ctx context.Context,
	state *AgentState,
	feedbacksRepo ports.FeedbacksRepository,
	nodeLogger *slog.Logger,
) *schema.Message {
	logger := nodeLogger.With("Node", listFeedbackItemsLambdaNodeName)
	telegramUserID := state.TelegramUserID
	// query database
	// *IMPORTANT* never add this list to the chat history
	currList, err := feedbacksRepo.ListIncompleteFeedbacks(ctx)
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
		"%s feedbacks in the current database: [%s]",
		SENSITIVE_PREFIX,
		strings.Join(feedbackContents, ", "),
	)
	return &schema.Message{
		Role:    schema.Assistant,
		Content: message,
	}
}

// ========== Feedback Action Branch ==========
func addFeedbacksActionBranch(graph Graph) {
	endNodes := map[string]bool{}
	endNodes[feedbacksUpdatePlannerLLMNodeName] = true
	endNodes[replyCompressionChainName] = true
	graph.AddBranch(listFeedbackItemsLambdaNodeName,
		compose.NewGraphBranch(feedbacksActionBranch, endNodes))
}

func feedbacksActionBranch(
	ctx context.Context, out []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			switch state.ActionType {
			case ActionUpdate:
				next = feedbacksUpdatePlannerLLMNodeName
			default:
				next = replyCompressionChainName
			}
			return nil
		})
	return next, nil
}

// ========= Feedbacks Planner ==========
// Plan the task based the current feedbacks from the database and the user's message
// in: []*schema.Message | out: *schema.Message
func addFeedbacksPlanningNode(graph Graph, jsonModel JSONModel, prompt string) {
	graph.AddChatModelNode(
		feedbacksUpdatePlannerLLMNodeName,
		jsonModel,
		compose.WithStatePreHandler(prepareForFeedbacksPlanning(prompt)),
	)
}

func prepareForFeedbacksPlanning(
	prompt string,
) compose.StatePreHandler[[]*schema.Message, *AgentState] {
	return func(
		ctx context.Context,
		input []*schema.Message,
		state *AgentState,
	) ([]*schema.Message, error) {
		msgs := organizeInputMessage(input, prompt)
		state.History = msgs
		return msgs, nil
	}
}

// ========= Update Feedback Items Lambda ==========

type FeedbackItems struct {
	Items []string `json:"items"`
}

func addUpdateFeedbacksLambdaNode(
	graph Graph,
	repo ports.FeedbacksRepository,
	logger *slog.Logger,
) {
	graph.AddLambdaNode(
		updateFeedbackItemsLambdaNodeName,
		compose.InvokableLambda[*schema.Message, []*schema.Message](
			runUpdateFeedbacksCommands(repo, logger)),
	)
}

func runUpdateFeedbacksCommands(
	feedbacksRepo ports.FeedbacksRepository,
	nodeLogger *slog.Logger,
) compose.InvokeWOOpt[*schema.Message, []*schema.Message] {
	return func(ctx context.Context,
		msg *schema.Message) ([]*schema.Message, error) {
		logger := nodeLogger.With("Node", updateFeedbackItemsLambdaNodeName)
		newFeedbacks := parseFeedbackItems(ctx, msg.Content, nodeLogger)
		var telegramUserID int64
		var msgs []*schema.Message
		compose.ProcessState[*AgentState](ctx,
			func(ctx context.Context, state *AgentState) error {
				telegramUserID = state.TelegramUserID
				msgs = state.History
				return nil
			})
		if len(newFeedbacks.Items) > 0 {
			results := executeFeedbacksCommands(
				ctx, feedbacksRepo, telegramUserID, newFeedbacks, logger)
			msgs = append(msgs, results...)
		}
		return msgs, nil
	}
}

func executeFeedbacksCommands(
	ctx context.Context,
	feedbacksRepo ports.FeedbacksRepository,
	telegramUserID int64,
	newFeedbacks *FeedbackItems,
	logger *slog.Logger,
) []*schema.Message {
	msgs := []*schema.Message{}
	result, err := feedbacksRepo.CreateFeedbackItems(
		ctx, telegramUserID, newFeedbacks.Items)
	if err != nil {
		logger.Error("failed to update the feedbacks to db", "error", err)
		msgs = append(msgs, &schema.Message{
			Role:    schema.Assistant,
			Content: "failed to add the feedbacks, please let user know and make apology",
		})
		return msgs
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
	return msgs
}

func parseFeedbackItems(
	ctx context.Context,
	rawMessage string,
	logger *slog.Logger) *FeedbackItems {
	var feedbackItems FeedbackItems
	if err := json.Unmarshal([]byte(rawMessage), &feedbackItems); err != nil {
		logger.Error("Invalid feedback tasks json", "error", err)
		return &FeedbackItems{Items: []string{}} // return empty object
	}
	return &feedbackItems
}
