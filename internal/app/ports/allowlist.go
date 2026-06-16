package ports

import "context"

type AllowListRepository interface {
	IsAllowedUser(ctx context.Context, telegramUserID int64) (bool, error)
}
