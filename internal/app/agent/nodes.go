package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	db "github.com/akane9506/cinna/internal/postgres"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	processInputLambdaNodeName      = "process_input"
	intentLLMNodeName               = "intent_classification"
	intentLambdaNodeName            = "intent_validation"
	listShoppingItemsLambdaNodeName = "list_shopping_items"
	shoppingTasksPlannerLLMNodeName = "plan_shopping_tasks"
	shoppingTaskRunLambdaNodeName   = "execute_shopping_command"
	cinnaChatNodeName               = "cinna_response"
)

type Intention struct {
	Intent string `json:"intent"`
	Action string `json:"action"`
}

type UpdateShoppingListCommands struct {
	Commands []UpdateShoppingListCommand `json:"commands"`
}

type UpdateShoppingListCommand struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Method   string `json:"method"`
}

// ========== Task Input ==========
// It passes input data, including user id, history messages, to the agent's state
// in: *TaskInput | out: []*schema.Message
func (a *CinnaReactAgent) AddProcessInputLambdaNode() {
	a.graph.AddLambdaNode(
		processInputLambdaNodeName,
		compose.InvokableLambda[*TaskInput, []*schema.Message](a.processTaskInput),
	)
}

func (a *CinnaReactAgent) processTaskInput(
	ctx context.Context, input *TaskInput) ([]*schema.Message, error) {
	compose.ProcessState[*CinnaAgentState](
		ctx, func(ctx context.Context, state *CinnaAgentState) error {
			state.TelegramUserID = input.telegramUserID
			return nil
		},
	)
	return input.chatHistory, nil
}

// ========== Intent classifier ==========
// * Don't save the classification output in the state
//
// The first node in the graph, classifies the user's intention to decide the next step
// in: []*schema.Message | out: *schema.Message
func (a *CinnaReactAgent) AddIntentClassificationNode() {
	a.graph.AddChatModelNode(intentLLMNodeName, a.jsonModel,
		compose.WithStatePreHandler(a.prepareForIntentClassification),
	)
}

func (a *CinnaReactAgent) prepareForIntentClassification(
	ctx context.Context,
	input []*schema.Message,
	state *CinnaAgentState) ([]*schema.Message, error) {
	msgs := organizeInputMessage(input, a.prompts.IntentClassification)
	state.SystemMessage = a.prompts.IntentClassification
	state.History = msgs
	return msgs, nil
}

// The lambda node helps parse the output from the intent classification, and cleans the history messages
// IMPORTANT: we don't need to include the intention classification output in this lambda
// in: *schema.Message | out: *IntentLambdaOutput
func (a *CinnaReactAgent) AddIntentOutputLambdaNode() {
	a.graph.AddLambdaNode(
		intentLambdaNodeName,
		compose.InvokableLambda[*schema.Message, []*schema.Message](a.processIntentOutput),
	)
}

func (a *CinnaReactAgent) processIntentOutput(
	ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
	logger := a.logger.With("Node", intentLambdaNodeName)
	intentValue := IntentFailed
	actionValue := ActionNone
	if msg == nil {
		// we do not pause the agent flow even if there is an data-related error
		logger.Error("failed to get message", "error", "nil input")
	} else {
		var intent Intention
		if err := json.Unmarshal([]byte(msg.Content), &intent); err != nil {
			logger.Error(
				"failed to parse intent message",
				"error", "failed to parse message", "content", msg.Content)
		} else {
			intentValue = normalizeIntent(intent.Intent)
			actionValue = normalizeAction(intent.Action)
		}
	}
	var output []*schema.Message
	compose.ProcessState[*CinnaAgentState](
		ctx, func(ctx context.Context, state *CinnaAgentState) error {
			state.ChatIntent = intentValue
			state.ActionType = actionValue
			msgs := cleanupOutputMessage(state.History)
			if intentValue == IntentFailed {
				// if parse failed, we also notify user that the parse failed, please contact bot service
				msgs = append(msgs, &schema.Message{
					Role:    schema.Assistant,
					Content: a.prompts.IntentFailedRecovery})
			}
			state.History = msgs
			output = msgs
			return nil
		})
	return output, nil
}

// ========== Intent Branch ==========
// route the task to different branch based on the classified intent
func (a *CinnaReactAgent) AddIntentBranch() {
	endNodes := map[string]bool{}
	endNodes[listShoppingItemsLambdaNodeName] = true
	endNodes[cinnaChatNodeName] = true
	a.graph.AddBranch(intentLambdaNodeName, compose.NewGraphBranch(a.intentBranch, endNodes))
}

