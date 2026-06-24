package ports

import (
	"context"

	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type Repositories struct {
	AllowList    AllowListRepository
	ShoppingList ShoppingListRepository
}

type AllowListRepository interface {
	IsAllowedUser(ctx context.Context, telegramUserID int64) (bool, error)
}

type ShoppingListRepository interface {
	CreateShoppingListItems(
		ctx context.Context,
		arg sqlc.CreateShoppingListItemsParams,
	) ([]sqlc.ShoppingList, error)

	ListShoppingListItems(
		ctx context.Context,
		telegramUserID int64,
	) ([]sqlc.ShoppingList, error)
}
