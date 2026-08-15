package telegram

// This file contains telegram handlers for package internal use

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

//go:embed command_list.md
var commandDoc string

//go:embed command_list_admin.md
var adminCommandDoc string

var adminCommands = []string{"/addmember"}

// general handler, processes all incoming updates
func (c *Client) handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	if isCommand(update.Message.Text) {
		c.handleCommands(ctx, b, update)
		return
	} else {
		c.handleText(ctx, b, update)
	}
}

// handle /-based commands
func (c *Client) handleCommands(
	ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := c.logger.With("path", "internal/telegram/handlers/handleCommands")
	telegramUserID := update.Message.From.ID
	isAdmin, err := c.repositories.AllowList.IsAdminUser(ctx, telegramUserID)
	if err != nil {
		logger.Error("failed to authenticate admin user",
			"telegram_userid", telegramUserID,
			"error", err)
		return
	}
	command, args := parseCommandAndArgs(update.Message.Text)
	if slices.Contains(adminCommands, command) && !isAdmin {
		c.sendCommandReply(
			ctx,
			b,
			update.Message.Chat.ID,
			"⛔ <b>Permission denied</b>\n\nthis command is only available to bot administrators.",
		)
		return
	}
	var replyMessage string
	switch command {
	case "/addmember":
		replyMessage = c.handleAddMemberCommand(ctx, b, args)
	case "/help":
		replyMessage = commandDoc
		if isAdmin {
			replyMessage += "\n" + adminCommandDoc
		}
	case "/notify":
		replyMessage = c.handleManageNotificationCommand(ctx, telegramUserID, args)
	default:
		replyMessage = "❌ <b>Unknown command</b>\n\nUse /help to view the available commands."
	}
	c.sendCommandReply(ctx, b, update.Message.Chat.ID, replyMessage)
}

func (c *Client) sendCommandReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	text string,
) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		c.logger.Error("failed to send command reply", "chat_id", chatID, "error", err)
	}
}

// interface to help test the function. Directly pass *bot.Bot to the function
type botChatValidator interface {
	GetChat(ctx context.Context, params *bot.GetChatParams) (*models.ChatFullInfo, error)
}

// add telegram user id to the allow list, newly added user will default to be "member"
func (c *Client) handleAddMemberCommand(
	ctx context.Context, b botChatValidator, args []string) string {
	logger := c.logger.With("path", "internal/telegram/handlers/handleAddMemberCommand")
	if len(args) != 1 {
		return "<b>Invalid arguments</b>\nUsage: <code>/addmember &lt;telegram_user_id&gt;</code>"
	}
	// parse user id
	userID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil || userID <= 0 {
		return "<b>Invalid Telegram user ID</b>\nThe user ID must be a positive integer."
	}
	// validate user id. this requires the user to setup a chat with Cinna
	_, err = b.GetChat(ctx, &bot.GetChatParams{ChatID: userID})
	if err != nil {
		logger.Warn("failed to validate telegram user",
			"target_user_id", userID,
			"error", err,
		)
		return "<b>User unavailable</b>\nAsk the user to start this bot, then try again."
	}
	_, err = c.repositories.AllowList.UpsertAllowedUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "<b>User already allowed</b>\nNo changes were made."
		}
		logger.Error(
			"failed to upsert user to the database",
			"telegram_user_id", userID,
			"error", err,
		)
		return "<b>Unable to add user</b>\nCheck the server logs for details."
	}
	return "<b>User added</b>\nThe user can now access Cinna."
}

func (c *Client) handleManageNotificationCommand(
	ctx context.Context,
	telegramUserID int64,
	args []string) string {
	logger := c.logger.With("path", "internal/telegram/handlers/handleManageNotificationCommans")
	if len(args) != 1 {
		return "<b>Invalid arguments</b>\nUsage: <code>/notify &lt;on|off&gt;</code>"
	}
	status := args[0]
	if status != "on" && status != "off" {
		return "<b>Invalid argument</b>\nThe argument should be either on or off"
	}
	if status == "on" {
		_, err := c.repositories.AllowList.SubscribeNotification(ctx, telegramUserID)
		if err != nil {
			logger.Error("failed to turn on notification for the user",
				"telegram_user_id", telegramUserID,
				"error", err,
			)
			return "Failed to turn on the notification\n."
		}
		return "Notification <b>On</b>\n"
	}
	_, err := c.repositories.AllowList.UnsubscribeNotification(ctx, telegramUserID)
	if err != nil {
		logger.Error("failed to turn off notification for the user",
			"telegram_user_id", telegramUserID,
			"error", err,
		)
		return "Failed to turn off the notification\n."
	}
	return "Notification <b>Off</b>\n"
}

// handle text message
func (c *Client) handleText(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message.Text == "" {
		return
	}
	logger := c.logger.With("path", "internal/telegram/handlers/handleText")
	typingDone := setTypingAction(ctx, b, update)
	loc, _ := time.LoadLocation(c.timezone) // config has already verified timezone loading
	now := time.Now().In(loc)
	reply, err := c.agentHandler.HandleText(
		ctx,
		update.Message.From.ID,
		now,
		update.Message.Text,
	)
	if err != nil {
		logger.Error(
			"failed to get reply from agent",
			"user_id", update.Message.From.ID,
			"error", err,
		)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "internal server error",
		})
		close(typingDone)
		return
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   reply,
	})
	if err != nil {
		logger.Error(
			"failed to send telegram response",
			"chat_id", update.Message.Chat.ID,
			"error", err,
		)
	}
	close(typingDone)
}

// ========== helper functions =========

// check if the trimmed text starts with '/'
func isCommand(text string) bool {
	command := strings.TrimSpace(text)
	if strings.HasPrefix(command, "/") {
		return true
	}
	return false
}

// parse the command into [command] + [args slice]
func parseCommandAndArgs(text string) (string, []string) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return "", []string{}
	}
	command, args := parts[0], parts[1:]
	return strings.ToLower(command), args
}

// persist the status typing when the bot is processing the message
func setTypingAction(ctx context.Context, b *bot.Bot, update *models.Update) chan struct{} {
	typingDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		sendTyping := func() {
			b.SendChatAction(ctx, &bot.SendChatActionParams{
				ChatID: update.Message.Chat.ID,
				Action: models.ChatActionTyping,
			})
		}
		sendTyping()
		for {
			select {
			case <-typingDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendTyping()
			}
		}
	}()
	return typingDone
}