func (a *CinnaReactAgent) intentBranch(ctx context.Context, out []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*CinnaAgentState](
		ctx, func(ctx context.Context, state *CinnaAgentState) error {
			switch state.ChatIntent {
			case IntentShopping:
				next = listShoppingItemsLambdaNodeName // we just put this as a place holder
			case IntentOther:
				next = cinnaChatNodeName
			default:
				next = cinnaChatNodeName
			}
			return nil
		})
	return next, nil
}

// ========== List Shopping Items Lambda ==========
// List shopping list items, separate into expired and unexpired group, and format into *schema.Message
// in: *[]schema.Message | out: []*schema.Message
func (a *CinnaReactAgent) AddListShoppingItemsLambda() {
	a.graph.AddLambdaNode(
		listShoppingItemsLambdaNodeName,
		compose.InvokableLambda[[]*schema.Message, []*schema.Message](a.processShoppingListItems),
	)
}

func (a *CinnaReactAgent) processShoppingListItems(
	ctx context.Context, msgs []*schema.Message) ([]*schema.Message, error) {
	compose.ProcessState[*CinnaAgentState](
		ctx,
		func(ctx context.Context, state *CinnaAgentState) error {
			finalListMessage := a.listShoppingListItems(ctx, state)
			msgs = append(msgs, finalListMessage)
			state.History = append(state.History, finalListMessage)
			return nil
		},
	)
	return msgs, nil
}

// query the database and get the shopping list items, separate them into
// expired and unexpired parts
func (a *CinnaReactAgent) listShoppingListItems(
	ctx context.Context,
	state *CinnaAgentState) *schema.Message {
	logger := a.logger.With("Node", listShoppingItemsLambdaNodeName)
	telegramUserID := state.TelegramUserID
	// query database
	shoppingListItems, err := a.repos.ShoppingList.ListShoppingListItems(ctx, telegramUserID)
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

// ========== Shopping List Update Planner ==========
// Plans the task based on the database and user's description
// in: []*schema.Message | out: *schema.Message
func (a *CinnaReactAgent) AddShoppingTaskPlanningNode() {
	a.graph.AddChatModelNode(shoppingTasksPlannerLLMNodeName, a.jsonModel,
		compose.WithStatePreHandler(a.prepareForShoppingTaskPlanning),
	)
}

func (a *CinnaReactAgent) prepareForShoppingTaskPlanning(
	ctx context.Context,
	input []*schema.Message,
	state *CinnaAgentState) ([]*schema.Message, error) {
	state.SystemMessage = a.prompts.ShoppingListPlanner
	msgs := organizeInputMessage(input, state.SystemMessage)
	state.History = msgs
	return msgs, nil
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

func (a *CinnaReactAgent) parseShoppingListCommands(
	rawMessage string) ShoppingListCommands {
	logger := a.logger.With("Node", shoppingTaskRunLambdaNodeName)
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

func (a *CinnaReactAgent) executeShoppingListCommands(
	ctx context.Context, telegramID int64, commands ShoppingListCommands) *schema.Message {
	logger := a.logger.With("Node", shoppingTaskRunLambdaNodeName)
	shoppingListRepo := a.repos.ShoppingList
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

// ========== Cinna Chat Model ==========
//
// The last llm node in the graph, generates user-facing response
// in: []*schema.Message | out: *[]schema.Message
func (a *CinnaReactAgent) AddCinnaResponseNode() {
	a.graph.AddChatModelNode(cinnaChatNodeName, a.chatModel,
		compose.WithStatePreHandler(a.prepareForCinnaChat))
}

func (a *CinnaReactAgent) prepareForCinnaChat(
	ctx context.Context,
	input []*schema.Message,
	state *CinnaAgentState) ([]*schema.Message, error) {
	msgs := organizeInputMessage(input, a.prompts.CinnaPersona)
	state.SystemMessage = a.prompts.CinnaPersona
	state.History = msgs
	return msgs, nil
}

// ========== Helper functions ==========

func logMessages(title string, msgs []*schema.Message) {
	fmt.Println()
	fmt.Println("Start Logging", title)
	for _, msg := range msgs {
		fmt.Println(fmt.Sprintf("[%s] %s", msg.Role, msg.Content))
	}
	fmt.Println()
}

func organizeInputMessage(input []*schema.Message, systemPrompt string) []*schema.Message {
	// inject intent classification prompt
	msgs := []*schema.Message{
		&schema.Message{Role: schema.System, Content: systemPrompt}}
	// include Tool and Assistant chat history
	for _, msg := range input {
		if msg.Role == schema.System {
			continue // we just ignore system message here
		} else {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

// exclude unnecessary messages from the collection of message
func cleanupOutputMessage(output []*schema.Message) []*schema.Message {
	msgs := []*schema.Message{}
	for _, msg := range output {
		if msg.Role == schema.System {
			continue
		}
		msgs = append(msgs, msg)
	}
	return msgs
}

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
