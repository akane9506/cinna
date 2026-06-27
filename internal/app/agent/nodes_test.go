package agent

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
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
	var items []sqlc.ShoppingList
	if fn, ok := args.Get(0).(func(context.Context, sqlc.CreateShoppingListItemsParams) []sqlc.ShoppingList); ok {
		items = fn(ctx, arg)
	} else if args.Get(0) != nil {
		items = args.Get(0).([]sqlc.ShoppingList)
	}
	var err error
	if fn, ok := args.Get(1).(func(context.Context, sqlc.CreateShoppingListItemsParams) error); ok {
		err = fn(ctx, arg)
	} else {
		err = args.Error(1)
	}
	return items, err
}

func (m *mockShoppingListRepo) UpdateShoppingListItems(
	ctx context.Context,
	arg sqlc.UpdateShoppingListItemsParams,
) ([]sqlc.ShoppingList, error) {
	args := m.Called(ctx, arg)
	items, _ := args.Get(0).([]sqlc.ShoppingList)
	return items, args.Error(1)
}

func (m *mockShoppingListRepo) RemoveShoppingListItems(
	ctx context.Context,
	arg sqlc.RemoveShoppingListItemsParams,
) ([]sqlc.ShoppingList, error) {
	args := m.Called(ctx, arg)
	var items []sqlc.ShoppingList
	if fn, ok := args.Get(0).(func(context.Context, sqlc.RemoveShoppingListItemsParams) []sqlc.ShoppingList); ok {
		items = fn(ctx, arg)
	} else if args.Get(0) != nil {
		items = args.Get(0).([]sqlc.ShoppingList)
	}
	var err error
	if fn, ok := args.Get(1).(func(context.Context, sqlc.RemoveShoppingListItemsParams) error); ok {
		err = fn(ctx, arg)
	} else {
		err = args.Error(1)
	}
	return items, err
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

func TestParseShoppingListCommands(t *testing.T) {
	t.Parallel()
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	testAgent := &CinnaReactAgent{
		logger: mockLogger,
	}
	tests := []struct {
		name     string
		raw      string
		expected ShoppingListCommands
	}{
		{
			name: "success",
			raw: `{"commands": [{"id": "1","method": "REMOVE","category": "grocery","name": "milk"},
			{"id": "2","method": "REMOVE","category": "grocery","name": "banana"},
    {"id": "","method": "ADD","category": "stationery","name": "一盒彩色马克笔(marker)"},
    {"id": "5","method": "MODIFY","category": "stationery","name": "A4 notebook"}
  ]}`,
			expected: ShoppingListCommands{
				MethodRemove: {
					ItemIds:    []int64{1, 2},
					Categories: []string{"grocery", "grocery"},
					ItemNames:  []string{"milk", "banana"}},
				MethodAdd: {
					ItemIds:    nil,
					Categories: []string{"stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)"}},
				MethodModify: {
					ItemIds:    []int64{5},
					Categories: []string{"stationery"},
					ItemNames:  []string{"A4 notebook"}},
			},
		},
		{
			name: "with invalid method",
			raw: `{"commands": [{"id": "-1","method": " REMOVE ","category": "grocery","name": "milk"},
			{"id": "2","method": "REMOVAL","category": "grocery","name": "banana"},
    {"id": "","method": "ADD","category": "stationery","name": "一盒彩色马克笔(marker)"},
    {"id": "5","method": "MODIFY","category": "stationery","name": "A4 notebook"}
  ]}`,
			expected: ShoppingListCommands{
				MethodAdd: {
					ItemIds:    nil,
					Categories: []string{"stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)"}},
				MethodModify: {
					ItemIds:    []int64{5},
					Categories: []string{"stationery"},
					ItemNames:  []string{"A4 notebook"}},
			},
		},
		{
			name:     "invalid json",
			raw:      `some invalid string`,
			expected: ShoppingListCommands{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testAgent.parseShoppingListCommands(tt.raw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecuteShoppingListCommand(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	shoppingListRepo := new(mockShoppingListRepo)
	mockRepos := &ports.Repositories{
		ShoppingList: shoppingListRepo,
	}
	var telegramUserID int64 = 123321
	testAgent := &CinnaReactAgent{
		repos:  mockRepos,
		logger: mockLogger,
	}
	tests := []struct {
		name               string
		input              ShoppingListCommands
		add                bool
		remove             bool
		expectedContain    []string
		expectedNotContain []string
		mockAddError       error
	}{
		{
			name: "success-add",
			input: ShoppingListCommands{
				MethodAdd: {
					ItemIds:    nil,
					Categories: []string{"stationery", "stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)", "A4 notebook"},
				},
			},
			add:             true,
			expectedContain: []string{"一盒彩色马克笔(marker)", "A4 notebook"},
		},
		{
			name: "failed-wrong length",
			input: ShoppingListCommands{
				MethodAdd: {
					ItemIds:    nil,
					Categories: []string{"stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)", "A4 notebook"},
				},
			},
			add:                true,
			expectedNotContain: []string{"一盒彩色马克笔(marker)", "A4 notebook"},
		},
		{
			name: "success-remove",
			input: ShoppingListCommands{
				MethodRemove: {
					ItemIds:    []int64{1, 2},
					Categories: []string{"stationery", "stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)", "A4 notebook"},
				},
			},
			remove:          true,
			expectedContain: []string{"一盒彩色马克笔(marker)", "A4 notebook"},
		},
		{
			name: "failed-remove-empty",
			input: ShoppingListCommands{
				MethodRemove: {
					ItemIds:    []int64{1, 2},
					Categories: []string{"stationery"},
					ItemNames:  []string{"一盒彩色马克笔(marker)", "A4 notebook"},
				},
			},
			remove:          true,
			expectedContain: []string{"no updates"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			if tt.add {
				shoppingListRepo.On("CreateShoppingListItems",
					mock.Anything,
					mock.Anything,
				).Return(func(ctx context.Context, arg sqlc.CreateShoppingListItemsParams) []sqlc.ShoppingList {
					items := make([]sqlc.ShoppingList, len(arg.ItemNames))
					for i := range arg.ItemNames {
						items[i] = sqlc.ShoppingList{
							ID:             int64(i + 1),
							TelegramUserID: arg.TelegramUserID,
							Name:           arg.ItemNames[i],
							Category:       arg.Categories[i],
						}
					}
					return items
				}, tt.mockAddError)
			}
			if tt.remove {
				shoppingListRepo.On("RemoveShoppingListItems",
					mock.Anything,
					mock.Anything,
				).Return(func(ctx context.Context, arg sqlc.RemoveShoppingListItemsParams) []sqlc.ShoppingList {
					items := make([]sqlc.ShoppingList, len(arg.ItemIds))
					for i, id := range tt.input[MethodRemove].ItemIds {
						if slices.Contains(arg.ItemIds, id) {
							items[i] = sqlc.ShoppingList{
								ID:             int64(i + 1),
								TelegramUserID: arg.TelegramUserID,
								Name:           tt.input[MethodRemove].ItemNames[i],
								Category:       tt.input[MethodRemove].Categories[i],
							}
						}
					}
					return items
				}, tt.mockAddError)
			}
			result := testAgent.executeShoppingListCommands(ctx, telegramUserID, tt.input)
			resultContent := result.Content
			if len(tt.expectedContain) > 0 {
				for _, name := range tt.expectedContain {
					assert.Contains(t, resultContent, name)
				}
			}
			if len(tt.expectedNotContain) > 0 {
				for _, name := range tt.expectedNotContain {
					assert.NotContains(t, resultContent, name)
				}
			}
		})
	}
}
