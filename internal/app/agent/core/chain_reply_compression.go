package core

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	replyCompressionChainName = "reply_compression"
	cinnaReplyNodeName        = "cinna_reply"
	postProcessLambdaName     = "parallel_post_process"
)

type ReplyCompressionChainState struct {
	// History     []*schema.Message
	Compression bool
	// Summary     string
}

type ReplyCompressionChain = *compose.Chain[[]*schema.Message, *schema.Message]

func (a *AgentCore) RegisterReplyCompressionChain() {
	chain := compose.NewChain[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(
			func(ctx context.Context) (state *ReplyCompressionChainState) {
				return &ReplyCompressionChainState{
					Compression: false,
				}
			},
		),
	)
	// appendCinnaChatNode(chain, a.chatModel, a.prompts.CinnaPersona)
	appendParallelNode(chain, a.chatModel, a.prompts.CinnaPersona)
	appendPostProcessLambda(chain, a.logger)
	a.graph.AddGraphNode(replyCompressionChainName, chain)
}

// Parallel post processor lambda
func appendPostProcessLambda(chain ReplyCompressionChain, logger *slog.Logger) {
	chain.AppendLambda(
		compose.InvokableLambda[map[string]any, *schema.Message](
			processParallelResult(logger),
		),
		compose.WithNodeKey(postProcessLambdaName),
	)
}

func processParallelResult(nodeLogger *slog.Logger) compose.InvokeWOOpt[map[string]any, *schema.Message] {
	return func(ctx context.Context, input map[string]any) (*schema.Message, error) {
		logger := nodeLogger.With("Node", postProcessLambdaName)
		chatResult, ok := input[cinnaReplyNodeName].(*schema.Message)
		if !ok {
			logger.Error("failed to get chat node result: key does not exist", "key", cinnaReplyNodeName)
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "chat node failed (this message needs to be optimized)"}, nil
		}
		return chatResult, nil
	}
}

// Parallel node
// in: []*schema.Message | out: map[string]any
func appendParallelNode(chain ReplyCompressionChain, chatModel ChatModel, chatPrompt string) {
	parallel := compose.NewParallel()
	parallel.AddChatModel(
		cinnaReplyNodeName,
		chatModel,
		compose.WithNodeKey(cinnaReplyNodeName),
		compose.WithStatePreHandler(prepareForChat(chatPrompt)))
	parallel.AddPassthrough("empty")
	chain.AppendParallel(parallel)
}

func prepareForChat(prompt string,
) compose.StatePreHandler[[]*schema.Message, *ReplyCompressionChainState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *ReplyCompressionChainState) ([]*schema.Message, error) {
		msgs := organizeInputMessageWithoutSensitiveInfo(input, prompt)
		return msgs, nil
	}
}
