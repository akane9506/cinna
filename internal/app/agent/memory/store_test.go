package memory

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const mockTelegramUserID int64 = 1234567

type mockAgentMemoryRepo struct {
	mock.Mock
}

func (m *mockAgentMemoryRepo) ListRecentAgentMemory(
	ctx context.Context,
	telegramUserID int64,
	maxHistoryLength int32,
) ([]sqlc.AgentMemory, error) {
	args := m.Called(ctx, telegramUserID, maxHistoryLength)
	return args.Get(0).([]sqlc.AgentMemory), args.Error(1)
}

func (m *mockAgentMemoryRepo) AppendAgentMemoryBatch(
	ctx context.Context,
	params sqlc.AppendAgentMemoryBatchParams,
) ([]sqlc.AgentMemory, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]sqlc.AgentMemory), args.Error(1)
}

func (m *mockAgentMemoryRepo) PruneAgentMemory(
	ctx context.Context,
	telegramUserID int64,
	keepCount int32) error {
	args := m.Called(ctx, telegramUserID, keepCount)
	return args.Error(0)
}

func TestShouldFlushBuffer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		bufferSize int
		expected   bool
	}{
		{"no buffer", 0, false},
		{"less buffer", defaultFlushBufferThreshold - 1, false},
		{"buffer threshold", defaultFlushBufferThreshold, true},
	}
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewMemoryStore(&mockAgentMemoryRepo{}, mockLogger)
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			userState := userState{
				buffer: make([]schema.Message, tt.bufferSize),
			}
			output := store.shouldFlushBuffer(&userState)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestFlushBuffer(t *testing.T) {
	t.Parallel()
	agentMemoryRepo := new(mockAgentMemoryRepo)
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewMemoryStore(agentMemoryRepo, mockLogger)
	tests := []struct {
		name               string
		buffer             []schema.Message
		inputRoles         []string
		inputContents      []string
		expectedFlushError error
		expectedPruneError error
	}{
		{
			name: "success",
			buffer: []schema.Message{
				schema.Message{Role: schema.User, Content: "user"},
				schema.Message{Role: schema.Assistant, Content: "assistant"},
			},
			inputRoles:         []string{string(schema.User), string(schema.Assistant)},
			inputContents:      []string{"user", "assistant"},
			expectedFlushError: nil,
		},
		{
			name: "success with only necessary messages",
			buffer: []schema.Message{
				schema.Message{Role: schema.User, Content: "user"},
				schema.Message{Role: schema.Assistant, Content: "assistant"},
				schema.Message{Role: schema.System, Content: "system"},
			},
			inputRoles:         []string{string(schema.User), string(schema.Assistant)},
			inputContents:      []string{"user", "assistant"},
			expectedFlushError: nil,
		},
		{
			name: "error flushing to db",
			buffer: []schema.Message{
				schema.Message{Role: schema.User, Content: "user"},
				schema.Message{Role: schema.Assistant, Content: "assistant"},
			},
			inputRoles:         []string{string(schema.User), string(schema.Assistant)},
			inputContents:      []string{"user", "assistant"},
			expectedFlushError: errors.New("failed to flush"),
		},
		{
			name: "error prune db",
			buffer: []schema.Message{
				schema.Message{Role: schema.User, Content: "user"},
				schema.Message{Role: schema.Assistant, Content: "assistant"},
			},
			inputRoles:         []string{string(schema.User), string(schema.Assistant)},
			inputContents:      []string{"user", "assistant"},
			expectedFlushError: nil,
			expectedPruneError: errors.New("failed to prune"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			state := &userState{
				buffer: tt.buffer,
			}
			agentMemoryRepo.On("AppendAgentMemoryBatch",
				mock.Anything,
				mock.MatchedBy(func(params sqlc.AppendAgentMemoryBatchParams) bool {
					return slices.Equal(params.Contents, tt.inputContents) &&
						slices.Equal(params.Roles, tt.inputRoles)
				}),
			).Return([]sqlc.AgentMemory{}, tt.expectedFlushError).Once()
			if tt.expectedFlushError == nil {
				agentMemoryRepo.On("PruneAgentMemory",
					mock.Anything,
					mock.AnythingOfType("int64"),
					mock.AnythingOfType("int32"),
				).Return(tt.expectedPruneError).Once()
			}
			err := store.flushBuffer(context.Background(), mockTelegramUserID, state)
			if tt.expectedFlushError == nil {
				assert.Equal(t, len(state.buffer), 0)
			} else {
				assert.NotEqual(t, len(state.buffer), 0)
			}
			if tt.expectedPruneError != nil || tt.expectedFlushError != nil {
				assert.NotNil(t, err)
			}
			if tt.expectedFlushError == nil && tt.expectedPruneError == nil {
				assert.Nil(t, err)
			}
		})
	}
}

