package agent

import "github.com/cloudwego/eino/schema"

// should find a better way to use it
type HistoryMessage struct {
	SystemMessage string // mainly for debugging purpose
	History       []*schema.Message
	// maybe add token usage-related fields here
}
