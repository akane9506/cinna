package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	telegramBotTokenEnv  = "TELEGRAM_BOT_TOKEN"
	allowedAdminUsersEnv = "ALLOWED_ADMIN_USERS"
)

type Config struct {
	TelegramBotToken  string
	AllowedAdminUsers []int64
}

func LoadConfig() (*Config, error) {
	token := strings.TrimSpace(os.Getenv(telegramBotTokenEnv))
	if token == "" {
		return nil, fmt.Errorf("%s is required", telegramBotTokenEnv)
	}
	adminUsers, err := parseAllowedAdminUsers()
	if err != nil {
		return nil, fmt.Errorf(
			"Error loading %s: %w", allowedAdminUsersEnv, err)
	}
	return &Config{
		TelegramBotToken:  token,
		AllowedAdminUsers: adminUsers,
	}, nil
}

func parseAllowedAdminUsers() ([]int64, error) {
	rawUsers := os.Getenv(allowedAdminUsersEnv)
	allowedAdminUsers := []int64{}
	for userStr := range strings.SplitSeq(rawUsers, ",") {
		userStr = strings.TrimSpace(userStr)
		if len(userStr) == 0 {
			continue
		}
		id, err := strconv.ParseInt(userStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid Admin User ID: %s", userStr)
		}
		allowedAdminUsers = append(allowedAdminUsers, id)
	}
	if len(allowedAdminUsers) == 0 {
		return nil, fmt.Errorf("Admin users shouldn't be empty")
	}
	return allowedAdminUsers, nil
}
