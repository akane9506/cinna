package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/akane9506/cinna/internal/app/ports"
	"github.com/akane9506/cinna/internal/postgres/sqlc"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mocks
type mockBotChatValidator struct {
	mock.Mock
}

func (m *mockBotChatValidator) GetChat(
	ctx context.Context, params *bot.GetChatParams) (*models.ChatFullInfo, error) {
	args := m.Called(ctx, params)
	return &models.ChatFullInfo{}, args.Error(0)
}

type mockAllowListRepository struct {
	mock.Mock
}

func (m *mockAllowListRepository) IsAdminUser(
	ctx context.Context, telegramUserID int64) (bool, error) {
	args := m.Called(ctx, telegramUserID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockAllowListRepository) IsAllowedUser(
	ctx context.Context, telegramUserID int64) (bool, error) {
	args := m.Called(ctx, telegramUserID)
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockAllowListRepository) UpsertAllowedUser(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	args := m.Called(ctx, telegramUserID)
	return sqlc.AllowedUser{}, args.Error(0)
}

func (m *mockAllowListRepository) SubscribeNotification(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	args := m.Called(ctx, telegramUserID)
	return sqlc.AllowedUser{}, args.Error(0)
}

func (m *mockAllowListRepository) UnsubscribeNotification(
	ctx context.Context, telegramUserID int64) (sqlc.AllowedUser, error) {
	args := m.Called(ctx, telegramUserID)
	return sqlc.AllowedUser{}, args.Error(0)
}

func (m *mockAllowListRepository) DailyNotificationSubscribers(
	ctx context.Context) ([]int64, error) {
	args := m.Called(ctx)
	return args.Get(0).([]int64), args.Error(1)
}

// tests

func TestIsCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          string
		expectedResult bool
	}{
		{"command", "/", true},
		{"command with leading and tailing white spaces", " /adc  ", true},
		{"non-command empty", "   ", false},
		{"non-command message", " abc  ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			result := isCommand(tt.input)
			assert.Equal(t, result, tt.expectedResult)
		})
	}
}

func TestParseCommandAndArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		input           string
		expectedCommand string
		expectedArgs    []string
	}{
		{
			name:            "command only",
			input:           "/addmember ",
			expectedCommand: "/addmember",
			expectedArgs:    []string{},
		},
		{
			name:            "command with args",
			input:           " /command 123456    abcde",
			expectedCommand: "/command",
			expectedArgs:    []string{"123456", "abcde"},
		},
		{
			name:            "empty commands",
			input:           "    ",
			expectedCommand: "",
			expectedArgs:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			cmd, args := parseCommandAndArgs(tt.input)
			assert.Equal(t, tt.expectedCommand, cmd)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestHandleAddMemberCommand(t *testing.T) {
	t.Parallel()
	mockAllowListRepo := new(mockAllowListRepository)
	mockBot := new(mockBotChatValidator)
	mockLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockClient := Client{
		logger:       mockLogger,
		repositories: &ports.Repositories{AllowList: mockAllowListRepo},
	}
	tests := []struct {
		name             string
		args             []string
		invalidArgsError bool
		parseIntError    bool
		mockGetChatError bool
		getChatError     error
		isNoRowsError    bool
		mockUpsertError  bool
		upsertError      error
	}{
		{
			name:             "invalid arguments",
			args:             []string{"123", "345"},
			invalidArgsError: true,
		},
		{
			name:          "invalid integer - error",
			args:          []string{"abc"},
			parseIntError: true,
		},
		{
			name:          "invalid integer - zero",
			args:          []string{"0"},
			parseIntError: true,
		},
		{
			name:             "invalid chat user",
			args:             []string{"123456"},
			mockGetChatError: true,
			getChatError:     errors.New("invalid user"),
		},
		{
			name:          "user already exist",
			args:          []string{"123456"},
			isNoRowsError: true,
			upsertError:   sql.ErrNoRows,
		},
		{
			name:            "error upsert",
			args:            []string{"123456"},
			mockUpsertError: true,
			upsertError:     errors.New("failed to upsert"),
		},
		{
			name: "success",
			args: []string{"123456"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			ctx := context.Background()
			if !tt.invalidArgsError && !tt.parseIntError {
				mockBot.On("GetChat",
					mock.Anything,
					mock.AnythingOfType("*bot.GetChatParams")).
					Return(tt.getChatError).Once()
			}
			if !tt.invalidArgsError && !tt.parseIntError && !tt.mockGetChatError {
				mockAllowListRepo.On("UpsertAllowedUser",
					mock.Anything,
					mock.AnythingOfType("int64")).
					Return(tt.upsertError).Once()
			}
			msg := mockClient.handleAddMemberCommand(ctx, mockBot, tt.args)
			if tt.invalidArgsError {
				assert.Contains(t, msg, "Invalid arguments")
				return
			}
			if tt.parseIntError {
				assert.Contains(t, msg, "Invalid Telegram user ID")
				return
			}
			if tt.mockGetChatError {
				assert.Contains(t, msg, "User unavailable")
				return
			}
			if tt.isNoRowsError {
				assert.Contains(t, msg, "User already allowed")
				return
			}
			if tt.mockUpsertError {
				assert.Contains(t, msg, "Unable to add user")
				return
			}
			assert.Contains(t, msg, "User added")
		})
	}
}

func TestHandleManageNotificationCommand(t *testing.T) {
	t.Parallel()

	const telegramUserID int64 = 123456

	tests := []struct {
		name             string
		args             []string
		want             string
		subscribeError   error
		unsubscribeError error
	}{
		{
			name: "invalid arguments",
			args: []string{},
			want: fmt.Sprintf("%s\n%s\n%s\n%s",
				"命令使用方式：<code>/notify &lt;on|off&gt;</code>\n",
				"<b>示例:</b>",
				"<code>/notify on</code> \t 打开每日提醒",
				"<code>/notify off</code> \t 关闭每日提醒",
			),
		},
		{
			name: "too many arguments",
			args: []string{"on", "extra"},
			want: fmt.Sprintf("%s\n%s\n%s\n%s",
				"命令使用方式：<code>/notify &lt;on|off&gt;</code>\n",
				"<b>示例:</b>",
				"<code>/notify on</code> \t 打开每日提醒",
				"<code>/notify off</code> \t 关闭每日提醒",
			),
		},
		{
			name: "invalid status",
			args: []string{"maybe"},
			want: "<b>Invalid argument</b>\nThe argument should be either on or off",
		},
		{
			name: "turn on success",
			args: []string{"on"},
			want: "Notification <b>On</b>\n",
		},
		{
			name:           "turn on failure",
			args:           []string{"on"},
			want:           "Failed to turn on the notification\n.",
			subscribeError: errors.New("subscribe failed"),
		},
		{
			name: "turn off success",
			args: []string{"off"},
			want: "Notification <b>Off</b>\n",
		},
		{
			name:             "turn off failure",
			args:             []string{"off"},
			want:             "Failed to turn off the notification\n.",
			unsubscribeError: errors.New("unsubscribe failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowListRepo := new(mockAllowListRepository)
			client := Client{
				logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
				repositories: &ports.Repositories{AllowList: allowListRepo},
			}
			ctx := context.Background()
			if len(tt.args) == 1 {
				switch tt.args[0] {
				case "on":
					allowListRepo.
						On("SubscribeNotification", ctx, telegramUserID).
						Return(tt.subscribeError).
						Once()
				case "off":
					allowListRepo.
						On("UnsubscribeNotification", ctx, telegramUserID).
						Return(tt.unsubscribeError).
						Once()
				}
			}
			got := client.handleManageNotificationCommand(
				ctx,
				telegramUserID,
				tt.args,
			)
			assert.Equal(t, tt.want, got)
			allowListRepo.AssertExpectations(t)
		})
	}
}
