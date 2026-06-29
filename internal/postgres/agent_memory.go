package db

import (
	"context"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type AgentMemoryRepository struct {
	queries *sqlc.Queries
}

var _ ports.AgentMemoryRepository = (*AgentMemoryRepository)(nil)

func NewAgentMemoryRepository(queries *sqlc.Queries) *AgentMemoryRepository {
	return &AgentMemoryRepository{
		queries: queries,
	}
}

func (a *AgentMemoryRepository) ListRecentAgentMemory(
	ctx context.Context,
	telegramUserID int64,
	maxHistoryLength int32,
) ([]sqlc.AgentMemory, error) {
	return a.queries.ListRecentAgentMemory(
		ctx,
		sqlc.ListRecentAgentMemoryParams{
			TelegramUserID: telegramUserID,
			MaxLength:      maxHistoryLength,
		})
}

func (a *AgentMemoryRepository) AppendAgentMemoryBatch(
	ctx context.Context,
	telegramUserID int64,
	roles []string,
	contents []string,
) ([]sqlc.AgentMemory, error) {
	return a.queries.AppendAgentMemoryBatch(
		ctx,
		sqlc.AppendAgentMemoryBatchParams{
			TelegramUserID: telegramUserID,
			Roles:          roles,
			Contents:       contents,
		})
}

func (a *AgentMemoryRepository) PruneAgentMemory(
	ctx context.Context,
	telegramUserID int64,
	keepCount int32,
) error {
	return a.queries.PruneAgentMemory(
		ctx,
		sqlc.PruneAgentMemoryParams{
			TelegramUserID: telegramUserID,
			KeepCount:      keepCount,
		})
}
