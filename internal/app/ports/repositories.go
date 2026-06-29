package ports

import (
	"context"

	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type Repositories struct {
	AllowList    AllowListRepository
	ShoppingList ShoppingListRepository
	AgentMemory  AgentMemoryRepository
}

type AllowListRepository interface {
	IsAllowedUser(ctx context.Context, telegramUserID int64) (bool, error)
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
		telegramUserID int64,
		roles []string,
		contents []string,
	) ([]sqlc.AgentMemory, error)

	PruneAgentMemory(
		ctx context.Context,
		telegramUserID int64,
		keepCount int32,
	) error
}
