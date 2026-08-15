package db

import (
	"context"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type AllowListRepository struct {
	queries *sqlc.Queries
}

var _ ports.AllowListRepository = (*AllowListRepository)(nil)

func NewAllowListRepository(queries *sqlc.Queries) *AllowListRepository {
	return &AllowListRepository{
		queries: queries,
	}
}

func (r *AllowListRepository) IsAllowedUser(
	ctx context.Context,
	telegramUserID int64) (bool, error) {
	return r.queries.IsAllowedUser(ctx, telegramUserID)
}

func (r *AllowListRepository) IsAdminUser(
	ctx context.Context, telegramUserID int64) (bool, error) {
	return r.queries.IsAdminUser(ctx, telegramUserID)
}

func (r *AllowListRepository) UpsertAllowedUser(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	return r.queries.UpsertAllowedUser(ctx, sqlc.UpsertAllowedUserParams{
		TelegramUserID: telegramUserID,
		Role:           "member", // any user added through this method is default to be member
	})
}

func (r *AllowListRepository) SubscribeNotification(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	return r.queries.SubscribeNotification(ctx, telegramUserID)
}

func (r *AllowListRepository) UnsubscribeNotification(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	return r.queries.UnsubscribeNotification(ctx, telegramUserID)
}

func (r *AllowListRepository) DailyNotificationSubscribers(
	ctx context.Context) ([]int64, error) {
	return r.queries.DailyNotificationSubscribers(ctx)
}
