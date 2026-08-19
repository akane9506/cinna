package telegram

// This file contains telegram handlers for package internal use

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// this client is for getting user's voice message
var telegramVoiceHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

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
	switch {
	case isCommand(update.Message.Text):
		c.handleCommands(ctx, b, update)
	case update.Message.Voice != nil:
		c.handleVoice(ctx, b, update)
	default:
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
		c.sendHTMLReply(
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
	c.sendHTMLReply(ctx, b, update.Message.Chat.ID, replyMessage)
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
	typingChan := setActionStatus(ctx, b, update, models.ChatActionTyping)
	err := c.sendAgentReply(ctx, b, update.Message.Chat.ID, update.Message.Text)
	if err != nil {
		logger.Error("error handling text message", "error", err)
	}
	close(typingChan)
}

// handle voice input
func (c *Client) handleVoice(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := c.logger.With("path", "internal/app/telegram/handlers_internal/handleVoice")
	chatID := update.Message.Chat.ID
	voiceFileID := update.Message.Voice.FileID
	// get voice file from telegram
	voiceChan := setActionStatus(ctx, b, update, models.ChatActionUploadVoice)
	voiceFile, err := getUserVoice(ctx, b, voiceFileID)
	if err != nil {
		close(voiceChan)
		logger.Error("error processing user voice", "error", err)
		c.sendPlainReply(ctx, b, chatID, "Fail to get voice file", logger)
		return
	}
	defer func() {
		// don't forget to close and delete the file
		voiceFile.Close()
		os.Remove(voiceFile.Name())
	}()
	text, err := c.agentHandler.HandleAudio(ctx, voiceFile)
	close(voiceChan)
	if err != nil {
		logger.Error("failed to transcribe audio", "error", err)
		c.sendPlainReply(ctx, b, chatID, "failed to transcribe audio", logger)
		return
	}
	// generate and send agent reply
	typingChan := setActionStatus(ctx, b, update, models.ChatActionTyping)
	err = c.sendAgentReply(ctx, b, chatID, text)
	if err != nil {
		logger.Error("error handling text message", "error", err)
	}
	close(typingChan)
}

// generate agent reply and send message
func (c *Client) sendAgentReply(
	ctx context.Context, b *bot.Bot, chatID int64, message string) error {
	loc, _ := time.LoadLocation(c.timezone) // config has already verified timezone loading
	now := time.Now().In(loc)
	reply, err := c.agentHandler.HandleText(ctx, chatID, now, message)
	if err != nil {
		if _, errInner := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "internal server error",
		}); errInner != nil {
			return fmt.Errorf("failed to get reply from agent: %w; failed to get reply from agent: %w",
				err, errInner)
		}
		return fmt.Errorf("failed to get reply from agent: %w", err)
	}
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   reply,
	})
	if err != nil {
		return fmt.Errorf("failed to send telegram response: %w", err)
	}
	return nil
}

// send HTML formatted message to the given chat
func (c *Client) sendHTMLReply(
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

// send plain text reply to the given chat
func (c *Client) sendPlainReply(
	ctx context.Context,
	b *bot.Bot,
	chatID int64,
	text string,
	logger *slog.Logger,
) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		logger.Error("failed to send reply", "telegram_user_id", chatID, "error", err)
	}
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
func setActionStatus(
	ctx context.Context,
	b *bot.Bot,
	update *models.Update,
	action models.ChatAction,
) chan struct{} {
	statusChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()

		sendStatus := func() {
			b.SendChatAction(ctx, &bot.SendChatActionParams{
				ChatID: update.Message.Chat.ID,
				Action: action,
			})
		}
		sendStatus()
		for {
			select {
			case <-statusChan:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				sendStatus()
			}
		}
	}()
	return statusChan
}

// get telegram voice message file
// we don't really limit the file size at this time, because user's voice message is usually small
func getUserVoice(ctx context.Context, b *bot.Bot, fileID string) (*os.File, error) {
	// step 1: get file url from telegram
	voiceFile, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get voice file: %w", err)
	}
	if voiceFile.FilePath == "" {
		return nil, errors.New("telegram returns an empty file path")
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		b.FileDownloadLink(voiceFile),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram voice download request: %w", err)
	}
	response, err := telegramVoiceHTTPClient.Do(request)
	if err != nil {
		var urlError *url.Error
		if errors.As(err, &urlError) {
			return nil, fmt.Errorf("failed to download voice file: %w", urlError.Err)
		}
		return nil, errors.New("failed to download voice file")
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"telegram file download returned status %d",
			response.StatusCode,
		)
	}
	// step 2: save audio buffer to .ogg file
	oggFile, err := os.CreateTemp("", "telegram-voice-*.ogg")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create temp ogg file: %w",
			err,
		)
	}
	cleanUp := func() {
		oggFile.Close()
		os.Remove(oggFile.Name())
	}
	if _, err = io.Copy(oggFile, response.Body); err != nil {
		cleanUp()
		return nil, fmt.Errorf(
			"failed to save audio to temp ogg file %q: %w",
			oggFile.Name(),
			err,
		)
	}
	// step 3: rewind file to the audio start position (time 0)
	if _, err = oggFile.Seek(0, io.SeekStart); err != nil {
		cleanUp()
		return nil, fmt.Errorf(
			"failed to rewind temp ogg file %q: %w",
			oggFile.Name(),
			err,
		)
	}
	return oggFile, nil
}
