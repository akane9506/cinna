package core

import (
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

const SENSITIVE_PREFIX = "[SENSITIVE]"

type GraphInput struct {
	TelegramUserID int64
	ChatHistory    []*schema.Message
}

type GraphOutput struct {
	Compression   bool
	OutputMessage *schema.Message
	ChatHistory   []*schema.Message
	Memory        *schema.Message
}

// For intent classification
type Intent string

const (
	IntentOther    Intent = "OTHER"
	IntentShopping Intent = "SHOPPING"
	IntentFeedback Intent = "FEEDBACK"
)

// for action classification
type Action string

const (
	ActionList   Action = "LIST"
	ActionUpdate Action = "UPDATE"
	ActionNone   Action = "NONE"
)

// for db updates
type DBMethod string

const (
	MethodAdd    DBMethod = "ADD"
	MethodRemove DBMethod = "REMOVE"
	MethodModify DBMethod = "MODIFY"
)

// shopping list categories
type ShoppingCategory string

const (
	ShoppingGrocery    ShoppingCategory = "grocery"
	ShoppingPharmacy   ShoppingCategory = "pharmacy"
	ShoppingPet        ShoppingCategory = "pet"
	ShoppingToy        ShoppingCategory = "toy"
	ShoppingStationary ShoppingCategory = "stationery"
	ShoppingOther      ShoppingCategory = "other"
)

// normalize the parsed intent
func normalizeIntent(raw string) Intent {
	intent := Intent(strings.ToUpper(strings.TrimSpace(raw)))
	switch intent {
	case IntentShopping, IntentFeedback, IntentOther:
		return intent
	default:
		return IntentOther
	}
}

// normalize the parsed action type
func normalizeAction(raw string) Action {
	action := Action(strings.ToUpper(strings.TrimSpace(raw)))
	switch action {
	case ActionList, ActionUpdate, ActionNone:
		return action
	default:
		return ActionNone
	}
}

// normalize db update method
// we need an error return for this one because the method affects the db operation
// invalid operation type will ruin the db
func normalizeMethod(raw string) (DBMethod, error) {
	method := DBMethod(strings.ToUpper(strings.TrimSpace(raw)))
	switch method {
	case MethodAdd, MethodModify, MethodRemove:
		return method, nil
	default:
		return "", fmt.Errorf("invalid method type: %s", method)
	}
}

// normalize shopping category
func normalizeShoppingCategory(raw string) ShoppingCategory {
	category := ShoppingCategory(strings.ToLower(strings.TrimSpace(raw)))
	switch category {
	case ShoppingGrocery, ShoppingPharmacy, ShoppingPet, ShoppingToy, ShoppingStationary, ShoppingOther:
		return category
	default:
		return ShoppingOther
	}
}

// log messages with fmt.Println method, not logging
func logMessages(title string, msgs []*schema.Message) {
	fmt.Println()
	fmt.Println("Start Logging", title)
	for _, msg := range msgs {
		fmt.Println(fmt.Sprintf("[%s] %s", msg.Role, msg.Content))
	}
	fmt.Println()
}

// clean up messages and insert system prompt at the beginning of the message queue
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

// clean up messages and sensitive contents,
// and insert system prompt at the beginning of the message queue
func organizeInputMessageWithoutSensitiveInfo(
	input []*schema.Message, systemPrompt string) []*schema.Message {
	// inject intent classification prompt
	msgs := []*schema.Message{
		&schema.Message{Role: schema.System, Content: systemPrompt}}
	// include Tool and Assistant chat history
	for _, msg := range input {
		if msg.Role == schema.System {
			continue // we just ignore system message here
		} else if strings.HasPrefix(msg.Content, SENSITIVE_PREFIX) {
			continue // also exclude sensitive information
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

func organizeCompressionMessage(
	input []*schema.Message,
	prompt string,
) []*schema.Message {
	var history strings.Builder
	for _, msg := range input {
		if msg.Role == schema.System {
			continue
		}
		if strings.HasPrefix(msg.Content, SENSITIVE_PREFIX) {
			continue
		}
		switch msg.Role {
		case schema.User:
			history.WriteString("[USER]\n")
		case schema.Assistant:
			history.WriteString("[ASSISTANT]\n")
		default:
			continue
		}
		history.WriteString(msg.Content)
		history.WriteString("\n\n")
	}
	return []*schema.Message{
		schema.SystemMessage(prompt),
		schema.UserMessage(history.String()),
	}
}
