package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-telegram/bot"
)

// This file exposes telegram handlers that can be used by other packages

const NUM_WORKERS = 2 // spawn two workers to process daily notifications concurrently

func (c *Client) HandleDailyNotification(ctx context.Context) error {
	logger := c.logger.With("path", "internal/app/telegram/handlers_external/HandleDailyNotification")
	subscribers, err := c.repositories.AllowList.DailyNotificationSubscribers(ctx)
	if err != nil {
		logger.Error("failed to get daily notification subscribers",
			"error", err,
		)
		return fmt.Errorf("failed to get daily notification subscribers")
	}

	jobs := make(chan int64)
	var wg sync.WaitGroup
	for range NUM_WORKERS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				userID, ok := <-jobs
				if !ok {
					break
				}
				c.sendDailyNotification(ctx, userID)
			}
		}()
	}

enqueue:
	for _, userID := range subscribers {
		select {
		case <-ctx.Done():
			break enqueue
		case jobs <- userID:
		}
	}

	close(jobs)
	wg.Wait()
	return ctx.Err()
}

func (c *Client) sendDailyNotification(ctx context.Context, userID int64) {
	// check if the chat exists
	logger := c.logger.With("path", "internal/app/telegram/handler_external/sendDailyNotification")
	_, err := c.bot.GetChat(ctx, &bot.GetChatParams{ChatID: userID})
	if err != nil {
		logger.Warn("failed to validate telegram chat",
			"telegram_user_id", userID,
			"error", err,
		)
		return
	}
	// process message with agent and send notification
	loc, _ := time.LoadLocation(c.timezone) // config has already verified timezone loading
	now := time.Now().In(loc)
	reply, err := c.agentHandler.HandleText(
		ctx,
		userID,
		now,
		"向用户发送简洁、有帮助的每日通知。内容应贴近用户当前情况，并突出最重要的更新或下一步行动；避免泛泛而谈和不必要的细节。",
	)
	if err != nil {
		logger.Error("Failed to generate daily notification message",
			"error", err,
		)
		return // do not let user know when the agent is down
	}
	c.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: userID, Text: reply})
}
