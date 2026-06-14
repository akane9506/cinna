package app

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	goEnv                = "GO_ENV"
	telegramBotTokenEnv  = "TELEGRAM_BOT_TOKEN"
	allowedAdminUsersEnv = "ALLOWED_ADMIN_USERS"
	webhookURLEnv        = "WEBHOOK_URL"
	webhookSecretEnv     = "WEBHOOK_SECRET"
)

const (
	RuntimeDev  = "development"
	RuntimeProd = "production"
)

type Config struct {
	RuntimeEnv        string
	TelegramBotToken  string
	WebhookURL        string
	WebhookSecret     string
	AllowedAdminUsers []int64
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	// setup runtime env
	runtimeEnv := strings.TrimSpace(os.Getenv(goEnv))
	if runtimeEnv == "" {
		config.RuntimeEnv = RuntimeDev
	} else if !slices.Contains([]string{RuntimeDev, RuntimeProd}, runtimeEnv) {
		return nil, fmt.Errorf("invalid %s: %s", goEnv, runtimeEnv)
	} else {
		config.RuntimeEnv = runtimeEnv
	}

	// setup telegram bot token
	botToken := strings.TrimSpace(os.Getenv(telegramBotTokenEnv))
	if botToken == "" {
		return nil, fmt.Errorf("%s is required", telegramBotTokenEnv)
	}
	config.TelegramBotToken = botToken

	// setup webhook if runtime environment is prod
	if config.RuntimeEnv == RuntimeProd {
		// setup webhook url
		webhookURL := strings.TrimSpace(os.Getenv(webhookURLEnv))
		if !strings.HasPrefix(webhookURL, "https://") {
			return nil, fmt.Errorf("%s should starts with https://, got: %s", webhookURLEnv, webhookURL)
		}
		config.WebhookURL = webhookURL
		webhookSecret := strings.TrimSpace(os.Getenv(webhookSecretEnv))
		if webhookSecret == "" {
			return nil, fmt.Errorf("%s shouldn't be empty", webhookSecretEnv)
		}
		config.WebhookSecret = webhookSecret
	}

	// setup allowed admin user list
	adminUsers, err := parseAllowedAdminUsers()
	if err != nil {
		return nil, fmt.Errorf(
			"Error loading %s: %w", allowedAdminUsersEnv, err)
	}
	config.AllowedAdminUsers = adminUsers

	return config, nil
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
