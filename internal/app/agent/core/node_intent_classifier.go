// Intent classifier
// * Don't save the classification output in the state

package core

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

const (
	intentLLMNodeName    = "intent_classification"
	intentLambdaNodeName = "intent_validation"
)

type Intention struct {
	Intent string `json:"intent"`
	Action string `json:"action"`
}

// followed the instruction in this file:
// https://github.com/cloudwego/eino-ext/blob/main/components/model/openai/examples/structured/structured.go
func createIntentSchema() *openai.ChatCompletionResponseFormatJSONSchema {
	schemaContent := &jsonschema.Schema{
		Type: string(schema.Object),
		Properties: orderedmap.New[string, *jsonschema.Schema](
			orderedmap.WithInitialData[string, *jsonschema.Schema](
				orderedmap.Pair[string, *jsonschema.Schema]{
					Key: "intent",
					Value: &jsonschema.Schema{
						Type: string(schema.String),
					},
				},
				orderedmap.Pair[string, *jsonschema.Schema]{
					Key: "action",
					Value: &jsonschema.Schema{
						Type: string(schema.String),
					},
				},
			),
		),
		Required:             []string{"intent", "action"},
		AdditionalProperties: jsonschema.FalseSchema,
	}
	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "intent_classification",
		Description: "the intention and the action of the current round chat",
		Strict:      true,
		JSONSchema:  schemaContent,
	}
}

// Add intent classification workflow to the graph
func (a *AgentCore) RegisterIntentClassifier(ctx context.Context) {
	logger := a.logger.With(
		"path", "internal/app/agent/core/node_intent_classifier/RegisterIntentClassifier")
	model, err := a.models.CreateJSONModel(ctx, createIntentSchema())
	if err != nil {
		logger.Error("failed to register intent classifier")
		return
	}
	addIntentClassificationNode(a.graph, model, a.prompts.IntentClassification)
	addIntentOutputLambdaNode(a.graph, a.logger)
}

// Classifies the user's intention and actions with JSON Model
// in: []*schema.Message | out: *schema.Message
func addIntentClassificationNode(graph Graph, jsonModel model.BaseChatModel, prompt string) {
	graph.AddChatModelNode(intentLLMNodeName, jsonModel,
		compose.WithStatePreHandler(prepareForIntentClassification(prompt)),
	)
}

func prepareForIntentClassification(
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

// Parses the intent classification message and cleans the history messages
// IMPORTANT: we don't need to include the intention classification output from lambda
// in: *schema.Message | out: *IntentLambdaOutput
func addIntentOutputLambdaNode(graph Graph, logger *slog.Logger) {
	graph.AddLambdaNode(
		intentLambdaNodeName,
		compose.InvokableLambda[*schema.Message, []*schema.Message](processIntentOutput(logger)),
	)
}

func processIntentOutput(
	agentLogger *slog.Logger,
) compose.InvokeWOOpt[*schema.Message, []*schema.Message] {
	return func(ctx context.Context, msg *schema.Message) ([]*schema.Message, error) {
		logger := agentLogger.With("Node", intentLambdaNodeName)
		intentValue := IntentOther
		actionValue := ActionNone
		if msg == nil {
			// we do not pause the agent flow even if there is a data-related error
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
		compose.ProcessState[*AgentState](
			ctx, func(ctx context.Context, state *AgentState) error {
				state.ChatIntent = intentValue
				state.ActionType = actionValue
				msgs := cleanupOutputMessage(state.History)
				state.History = msgs
				output = msgs
				return nil
			})
		return output, nil
	}
}
