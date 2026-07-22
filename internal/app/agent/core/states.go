package core

import "github.com/cloudwego/eino/schema"

// should find a better way to use it
type AgentState struct {
	TelegramUserID int64
	ChatIntent     Intent // for the general chat intention
	ActionType     Action // whether list the db items for update db
	History        []*schema.Message
	// SystemMessage  string // mainly for debugging purpose
	// maybe add token usage-related fields here
}
