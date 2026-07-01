package db

import (
	"context"
	"log/slog"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/akane9506/cinna/internal/security"
)

type AgentMemoryRepository struct {
	queries       *sqlc.Queries
	messageCipher *security.MessageCipher
	logger        *slog.Logger
}

var _ ports.AgentMemoryRepository = (*AgentMemoryRepository)(nil)

func NewAgentMemoryRepository(
	queries *sqlc.Queries,
	messageCipher *security.MessageCipher,
	logger *slog.Logger,
) *AgentMemoryRepository {
	return &AgentMemoryRepository{
		queries:       queries,
		messageCipher: messageCipher,
		logger:        logger,
	}
}

func (a *AgentMemoryRepository) ListRecentAgentMemory(
	ctx context.Context,
	telegramUserID int64,
	maxHistoryLength int32,
) ([]sqlc.AgentMemory, error) {
	memory, err := a.queries.ListRecentAgentMemory(
		ctx,
		sqlc.ListRecentAgentMemoryParams{
			TelegramUserID: telegramUserID,
			MaxLength:      maxHistoryLength,
		})
	if err != nil {
		return nil, err
	}
	return a.decryptMessages(memory), nil
}

func (a *AgentMemoryRepository) AppendAgentMemoryBatch(
	ctx context.Context,
	params sqlc.AppendAgentMemoryBatchParams,
) ([]sqlc.AgentMemory, error) {
	encryptedParams := a.encryptMessages(&params)
	return a.queries.AppendAgentMemoryBatch(ctx, *encryptedParams)
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

func (a *AgentMemoryRepository) encryptMessages(params *sqlc.AppendAgentMemoryBatchParams) *sqlc.AppendAgentMemoryBatchParams {
	logger := a.logger.With("path", "internal/postgres/agent_memory/encryptMessages")
	telegramUserID := params.TelegramUserID
	validRoles := []string{}
	validContents := []string{}
	for idx, content := range params.Contents {
		encryptedMessage, err := a.messageCipher.Encrypt(telegramUserID, content)
		if err != nil {
			// we don't break the progress when encryption failed, therefore we log errors without returning them
			logger.Error("failed to encrypt message", "error", err)
			continue
		}
		validContents = append(validContents, encryptedMessage)
		validRoles = append(validRoles, params.Roles[idx])
	}
	params.Contents = validContents
	params.Roles = validRoles
	return params
}

func (a *AgentMemoryRepository) decryptMessages(memory []sqlc.AgentMemory) []sqlc.AgentMemory {
	logger := a.logger.With("path", "internal/postgres/agent_memory/decryptMessages")
	outputMemory := []sqlc.AgentMemory{}
	for _, mem := range memory {
		decryptedContent, err := a.messageCipher.Decrypt(mem.TelegramUserID, mem.Content)
		if err != nil {
			logger.Error("failed to decrypt message", "error", err)
			continue
		}
		newMem := mem
		newMem.Content = decryptedContent
		outputMemory = append(outputMemory, newMem)
	}
	return outputMemory
}
