package core

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	replyOnlyChainName        = "reply_only_chain"
	replyCompressionChainName = "reply_compression_chain"
	cinnaReplyNodeName        = "cinna_reply"
	compressionNodeName       = "memory_compression"
	postProcessLambdaName     = "parallel_post_process"
)

type ReplyCompressionChainState struct {
	Compression bool
}

type ReplyCompressionOutput struct {
	Compression bool
	Reply       *schema.Message
	Memory      *schema.Message
}

type ReplyCompressionChain = *compose.Chain[[]*schema.Message, *schema.Message]

func (a *AgentCore) RegisterReplyCompressionChain(includeCompression bool) {
	chain := compose.NewChain[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(
			func(ctx context.Context) (state *ReplyCompressionChainState) {
				return &ReplyCompressionChainState{
					Compression: false,
				}
			},
		),
	)
	nodeName := replyCompressionChainName
	if !includeCompression {
		appendReplyOnlyParallelNode(chain, a.chatModel, a.prompts.CinnaPersona)
		nodeName = replyOnlyChainName
	} else {
		appendReplyCompressionParallelNode(chain,
			a.chatModel, a.prompts.CinnaPersona, a.prompts.MemoryCompression)
	}
	appendPostProcessLambda(chain, a.logger, includeCompression)
	a.graph.AddGraphNode(nodeName, chain)
}

// Post processor lambda
func appendPostProcessLambda(chain ReplyCompressionChain, logger *slog.Logger, includeCompression bool) {
	chain.AppendLambda(
		compose.InvokableLambda[map[string]any, *schema.Message](
			processParallelResult(logger, includeCompression),
		),
		compose.WithNodeKey(postProcessLambdaName),
	)
}

func processParallelResult(nodeLogger *slog.Logger, includeCompression bool) compose.InvokeWOOpt[map[string]any, *schema.Message] {
	return func(ctx context.Context, input map[string]any) (*schema.Message, error) {
		logger := nodeLogger.With("Node", postProcessLambdaName)
		chatResult, ok := input[cinnaReplyNodeName].(*schema.Message)
		if !ok {
			logger.Error("failed to get chat node result: key does not exist", "key", cinnaReplyNodeName)
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "chat node failed (this message needs to be optimized)"}, nil
		}
		if includeCompression {
			compressionResult, ok := input[compressionNodeName].(*schema.Message)
			if !ok || compressionResult == nil {
				logger.Error("failed to get compression result: key does not exist", "key", compressionNodeName)
			} else {
				// keep this one for the production test
				logger.Info("memory compression", "content", compressionResult.Content)
			}
		}
		return chatResult, nil
	}
}

// Parallel node without compression
// in: []*schema.Message | out: map[string]any
func appendReplyOnlyParallelNode(chain ReplyCompressionChain, chatModel ChatModel, chatPrompt string) {
	parallel := compose.NewParallel()
	parallel.AddChatModel(
		cinnaReplyNodeName,
		chatModel,
		compose.WithNodeKey(cinnaReplyNodeName),
		compose.WithStatePreHandler(prepareForReply(chatPrompt)))
	parallel.AddPassthrough("empty")
	chain.AppendParallel(parallel)
}

// Parallel node with compression
// in: []*schema.Message | out: map[string]any
func appendReplyCompressionParallelNode(
	chain ReplyCompressionChain,
	chatModel ChatModel,
	replyPrompt string, compressionPrompt string) {
	parallel := compose.NewParallel()
	parallel.AddChatModel(
		cinnaReplyNodeName,
		chatModel,
		compose.WithNodeKey(cinnaReplyNodeName),
		compose.WithStatePreHandler(prepareForReply(replyPrompt)))
	parallel.AddChatModel(
		compressionNodeName,
		chatModel,
		compose.WithNodeKey(compressionNodeName),
		compose.WithStatePreHandler(prepareForCompression(compressionPrompt)),
	)
	chain.AppendParallel(parallel)
}

func prepareForCompression(prompt string,
) compose.StatePreHandler[[]*schema.Message, *ReplyCompressionChainState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *ReplyCompressionChainState) ([]*schema.Message, error) {
		state.Compression = true
		msgs := organizeCompressionMessage(input, prompt)
		return msgs, nil
	}
}

func prepareForReply(prompt string,
) compose.StatePreHandler[[]*schema.Message, *ReplyCompressionChainState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *ReplyCompressionChainState) ([]*schema.Message, error) {
		msgs := organizeInputMessageWithoutSensitiveInfo(input, prompt)
		return msgs, nil
	}
}