func TestLoadHistoryFromDB(t *testing.T) {
	t.Parallel()
	agentMemoryRepo := new(mockAgentMemoryRepo)
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := NewMemoryStore(agentMemoryRepo, mockLogger)
	tests := []struct {
		name                string
		mockMemory          []sqlc.AgentMemory
		expectedOutput      []schema.Message
		mockListMemoryError error
	}{
		{
			name: "success",
			mockMemory: []sqlc.AgentMemory{
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.Assistant),
					Content:        "assistant",
				},
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.User),
					Content:        "user",
				},
			},
			expectedOutput: []schema.Message{
				{
					Role:    schema.Assistant,
					Content: "assistant",
				},
				{
					Role:    schema.User,
					Content: "user",
				},
			},
		},
		{
			name: "failed with wrong id",
			mockMemory: []sqlc.AgentMemory{
				{
					TelegramUserID: 66666,
					Role:           string(schema.Assistant),
					Content:        "assistant",
				},
				{
					TelegramUserID: 66666,
					Role:           string(schema.User),
					Content:        "user",
				},
			},
			expectedOutput: []schema.Message{},
		},
		{
			name: "excludes incompatible message",
			mockMemory: []sqlc.AgentMemory{
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.Assistant),
					Content:        "assistant",
				},
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.System),
					Content:        "system",
				},
			},
			expectedOutput: []schema.Message{
				{
					Role:    schema.Assistant,
					Content: "assistant",
				},
			},
		},
		{
			name:                "failed with error",
			mockMemory:          []sqlc.AgentMemory{},
			expectedOutput:      []schema.Message{},
			mockListMemoryError: errors.New("failed to get list"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			agentMemoryRepo.On("ListRecentAgentMemory",
				mock.Anything,
				mock.AnythingOfType("int64"),
				mock.AnythingOfType("int32"),
			).Return(tt.mockMemory, tt.mockListMemoryError).Once()
			output, err := store.loadHistoryFromDB(context.Background(), mockTelegramUserID)
			assert.Equal(t, tt.expectedOutput, output)
			if tt.mockListMemoryError != nil {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestAppend(t *testing.T) {
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMemoryRepo := new(mockAgentMemoryRepo)
	// t.Parallel()
	tests := []struct {
		name            string
		msg             *schema.Message
		flushBuffer     bool
		exceedMaxLength bool
		invalidMessage  bool
	}{
		{
			name: "without flush buffer",
			msg:  &schema.Message{Role: schema.Assistant, Content: "assistant", ToolCallID: "123"},
		},
		{
			name:            "with exceeded chat length",
			msg:             &schema.Message{Role: schema.Assistant, Content: "assistant"},
			exceedMaxLength: true,
		},
		{
			name:        "with flush buffer",
			msg:         &schema.Message{Role: schema.Assistant, Content: "assistant"},
			flushBuffer: true,
		},
		{
			name:           "with invalid message type",
			msg:            &schema.Message{Role: schema.System, Content: "system"},
			invalidMessage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore(mockMemoryRepo, mockLogger)
			store.users[mockTelegramUserID] = &userState{}
			ctx := context.Background()
			if tt.exceedMaxLength {
				store.users[mockTelegramUserID].history = make([]schema.Message, defaultMaxChatLength+4)
				assert.Equal(t, len(store.users[mockTelegramUserID].history), defaultMaxChatLength+4)
			}
			if tt.flushBuffer {
				store.users[mockTelegramUserID].buffer = make([]schema.Message, defaultFlushBufferThreshold+1)
				assert.Equal(t, len(store.users[mockTelegramUserID].buffer), defaultFlushBufferThreshold+1)
			}
			if tt.flushBuffer {
				mockMemoryRepo.On("AppendAgentMemoryBatch",
					mock.Anything,
					mock.Anything,
				).Return([]sqlc.AgentMemory{}, nil).Once()
				mockMemoryRepo.On("PruneAgentMemory",
					mock.Anything,
					mock.AnythingOfType("int64"),
					mock.AnythingOfType("int32"),
				).Return(nil).Once()
			}
			store.Append(ctx, mockTelegramUserID, tt.msg)
			userState := store.users[mockTelegramUserID]
			assert.NotNil(t, userState)
			if !tt.invalidMessage {
				lastMsg := userState.history[len(userState.history)-1]
				assert.Equal(t, lastMsg.Content, tt.msg.Content)
				if tt.msg.ToolCallID != "" {
					assert.NotEqual(t, lastMsg.ToolCallID, tt.msg.ToolCallID)
				}
			} else {
				assert.Equal(t, len(userState.history), 0)
			}
			assert.LessOrEqual(t, len(userState.history), defaultMaxChatLength)
			assert.Less(t, len(userState.buffer), defaultFlushBufferThreshold)
		})
	}
}

func TestGet(t *testing.T) {
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockMemoryRepo := new(mockAgentMemoryRepo)
	tests := []struct {
		name          string
		loaded        bool
		localHistory  []schema.Message
		remoteHistory []sqlc.AgentMemory
	}{
		{
			name:         "get local history",
			loaded:       true,
			localHistory: []schema.Message{{Role: schema.User, Content: "hello"}},
		},
		{
			name:         "get remote history",
			loaded:       false,
			localHistory: []schema.Message{{Role: schema.User, Content: "hello"}},
			remoteHistory: []sqlc.AgentMemory{
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.Assistant),
					Content:        "assistant",
				},
				{
					TelegramUserID: mockTelegramUserID,
					Role:           string(schema.User),
					Content:        "user",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			store := NewMemoryStore(mockMemoryRepo, mockLogger)
			userState := &userState{}
			store.users[mockTelegramUserID] = userState
			ctx := context.Background()
			if tt.loaded {
				userState.loaded = true
				userState.history = tt.localHistory
			} else {
				mockMemoryRepo.On("ListRecentAgentMemory",
					mock.Anything,
					mock.AnythingOfType("int64"),
					mock.AnythingOfType("int32"),
				).Return(tt.remoteHistory, nil).Once()
			}
			result := store.Get(ctx, mockTelegramUserID)
			for idx, msg := range result {
				if tt.loaded {
					assert.Equal(t, tt.localHistory[idx], *msg)
				} else {
					assert.Equal(t, tt.remoteHistory[idx].Role, string(msg.Role))
					assert.Equal(t, tt.remoteHistory[idx].Content, string(msg.Content))
				}
			}
		})
	}
}
