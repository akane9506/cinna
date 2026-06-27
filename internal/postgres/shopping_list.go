package db

import (
	"context"
	"time"

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

func (r *ShoppingListRepository) UpdateShoppingListItems(
	ctx context.Context,
	arg sqlc.UpdateShoppingListItemsParams,
) ([]sqlc.ShoppingList, error) {
	return r.queries.UpdateShoppingListItems(ctx, arg)
}

func (r *ShoppingListRepository) RemoveShoppingListItems(
	ctx context.Context,
	arg sqlc.RemoveShoppingListItemsParams,
) ([]sqlc.ShoppingList, error) {
	return r.queries.RemoveShoppingListItems(ctx, arg)
}

func (r *ShoppingListRepository) ListShoppingListItems(
	ctx context.Context,
	telegramUserID int64) ([]sqlc.ShoppingList, error) {
	return r.queries.ListShoppingListItems(ctx, telegramUserID)
}

func IsExpired(item *sqlc.ShoppingList) bool {
	// the current cutoff is one month
	cutoff := time.Now().AddDate(0, -1, 0)
	if item.UpdatedAt.Time.Before(cutoff) {
		return true
	}
	return false
}
