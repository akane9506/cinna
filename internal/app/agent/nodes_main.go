package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	processInputLambdaNodeName = "process_input"
	intentLLMNodeName          = "intent_classification"
	intentLambdaNodeName       = "intent_validation"
	cinnaChatNodeName          = "cinna_response"
)

type Intention struct {
	Intent string `json:"intent"`
	Action string `json:"action"`
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
	endNodes[listFeedbackItemsLambdaNodeName] = true
	endNodes[cinnaChatNodeName] = true
	a.graph.AddBranch(intentLambdaNodeName, compose.NewGraphBranch(a.intentBranch, endNodes))
}

func (a *CinnaReactAgent) intentBranch(ctx context.Context, out []*schema.Message) (string, error) {
	var next string
	compose.ProcessState[*CinnaAgentState](
		ctx, func(ctx context.Context, state *CinnaAgentState) error {
			switch state.ChatIntent {
			case IntentShopping:
				next = listShoppingItemsLambdaNodeName
			case IntentFeedback:
				next = listFeedbackItemsLambdaNodeName
			default:
				next = cinnaChatNodeName
			}
			return nil
		})
	return next, nil
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
