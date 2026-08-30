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
) ([]sqlc.AgentMemory, error) {
	memory, err := a.queries.ListRecentAgentMemory(
		ctx,
		telegramUserID,
	)
	if err != nil {
		return nil, err
	}
	return a.decryptMessages(memory), nil
}

func (a *AgentMemoryRepository) AppendAgentMemoryBatch(
	ctx context.Context,
	params sqlc.AppendAgentMemoryBatchParams,
) ([]sqlc.AgentMemory, error) {
	validContents, validRoles := a.encryptMessages(
		params.TelegramUserID, params.Contents, params.Roles)
	encryptedParams := sqlc.AppendAgentMemoryBatchParams{
		TelegramUserID: params.TelegramUserID,
		Contents:       validContents,
		Roles:          validRoles,
	}
	return a.queries.AppendAgentMemoryBatch(ctx, encryptedParams)
}

func (a *AgentMemoryRepository) ReplaceAgentMemory(
	ctx context.Context,
	params sqlc.ReplaceAgentMemoryParams,
) ([]sqlc.AgentMemory, error) {
	validContents, validRoles := a.encryptMessages(
		params.TelegramUserID, params.Contents, params.Roles)
	encryptedParams := sqlc.ReplaceAgentMemoryParams{
		TelegramUserID: params.TelegramUserID,
		Contents:       validContents,
		Roles:          validRoles,
	}
	return a.queries.ReplaceAgentMemory(ctx, encryptedParams)
}

func (a *AgentMemoryRepository) encryptMessages(
	telegramUserID int64,
	contents []string,
	roles []string) ([]string, []string) {
	logger := a.logger.With("path", "internal/postgres/agent_memory/encryptMessages")
	validRoles := []string{}
	validContents := []string{}
	for idx, content := range contents {
		encryptedMessage, err := a.messageCipher.Encrypt(telegramUserID, content)
		if err != nil {
			// we don't break the progress when encryption failed, therefore we log errors without returning them
			logger.Error("failed to encrypt message", "error", err)
			continue
		}
		validContents = append(validContents, encryptedMessage)
		validRoles = append(validRoles, roles[idx])
	}
	return validContents, validRoles
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
