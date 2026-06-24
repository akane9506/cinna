package db

import (
	"context"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type ShoppingListRepository struct {
	queries *sqlc.Queries
}

var _ ports.ShoppingListRepository = (*ShoppingListRepository)(nil)

func NewShoppingListRepository(queries *sqlc.Queries) *ShoppingListRepository {
	return &ShoppingListRepository{
		queries: queries,
	}
}

func (r *ShoppingListRepository) CreateShoppingListItems(
	ctx context.Context,
	arg sqlc.CreateShoppingListItemsParams) ([]sqlc.ShoppingList, error) {
	return r.queries.CreateShoppingListItems(ctx, arg)
}

func (r *ShoppingListRepository) ListShoppingListItems(
	ctx context.Context,
	telegramUserID int64) ([]sqlc.ShoppingList, error) {
	return r.queries.ListShoppingListItems(ctx, telegramUserID)
}
