package ports

import (
	"context"

	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type Repositories struct {
	AllowList    AllowListRepository
	ShoppingList ShoppingListRepository
	AgentMemory  AgentMemoryRepository
	Feedback     FeedbacksRepository
}

type AllowListRepository interface {
	IsAdminUser(ctx context.Context, telegramUserID int64) (bool, error)
	IsAllowedUser(ctx context.Context, telegramUserID int64) (bool, error)
	UpsertAllowedUser(
		ctx context.Context,
		telegramUserID int64) (sqlc.AllowedUser, error)
	SubscribeNotification(
		ctx context.Context,
		telegramUserID int64) (sqlc.AllowedUser, error)
	UnsubscribeNotification(
		ctx context.Context,
		telegramUserID int64) (sqlc.AllowedUser, error)
}

type ShoppingListRepository interface {
	CreateShoppingListItems(
		ctx context.Context,
		arg sqlc.CreateShoppingListItemsParams,
	) ([]sqlc.ShoppingList, error)

	RemoveShoppingListItems(
		ctx context.Context,
		arg sqlc.RemoveShoppingListItemsParams,
	) ([]sqlc.ShoppingList, error)

	UpdateShoppingListItems(
		ctx context.Context,
		arg sqlc.UpdateShoppingListItemsParams,
	) ([]sqlc.ShoppingList, error)

	ListShoppingListItems(
		ctx context.Context,
		telegramUserID int64,
	) ([]sqlc.ShoppingList, error)
}

type AgentMemoryRepository interface {
	ListRecentAgentMemory(
		ctx context.Context,
		telegramUserID int64,
		maxHistoryLength int32,
	) ([]sqlc.AgentMemory, error)

	AppendAgentMemoryBatch(
		ctx context.Context,
		params sqlc.AppendAgentMemoryBatchParams,
	) ([]sqlc.AgentMemory, error)

	ReplaceAgentMemory(
		ctx context.Context,
		params sqlc.ReplaceAgentMemoryParams,
	) ([]sqlc.AgentMemory, error)
}

type FeedbacksRepository interface {
	CreateFeedbackItems(
		ctx context.Context,
		telegramUserID int64,
		contents []string,
	) ([]sqlc.Feedback, error)

	ListIncompleteFeedbacks(
		ctx context.Context) ([]sqlc.ListIncompleteFeedbacksRow, error)
}
