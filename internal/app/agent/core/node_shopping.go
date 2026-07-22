// Shopping list workflow

package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/akane9506/cinna/internal/app/ports"
	db "github.com/akane9506/cinna/internal/postgres"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	listShoppingItemsLambdaNodeName = "list_shopping_items"
	shoppingTasksPlannerLLMNodeName = "plan_shopping_tasks"
	shoppingTaskRunLambdaNodeName   = "execute_shopping_command"
)

type UpdateShoppingListCommands struct {
	Commands []UpdateShoppingListCommand `json:"commands"`
}

type UpdateShoppingListCommand struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Method   string `json:"method"`
}

func (a *AgentCore) RegisterShoppingListWorkflow() {
	addListShoppingItemsLambda(a.graph, a.repos.ShoppingList, a.logger)
	addShoppingTaskPlanningNode(a.graph, a.jsonModel, a.prompts.ShoppingListPlanner)
	addRunShoppingTaskLambdaNode(a.graph, a.repos.ShoppingList, a.logger)
}

func (a *AgentCore) AddShoppingActionBranch() {
	addShoppingActionBranch(a.graph)
}

// ========== List Shopping Items Lambda ==========
// List shopping list items, separate into expired and unexpired group, and format into *schema.Message
// in: *[]schema.Message | out: []*schema.Message
func addListShoppingItemsLambda(
	graph Graph, repo ports.ShoppingListRepository, logger *slog.Logger) {
	graph.AddLambdaNode(
		listShoppingItemsLambdaNodeName,
		compose.InvokableLambda[[]*schema.Message, []*schema.Message](
			processShoppingListItems(repo, logger)),
	)
}

func processShoppingListItems(
	repo ports.ShoppingListRepository,
	logger *slog.Logger,
) compose.InvokeWOOpt[[]*schema.Message, []*schema.Message] {
	return func(ctx context.Context, msgs []*schema.Message) ([]*schema.Message, error) {
		compose.ProcessState[*AgentState](
			ctx,
			func(ctx context.Context, state *AgentState) error {
				finalListMessage := listShoppingListItems(ctx, state, repo, logger)
				msgs = append(msgs, finalListMessage)
				state.History = append(state.History, finalListMessage)
				return nil
			},
		)
		return msgs, nil
	}
}

// query the database and get the shopping list items, separate them into
// expired and unexpired parts
func listShoppingListItems(
	ctx context.Context,
	state *AgentState,
	repo ports.ShoppingListRepository,
	nodeLogger *slog.Logger,
) *schema.Message {
	logger := nodeLogger.With("Node", listShoppingItemsLambdaNodeName)
	telegramUserID := state.TelegramUserID
	// query database
	shoppingListItems, err := repo.ListShoppingListItems(ctx, telegramUserID)
	if err != nil {
		logger.Error("failed to get shopping list items", "error", err)
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "failed to get current list from database",
		}
	}
	var activeItems, expiredItems []string
	// group items into expired and unexpired
	for _, item := range shoppingListItems {
		itemName := formatItemName(item)
		if db.IsExpired(&item) {
			expiredItems = append(expiredItems, itemName)
		} else {
			activeItems = append(activeItems, itemName)
		}
	}
	// format message
	finalList := fmt.Sprintf(
		"items in the current database: {activeItems: [%s], expiredItems: [%s]}",
		strings.Join(activeItems, ", "),
		strings.Join(expiredItems, ", "),
	)
	finalListMessage := &schema.Message{
		Role:    schema.Assistant,
		Content: finalList,
	}
	return finalListMessage
}

// ========== Shopping Action Branch ==========
// route the task to different branch based on the classified intent
func addShoppingActionBranch(graph Graph) {
	endNodes := map[string]bool{}
	endNodes[shoppingTasksPlannerLLMNodeName] = true
	endNodes[cinnaChatNodeName] = true
	graph.AddBranch(
		listShoppingItemsLambdaNodeName,
		compose.NewGraphBranch(shoppingActionBranch, endNodes))
}

func shoppingActionBranch(ctx context.Context, out []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*AgentState](
		ctx, func(ctx context.Context, state *AgentState) error {
			switch state.ActionType {
			case ActionUpdate:
				next = shoppingTasksPlannerLLMNodeName
			case ActionList:
				next = cinnaChatNodeName
			default:
				next = cinnaChatNodeName
			}
			return nil
		})
	return next, nil
}

// ========== Shopping List Update Planner ==========
// Plans the task based on the database and user's description
// in: []*schema.Message | out: *schema.Message
func addShoppingTaskPlanningNode(graph Graph, jsonModel JSONModel, prompt string) {
	graph.AddChatModelNode(shoppingTasksPlannerLLMNodeName, jsonModel,
		compose.WithStatePreHandler(prepareForShoppingTaskPlanning(prompt)),
	)
}

func prepareForShoppingTaskPlanning(
	prompt string,
) compose.StatePreHandler[[]*schema.Message, *AgentState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *AgentState) ([]*schema.Message, error) {
		msgs := organizeInputMessage(input, prompt)
		state.History = msgs
		return msgs, nil
	}
}

// ========== Update Shopping List Lambda ==========
// parse the output from the planner, and execute commands
// in: *schema.Message | out: []*schema.Message

type ShoppingListCommandParams struct {
	ItemIds    []int64
	ItemNames  []string
	Categories []string
}

type ShoppingListCommands map[DBMethod]*ShoppingListCommandParams

