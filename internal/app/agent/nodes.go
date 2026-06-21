package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	intentLLMNodeName    = "intent_classification"
	intentLambdaNodeName = "intent_validation"
	cinnaChatNodeName    = "cinna_response"
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

// will work on this one next
func (a *CinnaReactAgent) AddIntentOutputLambdaNode() {
	a.graph.AddLambdaNode(
		intentLambdaNodeName,
		compose.InvokableLambda[*schema.Message, []*schema.Message](func(
			ctx context.Context,
			msg *schema.Message,
		) ([]*schema.Message, error) {

			return nil, nil
		}),
	)
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
