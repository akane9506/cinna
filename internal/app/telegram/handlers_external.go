package telegram

import (
	"context"
	"errors"
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
		return fmt.Errorf("failed to get daily notification subscribers: %w", err)
	}

	jobs := make(chan int64)
	errs := make(chan error, len(subscribers))
	var wg sync.WaitGroup
	for range NUM_WORKERS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range jobs {
				if err := c.sendDailyNotification(ctx, userID); err != nil {
					errs <- err
				}
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
	close(errs)

	var notificationErrors []error
	for err := range errs {
		notificationErrors = append(notificationErrors, err)
	}
	return errors.Join(append(notificationErrors, ctx.Err())...)
}

func (c *Client) sendDailyNotification(
	ctx context.Context, userID int64) error {
	// check if the chat exists
	_, err := c.bot.GetChat(ctx, &bot.GetChatParams{ChatID: userID})
	if err != nil {
		return fmt.Errorf(
			"failed to validate telegram chat for user %d: %w",
			userID,
			err,
		)
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
		return fmt.Errorf("failed to generate daily notification for user %d: %w",
			userID,
			err,
		)
	}
	_, err = c.bot.SendMessage(ctx, &bot.SendMessageParams{ChatID: userID, Text: reply})
	if err != nil {
		return fmt.Errorf("failed to send message to user %d: %w",
			userID,
			err,
		)
	}
	return nil
}
