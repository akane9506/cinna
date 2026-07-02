package agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockFeedbacksRepo struct {
	mock.Mock
}

func (m *mockFeedbacksRepo) ListIncompleteFeedbacks(
	ctx context.Context) ([]sqlc.ListIncompleteFeedbacksRow, error) {
	args := m.Called(ctx)
	return args.Get(0).([]sqlc.ListIncompleteFeedbacksRow), args.Error(1)
}

func (m *mockFeedbacksRepo) CreateFeedbackItems(
	ctx context.Context, telegramUserID int64, contexts []string,
) ([]sqlc.Feedback, error) {
	args := m.Called(ctx, telegramUserID, contexts)
	return args.Get(0).([]sqlc.Feedback), args.Error(1)
}

func TestProcessFeedbackListLambda(t *testing.T) {
	feedbacksRepo := new(mockFeedbacksRepo)
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockAgentState := &CinnaAgentState{TelegramUserID: 123321}
	mockCinnaAgent := &CinnaReactAgent{
		logger: mockLogger,
		repos: &ports.Repositories{
			Feedback: feedbacksRepo,
		},
	}
	tests := []struct {
		name      string
		mockLists []sqlc.ListIncompleteFeedbacksRow
		mockError error
	}{
		{
			name: "success",
			mockLists: []sqlc.ListIncompleteFeedbacksRow{
				{ID: 1, Content: "feedback item 1"},
				{ID: 2, Content: "feedback item 2"},
				{ID: 3, Content: "feedback item 3"},
			},
		},
		{
			name:      "failed",
			mockLists: []sqlc.ListIncompleteFeedbacksRow{},
			mockError: errors.New("failed"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			feedbacksRepo.On("ListIncompleteFeedbacks", mock.Anything).
				Return(tt.mockLists, tt.mockError).Once()
			ctx := context.Background()
			message := mockCinnaAgent.listFeedbackItems(ctx, mockAgentState)
			if tt.mockError == nil {
				assert.Contains(t, message.Content, "feedbacks in the current database")
				for _, item := range tt.mockLists {
					assert.Contains(t, message.Content, item.Content)
				}
			} else {
				assert.Contains(t, message.Content, "failed to get feedbacks from database")
			}
		})
	}
}
