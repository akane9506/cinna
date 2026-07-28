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
	passThroughNodeName       = "non_compression_passthrough"
)

type ReplyCompressionChainState struct {
	Compression bool
}

type ReplyCompressionOutput struct {
	Compression bool
	Reply       *schema.Message
	Memory      *schema.Message
}

type ReplyCompressionChain = *compose.Chain[[]*schema.Message, *ReplyCompressionOutput]

func (a *AgentCore) RegisterReplyCompressionChain(includeCompression bool) {
	chain := compose.NewChain[[]*schema.Message, *ReplyCompressionOutput](
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
	appendPostProcessLambda(chain, a.logger)
	a.graph.AddGraphNode(nodeName, chain)
}

// Post processor lambda
func appendPostProcessLambda(chain ReplyCompressionChain, logger *slog.Logger) {
	chain.AppendLambda(
		compose.InvokableLambda[map[string]any, *ReplyCompressionOutput](
			processParallelResult(logger),
		),
		compose.WithNodeKey(postProcessLambdaName),
	)
}

func processParallelResult(nodeLogger *slog.Logger) compose.InvokeWOOpt[map[string]any, *ReplyCompressionOutput] {
	return func(ctx context.Context, input map[string]any) (*ReplyCompressionOutput, error) {
		logger := nodeLogger.With("Node", postProcessLambdaName)
		output := &ReplyCompressionOutput{}
		var includeCompression bool = false
		compose.ProcessState[*ReplyCompressionChainState](
			ctx, func(ctx context.Context, state *ReplyCompressionChainState) error {
				includeCompression = state.Compression
				return nil
			})
		chatResult, ok := input[cinnaReplyNodeName].(*schema.Message)
		if !ok {
			logger.Error("failed to get chat node result: key does not exist", "key", cinnaReplyNodeName)
			output.Reply = &schema.Message{
				Role:    schema.Assistant,
				Content: "chat node failed (this message needs to be optimized)"}
		} else {
			output.Reply = chatResult
		}
		if includeCompression {
			compressionResult, ok := input[compressionNodeName].(*schema.Message)
			if !ok || compressionResult == nil {
				logger.Error("failed to get compression result: key does not exist", "key", compressionNodeName)
				output.Compression = false // when error happens, we set this to false to avoid it from polluting downstream process
			} else {
				output.Compression = true
				output.Memory = compressionResult
			}
		}
		return output, nil
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
	parallel.AddPassthrough(passThroughNodeName)
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
