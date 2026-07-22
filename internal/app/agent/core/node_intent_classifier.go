// Intent classifier
// * Don't save the classification output in the state

package core

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	intentLLMNodeName    = "intent_classification"
	intentLambdaNodeName = "intent_validation"
)

func (a *AgentCore) RegisterIntentClassifier() {
	addIntentClassificationNode(a.graph, a.jsonModel, a.prompts.IntentClassification)
	addIntentOutputLambdaNode(a.graph, a.logger)
}

// Classifies the user's intention and actions with JSON Model
// in: []*schema.Message | out: *schema.Message
func addIntentClassificationNode(graph Graph, jsonModel *deepseek.ChatModel, prompt string) {
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
