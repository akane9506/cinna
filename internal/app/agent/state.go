package agent

import "github.com/cloudwego/eino/schema"

// should find a better way to use it
type HistoryMessage struct {
	SystemMessage string // mainly for debugging purpose
	ChatIntent    string // for the general chat intention
	ActionType    string // whether list the db items for update db
	History       []*schema.Message
	// maybe add token usage-related fields here
}
