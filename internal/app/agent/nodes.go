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
	intentLLMNodeName    = "intent_classification"
	intentLambdaNodeName = "intent_validation"
	cinnaChatNodeName    = "cinna_response"
)

type Intention struct {
	Intent string `json:"intent"`
}

type IntentLambdaOutput struct {
	Intent   string
	Messages []*schema.Message
}

const (
	IntentOther    string = "OTHER"
	IntentShopping string = "SHOPPING"
	IntentFeedback string = "FEEDBACK"
	IntentFailed   string = "FAILED"
)

//	========== Intent classifier ==========
//
// The first node in the graph, classifies the user's intention to decide the next step
// in: []*schema.Message | out: *schema.Message
func (a *CinnaReactAgent) AddIntentClassificationNodes() {
	a.graph.AddChatModelNode(intentLLMNodeName, a.jsonModel,
		compose.WithStatePreHandler(
			func(
				ctx context.Context,
				input []*schema.Message,
				state *HistoryMessage) ([]*schema.Message, error) {
				msgs := organizeInputMessage(input, a.prompts.IntentClassification)
				state.SystemMessage = a.prompts.IntentClassification
				state.History = msgs
				return msgs, nil
			}),
	)
}

// The lambda node helps parse the output from the intent classification, and cleans the history messages
// IMPORTANT: we don't need to include the intention classification output in this lambda
// in: *schema.Message | out: *IntentLambdaOutput
func (a *CinnaReactAgent) AddIntentOutputLambdaNode() {
	a.graph.AddLambdaNode(
		intentLambdaNodeName,
		compose.InvokableLambda[*schema.Message, *IntentLambdaOutput](func(
			ctx context.Context,
			msg *schema.Message,
		) (*IntentLambdaOutput, error) {
			logger := a.logger.With("Node", intentLambdaNodeName)
			if msg == nil {
				// we do not pause the agent flow even if there is an data-related error
				logger.Error("failed to get message", "error", "nil input")
				return &IntentLambdaOutput{
					Intent: IntentFailed,
				}, nil
			}
			var intent Intention
			if err := json.Unmarshal([]byte(msg.Content), &intent); err != nil {
				logger.Error(
					"failed to parse intent message",
					"error", "failed to parse message", "content", msg.Content)
				intent.Intent = IntentFailed
			}
			return &IntentLambdaOutput{
				Intent: normalizeIntent(intent.Intent),
			}, nil
		}),
		compose.WithStatePostHandler(
			func(
				ctx context.Context,
				output *IntentLambdaOutput,
				state *HistoryMessage) (*IntentLambdaOutput, error) {
				msgs := cleanupOutputMessage(state.History)
				if output.Intent == IntentFailed {
					// if parse failed, we also notify user that the parse failed, please contact bot service
					msgs = append(msgs, &schema.Message{
						Role:    schema.Assistant,
						Content: a.prompts.IntentFailedRecovery})
				}
				state.History = msgs
				output.Messages = state.History
				return output, nil
			},
		),
	)
}

// this node goes after the intent classification branch; it removes the intent
// but preserves (outputs) the []*schema.Message
func (a *CinnaReactAgent) AddIntentCleanupLambdaNode() {

}

// ========== Cinna Chat Model ==========
//
// The last llm node in the graph, generates user-facing response
// in: []*schema.Message | out: *[]schema.Message
func (a *CinnaReactAgent) AddCinnaResponseNode() {
	a.graph.AddChatModelNode(cinnaChatNodeName, a.chatModel,
		compose.WithStatePreHandler(
			func(
				ctx context.Context,
				input []*schema.Message,
				state *HistoryMessage) ([]*schema.Message, error) {
				msgs := organizeInputMessage(input, a.prompts.CinnaPersona)
				state.SystemMessage = a.prompts.CinnaPersona
				state.History = msgs
				return msgs, nil
			}),
	)
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

// normalize the parsed intent
func normalizeIntent(raw string) string {
	raw = strings.ToUpper(raw)
	switch raw {
	case IntentShopping, IntentFeedback, IntentOther:
		return raw
	default:
		return IntentFailed
	}
}
