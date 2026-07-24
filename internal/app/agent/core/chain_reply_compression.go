package core

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const replyCompressionChainName = "cinna_reply_compression"
const cinnaReplyNodeName = "cinna_reply"

type ReplyCompressionChainState struct {
	History     []*schema.Message
	Compression bool
	Summary     string
}

type ReplyCompressionChain = *compose.Chain[[]*schema.Message, *schema.Message]

func (a *AgentCore) RegisterReplyCompressionChain() {
	chain := compose.NewChain[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(
			func(ctx context.Context) (state *ReplyCompressionChainState) {
				return &ReplyCompressionChainState{
					History:     make([]*schema.Message, 0, 4),
					Compression: false,
					Summary:     "",
				}
			},
		),
	)
	appendCinnaChatNode(chain, a.chatModel, a.prompts.CinnaPersona)
	a.graph.AddGraphNode(replyCompressionChainName, chain)
}

// Cinna chat node
// in: []*schema.Message | out: *schema.Message
func appendCinnaChatNode(chain ReplyCompressionChain, chatModel ChatModel, prompt string) {
	chain.AppendChatModel(
		chatModel,
		compose.WithNodeKey(cinnaReplyNodeName),
		compose.WithStatePreHandler(prepareForChat(prompt)),
	)
}

func prepareForChat(prompt string,
) compose.StatePreHandler[[]*schema.Message, *ReplyCompressionChainState] {
	return func(ctx context.Context,
		input []*schema.Message,
		state *ReplyCompressionChainState) ([]*schema.Message, error) {
		msgs := organizeInputMessageWithoutSensitiveInfo(input, prompt)
		state.History = msgs
		return msgs, nil
	}
}
