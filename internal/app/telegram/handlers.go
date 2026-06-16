package telegram

import (
	"context"
)

type Handler interface {
	HandleText(
		ctx context.Context,
		userID int64,
		text string,
	) (string, error)
}

// Echo handler (for test only)

type EchoHandler struct{}

func NewEchoHandler() *EchoHandler {
	return &EchoHandler{}
}

func (h *EchoHandler) HandleText(
	ctx context.Context,
	userID int64,
	text string,
) (string, error) {
	return text, nil
}
