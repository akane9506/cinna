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
	// flush buffer to the history db when buffer size equals or higher than this threshold
	defaultFlushBufferThreshold = 2 // we can increase this when having a constant-running instance
)

type MemoryStore struct {
	mu                   sync.RWMutex // protects the user map only
	flushBufferThreshold int

	users       map[int64]*userState
	agentMemory ports.AgentMemoryRepository

	logger *slog.Logger
}

type userState struct {
	mu        sync.Mutex // protects user's buffer and history read/write
	requestMu sync.Mutex // serializes the user's requests - avoid concurrent msgs ruin the history

	history []schema.Message
	buffer  []schema.Message
	loaded  bool
}

// create a memory store
func NewMemoryStore(agentMemoryRepo ports.AgentMemoryRepository, logger *slog.Logger) *MemoryStore {
	store := &MemoryStore{
		users:                map[int64]*userState{},
		flushBufferThreshold: defaultFlushBufferThreshold,
		agentMemory:          agentMemoryRepo,
		logger:               logger,
	}
	return store
}

func (m *MemoryStore) UpdateChatHistory(ctx context.Context, userID int64, msgs []*schema.Message) {
	// we don't store system message.
	// because system message will be inserted into the chat right before invoking the chat model
	if msgs == nil || len(msgs) == 0 {
		return
	}
	userState := m.getUserState(userID)
	userState.mu.Lock()
	defer userState.mu.Unlock()
	numNewMsgs := len(msgs) - len(userState.history)
	if numNewMsgs == 0 {
		return
	}
	newMsgs := msgs[len(msgs)-numNewMsgs:]
	history := userState.history
	buffer := userState.buffer
	for _, msg := range newMsgs {
		// currently let's allow the memory to grow. The new history compression method
		// will be proposed in the next step
		history = append(history, *msg)
		buffer = append(buffer, *msg)
	}
	userState.history = history
	userState.buffer = buffer
	if m.shouldFlushBuffer(userState) {
		if err := m.flushBuffer(ctx, userID, userState); err != nil {
			m.logger.Error("error flush buffer history to DB",
				"telegram_user_id", userID,
				"error", err,
			)
		}
	}
}

func (m *MemoryStore) ReplaceChatHistory(
	ctx context.Context, userID int64, msgs []*schema.Message) error {
	if msgs == nil || len(msgs) == 0 {
		return fmt.Errorf("empty message received")
	}
	userState := m.getUserState(userID)
	userState.mu.Lock()
	defer userState.mu.Unlock()
	history := []schema.Message{}
	roles := []string{}
	contents := []string{}
	for _, msg := range msgs {
		// currently let's allow the memory to grow. The new history compression method
		// will be proposed in the next step
		history = append(history, *msg)
		roles = append(roles, string(msg.Role))
		contents = append(contents, msg.Content)
	}
	replaceAgentMemoryParams := sqlc.ReplaceAgentMemoryParams{
		TelegramUserID: userID,
		Roles:          roles,
		Contents:       contents,
	}
	_, err := m.agentMemory.ReplaceAgentMemory(ctx, replaceAgentMemoryParams)
	if err != nil {
		m.logger.Error(
			"failed to replace agent memory",
			"error", err)
		return err
	}
	userState.history = history
	userState.buffer = []schema.Message{}
	return nil
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

func (m *MemoryStore) LockUserRequest(userID int64) func() {
	state := m.getUserState(userID)
	state.requestMu.Lock()
	return state.requestMu.Unlock
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
	return nil
}

func (m *MemoryStore) loadHistoryFromDB(ctx context.Context, userID int64) ([]schema.Message, error) {
	parsedHistory := []schema.Message{}
	logger := m.logger.With("path", "internal/app/agent/memory/state/loadHistoryFromDB")
	dbHistory, err := m.agentMemory.ListRecentAgentMemory(ctx, userID)
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
