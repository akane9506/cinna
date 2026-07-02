package db

import (
	"context"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
)

type FeedbacksRepository struct {
	queries *sqlc.Queries
}

type FeedbackStatus string

const (
	StatusPending    FeedbackStatus = "pending"
	StatusInProgress FeedbackStatus = "in_progress"
	StatusCompleted  FeedbackStatus = "completed"
)

type FeedbackItem struct {
	Content string
	Status  FeedbackStatus
}

var _ ports.FeedbacksRepository = (*FeedbacksRepository)(nil)

func NewFeedbacksRepository(
	queries *sqlc.Queries) *FeedbacksRepository {
	return &FeedbacksRepository{
		queries: queries,
	}
}

// retrieves pending feedbacks from the database
func (f *FeedbacksRepository) ListPendingFeedbacks(
	ctx context.Context) ([]sqlc.ListPendingFeedbacksRow, error) {
	return f.queries.ListPendingFeedbacks(ctx)
}

// add feedback items to the database
func (f *FeedbacksRepository) CreateFeedbackItems(
	ctx context.Context,
	telegramUserID int64,
	contents []string,
) ([]sqlc.Feedback, error) {
	if len(contents) == 0 {
		return nil, nil
	}
	params := sqlc.CreateFeedbackItemsParams{
		TelegramUserID: telegramUserID,
		Contents:       contents,
	}
	return f.queries.CreateFeedbackItems(ctx, params)
}
