package memory

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/cloudwego/eino/schema"
)

const (
	// number of history messages for conversation,  including assistant and user msg, should be enough
	defaultMaxChatLength = 30

	// number of history stored in the database
	defaultMaxDBLength = 100

	// flush buffer to the history db when buffer size equals or higher than this threshold
	defaultFlushBufferThreshold = 6
)

type MemoryStore struct {
	mu                   sync.RWMutex // protects the user map only
	maxChatLength        int
	maxDBLength          int
	flushBufferThreshold int

	users       map[int64]*userState
	agentMemory ports.AgentMemoryRepository

	logger *slog.Logger
}

type userState struct {
	mu      sync.Mutex // protects user's buffer and history read/write
	history []schema.Message
	buffer  []schema.Message
	loaded  bool
}

// create a memory store
func NewMemoryStore(agentMemoryRepo ports.AgentMemoryRepository, logger *slog.Logger) *MemoryStore {
	store := &MemoryStore{
		users:                map[int64]*userState{},
		maxChatLength:        defaultMaxChatLength,
		maxDBLength:          defaultMaxDBLength,
		flushBufferThreshold: defaultFlushBufferThreshold,
		agentMemory:          agentMemoryRepo,
		logger:               logger,
	}
	return store
}

func (m *MemoryStore) Append(ctx context.Context, userID int64, msg *schema.Message) {
	// we don't store system message.
	// because system message will be inserted into the chat right before invoking the chat model
	if msg == nil || msg.Role == schema.System {
		return
	}
	userState := m.getUserState(userID)
	userState.mu.Lock()
	defer userState.mu.Unlock()
	// exclude reasoning and other token-consuming information
	cleanMessage := schema.Message{Role: msg.Role, Content: msg.Content}
	history := append(userState.history, cleanMessage)
	userState.buffer = append(userState.buffer, cleanMessage)
	if len(history) > m.maxChatLength {
		history = history[len(history)-m.maxChatLength:]
	}
	// check buffer size and decide whether to update db or not
	if m.shouldFlushBuffer(userState) {
		if err := m.flushBuffer(ctx, userID, userState); err != nil {
			m.logger.Error("error flush buffer history to DB",
				"telegram_user_id", userID,
				"error", err,
			)
		}
	}
	userState.history = history
}

func (m *MemoryStore) Get(ctx context.Context, userID int64) []*schema.Message {
	userState := m.getUserState(userID)
	userState.mu.Lock()
	defer userState.mu.Unlock()
	if !userState.loaded {
		history, err := m.loadHistoryFromDB(ctx, userID)
		if err != nil {
			m.logger.Error("failed to load chat history from db",
				"telegram_user_id", userID, "error", err)
		} else {
			userState.history = history
		}
		// we let it loaded even when there's an error, to avoid keep calling DB
		userState.loaded = true
	}
	output := make([]*schema.Message, 0, len(userState.history))
	for _, msg := range userState.history {
		output = append(output, &msg)
	}
	return output
}

// ========== user state map operations ==========

func (m *MemoryStore) getUserState(userID int64) *userState {
	m.mu.Lock()
	defer m.mu.Unlock()
	state := m.users[userID]
	if state == nil {
		state = &userState{}
		m.users[userID] = state
	}
	return state
}

// ========== DB memory operations ==========
func (m *MemoryStore) shouldFlushBuffer(userState *userState) bool {
	return len(userState.buffer) >= m.flushBufferThreshold
}

func (m *MemoryStore) flushBuffer(ctx context.Context, userID int64, userState *userState) error {
	roles := []string{}
	contents := []string{}
	for _, message := range userState.buffer {
		role := message.Role
		if role != schema.Assistant && role != schema.User {
			continue
		}
		roles = append(roles, string(role))
		contents = append(contents, message.Content)
	}
	_, err := m.agentMemory.AppendAgentMemoryBatch(ctx, sqlc.AppendAgentMemoryBatchParams{
		TelegramUserID: userID,
		Roles:          roles,
		Contents:       contents,
	})
	if err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}
	// if flush succeeded, clear the buffer
	userState.buffer = []schema.Message{}
	// and prune the agent memory size
	err = m.agentMemory.PruneAgentMemory(ctx, userID, int32(m.maxDBLength))
	if err != nil {
		return fmt.Errorf("failed to prune history: %w", err)
	}
	return nil
}

func (m *MemoryStore) loadHistoryFromDB(ctx context.Context, userID int64) ([]schema.Message, error) {
	parsedHistory := []schema.Message{}
	logger := m.logger.With("path", "internal/app/agent/memory/state/loadHistoryFromDB")
	dbHistory, err := m.agentMemory.ListRecentAgentMemory(ctx, userID, int32(m.maxChatLength))
	if err != nil {
		return parsedHistory, err
	}
	for _, message := range dbHistory {
		if message.TelegramUserID != userID {
			logger.Warn(
				"wrong message fetched",
				"user_id", userID,
				"fetched_user_id", message.TelegramUserID,
			)
			continue
		}
		switch schema.RoleType(message.Role) {
		case schema.Assistant:
			parsedHistory = append(parsedHistory,
				schema.Message{Role: schema.Assistant, Content: message.Content})
		case schema.User:
			parsedHistory = append(parsedHistory,
				schema.Message{Role: schema.User, Content: message.Content})
		default:
			logger.Warn("wrong message role detected", "role", string(message.Role))
		}
	}
	return parsedHistory, nil
}
