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
