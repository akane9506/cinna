package telegram

import "context"

type Handler interface {
	HandleText(
		ctx context.Context,
		userID int64,
		text string,
	) (string, error)
}
