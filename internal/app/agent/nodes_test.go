package agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/cloudwego/eino/schema"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type mockShoppingListRepo struct {
	mock.Mock
}

func (m *mockShoppingListRepo) CreateShoppingListItems(
	ctx context.Context,
	arg sqlc.CreateShoppingListItemsParams,
) ([]sqlc.ShoppingList, error) {
	args := m.Called(ctx, arg)

	items, _ := args.Get(0).([]sqlc.ShoppingList)
	return items, args.Error(1)
}

func (m *mockShoppingListRepo) ListShoppingListItems(
	ctx context.Context,
	telegramUserID int64,
) ([]sqlc.ShoppingList, error) {
	args := m.Called(ctx, telegramUserID)

	items, _ := args.Get(0).([]sqlc.ShoppingList)
	return items, args.Error(1)
}

func getUnexpiredItems() []sqlc.ShoppingList {
	now := time.Now()
	var telegramUserID int64 = 123321
	return []sqlc.ShoppingList{
		{
			ID:             1,
			TelegramUserID: telegramUserID,
			Name:           "milk",
			Category:       "grocery",
			UpdatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		},
		{
			ID:             2,
			TelegramUserID: telegramUserID,
			Name:           "bandages",
			Category:       "pharmacy",
			UpdatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		},
		{
			ID:             3,
			TelegramUserID: telegramUserID,
			Name:           "cat food",
			Category:       "pet_store",
			UpdatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		},
	}
}

func getExpiredItems() []sqlc.ShoppingList {
	now := time.Now().AddDate(0, -2, 0)
	var telegramUserID int64 = 123321
	return []sqlc.ShoppingList{
		{
			ID:             4,
			TelegramUserID: telegramUserID,
			Name:           "lego set",
			Category:       "toy_shop",
			UpdatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		},
		{
			ID:             5,
			TelegramUserID: telegramUserID,
			Name:           "notebook",
			Category:       "stationery",
			UpdatedAt: pgtype.Timestamptz{
				Time:  now,
				Valid: true,
			},
		},
	}
}

func getFullList() []sqlc.ShoppingList {
	expired := getExpiredItems()
	unexpired := getUnexpiredItems()
	var fullList []sqlc.ShoppingList
	fullList = append(fullList, expired...)
	fullList = append(fullList, unexpired...)
	return fullList
}

// Tests
func TestListShoppingListItems(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	shoppingListRepo := new(mockShoppingListRepo)
	mockRepos := &ports.Repositories{
		ShoppingList: shoppingListRepo,
	}
	ctx := context.Background()
	testAgent := &CinnaReactAgent{
		repos:  mockRepos,
		logger: logger,
	}
	tests := []struct {
		name         string
		mockDBOutput []sqlc.ShoppingList
		error        error
		hasUnexpired bool
		hasExpired   bool
	}{
		{
			"success", getFullList(), nil, true, true,
		},
		{
			"success with expired items", getExpiredItems(), nil, false, true,
		},
		{
			"success with unexpired items", getUnexpiredItems(), nil, true, false,
		},
		{
			"failed", nil, errors.New("Failed"), false, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockState := &CinnaAgentState{
				History: make([]*schema.Message, 4),
			}
			shoppingListRepo.On(
				"ListShoppingListItems",
				mock.Anything,
				mock.Anything,
			).Return(tt.mockDBOutput, tt.error).Once()
			msg := testAgent.listShoppingListItems(ctx, mockState)
			t.Log(msg)
			assert.NotNil(t, msg)
			assert.Equal(t, msg.Role, schema.Assistant)
			if tt.error != nil {
				assert.Contains(t, msg.Content, "failed to get current list from database")
			} else {
				if !tt.hasExpired {
					assert.Contains(t, msg.Content, "expiredItems: []")
				}
				if !tt.hasUnexpired {
					assert.Contains(t, msg.Content, "activeItems: []")
				}
			}
			shoppingListRepo.AssertExpectations(t)
		})
	}
}
