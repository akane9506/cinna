package agent

import (
	"github.com/akane9506/cinna/internal/app/agent/core"
	"github.com/cloudwego/eino/schema"
)

// should find a better way to use it
type CinnaAgentState struct {
	TelegramUserID int64
	SystemMessage  string      // mainly for debugging purpose
	ChatIntent     core.Intent // for the general chat intention
	ActionType     core.Action // whether list the db items for update db
	History        []*schema.Message
	// maybe add token usage-related fields here
}