func addRunShoppingTaskLambdaNode(
	graph Graph, repo ports.ShoppingListRepository, logger *slog.Logger) {
	graph.AddLambdaNode(
		shoppingTaskRunLambdaNodeName,
		compose.InvokableLambda[*schema.Message, []*schema.Message](
			runShoppingListCommands(repo, logger)),
	)
}

func runShoppingListCommands(
	repo ports.ShoppingListRepository,
	logger *slog.Logger,
) compose.InvokeWOOpt[*schema.Message, []*schema.Message] {
	return func(ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
		// parse input message into commands
		commands := parseShoppingListCommands(msg.Content, logger)
		var telegramUserID int64
		var msgs []*schema.Message
		compose.ProcessState[*AgentState](ctx,
			func(ctx context.Context, state *AgentState) error {
				telegramUserID = state.TelegramUserID
				msgs = state.History
				return nil
			})
		// execute commands
		result := executeShoppingListCommands(ctx, telegramUserID, commands, repo, logger)
		msgs = append(msgs, result)
		return msgs, nil
	}
}

func parseShoppingListCommands(
	rawMessage string,
	nodeLogger *slog.Logger,
) ShoppingListCommands {
	logger := nodeLogger.With("Node", shoppingTaskRunLambdaNodeName)
	organizedCommands := ShoppingListCommands{}
	// parse message into commands
	var rawCommands UpdateShoppingListCommands
	if err := json.Unmarshal([]byte(rawMessage), &rawCommands); err != nil {
		logger.Error("Invalid json", "error", err)
		return organizedCommands
	}
	// iterate over each command, and put valid ones into the command params
	for _, command := range rawCommands.Commands {
		// validate method
		method, err := normalizeMethod(command.Method)
		if err != nil {
			logger.Error("Invalid method", "error", err)
			continue
		}
		command.Method = string(method) // avoid case and white space problems
		// validate name
		itemName := strings.TrimSpace(command.Name)
		if itemName == "" {
			continue
		}
		category := normalizeShoppingCategory(command.Category)
		// validate id (only when the method is not ADD)
		var id int64
		if method != MethodAdd {
			id, err = strconv.ParseInt(command.ID, 10, 64)
			if err != nil || id <= 0 {
				logger.Error("Invalid id", "id", command.ID, "error", err)
				continue
			}
		}
		params, ok := organizedCommands[method]
		if !ok {
			params = &ShoppingListCommandParams{}
		}
		if method != MethodAdd {
			params.ItemIds = append(params.ItemIds, id)
		}
		params.Categories = append(params.Categories, string(category))
		params.ItemNames = append(params.ItemNames, itemName)
		organizedCommands[method] = params
	}
	return organizedCommands
}

func executeShoppingListCommands(
	ctx context.Context,
	telegramID int64,
	commands ShoppingListCommands,
	shoppingListRepo ports.ShoppingListRepository,
	nodeLogger *slog.Logger,
) *schema.Message {
	logger := nodeLogger.With("Node", shoppingTaskRunLambdaNodeName)
	var messageContents []string
	// add items to shopping list
	for method := range commands {
		params, ok := commands[method]
		if !ok {
			continue
		}
		ids := params.ItemIds
		itemName := params.ItemNames
		categories := params.Categories
		// validate input param sizes
		if len(itemName) != len(categories) {
			logger.Error("invalid params",
				"method", method,
				"item_name size", len(itemName),
				"category size", len(categories),
			)
			continue
		}
		if method != MethodAdd && (len(ids) != len(itemName)) {
			logger.Error("invalid params",
				"method", method,
				"ids size", len(ids),
				"item_name size", len(itemName),
			)
			continue
		}
		var results []sqlc.ShoppingList
		var err error
		switch method {
		case MethodAdd:
			results, err = shoppingListRepo.CreateShoppingListItems(
				ctx,
				sqlc.CreateShoppingListItemsParams{
					TelegramUserID: telegramID,
					ItemNames:      params.ItemNames,
					Categories:     params.Categories,
				},
			)
		case MethodModify:
			results, err = shoppingListRepo.UpdateShoppingListItems(
				ctx,
				sqlc.UpdateShoppingListItemsParams{
					TelegramUserID: telegramID,
					ItemIds:        params.ItemIds,
					ItemNames:      params.ItemNames,
					Categories:     params.Categories,
				},
			)
		case MethodRemove:
			results, err = shoppingListRepo.RemoveShoppingListItems(
				ctx,
				sqlc.RemoveShoppingListItemsParams{
					TelegramUserID: telegramID,
					ItemIds:        params.ItemIds,
				},
			)
		}
		if err != nil {
			logger.Error("failed to update shopping list", "method", method, "error", err)
			continue
		}
		// add formatted results to the response message
		formattedResult := formatUpdateResult(method, results)
		if formattedResult != "" {
			messageContents = append(messageContents, formattedResult)
		}
	}
	if len(messageContents) == 0 {
		return &schema.Message{
			Role:    schema.Assistant,
			Content: "no updates, please let the user know", // should use a better version of content
		}
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: strings.Join(messageContents, ". "),
	}
}

// ========== Helper functions ==========

// formate shopping list item name
func formatItemName(item sqlc.ShoppingList) string {
	return fmt.Sprintf(
		"{id: %d, category: %s, name: %s}",
		item.ID,
		item.Category,
		item.Name)
}

// format db update results into human/llm-friendly assistant messages
func formatUpdateResult(method DBMethod, results []sqlc.ShoppingList) string {
	if len(results) == 0 {
		return ""
	}
	var items []string
	for _, result := range results {
		items = append(items, result.Name)
	}
	output := fmt.Sprintf("Successfully %s shopping list: %s", method, strings.Join(items, ", "))
	return output
}
