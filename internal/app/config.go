package app

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	goEnv               = "GO_ENV"
	telegramBotTokenEnv = "TELEGRAM_BOT_TOKEN"
	webhookURLEnv       = "WEBHOOK_URL"
	webhookSecretEnv    = "WEBHOOK_SECRET"
	dbURLEnv            = "DATABASE_URL"
	// LLM key
	deepseekEnv = "DEEPSEEK_API_KEY"
	// Encryption key
	messageEncryptionKeyIDEnv = "MESSAGE_ENCRYPTION_KEY_ID"
	messageEncryptionKeyEnv   = "MESSAGE_ENCRYPTION_KEY"
	// Timezone information
	appTimezoneEnv = "APP_TIMEZONE"
)

const (
	RuntimeDev  = "development"
	RuntimeProd = "production"
)

type Config struct {
	RuntimeEnv             string
	TelegramBotToken       string
	WebhookURL             string
	WebhookSecret          string
	DatabaseURL            string
	DeepseekAPIKey         string
	MessageEncryptionKeyID string
	MessageEncryptionKey   string
	AppTimezone            string
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

	// setup database url
	dbUrl := strings.TrimSpace(os.Getenv(dbURLEnv))
	if dbUrl == "" {
		return nil, fmt.Errorf("%s is required", dbURLEnv)
	}
	config.DatabaseURL = dbUrl

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

	// setup LLM API keys
	deepseekAPIKey := strings.TrimSpace(os.Getenv(deepseekEnv))
	config.DeepseekAPIKey = deepseekAPIKey

	// setup encryption params
	messageEncryptionKey := strings.TrimSpace(os.Getenv(messageEncryptionKeyEnv))
	if messageEncryptionKey == "" {
		return nil, fmt.Errorf("%s is required", messageEncryptionKeyEnv)
	}
	messageEncryptionKeyID := strings.TrimSpace(os.Getenv(messageEncryptionKeyIDEnv))
	if messageEncryptionKeyID == "" {
		return nil, fmt.Errorf("%s is required", messageEncryptionKeyIDEnv)
	}
	config.MessageEncryptionKeyID = messageEncryptionKeyID
	config.MessageEncryptionKey = messageEncryptionKey

	// setup timezone
	appTimezone := strings.TrimSpace(os.Getenv(appTimezoneEnv))
	if appTimezone == "" {
		return nil, fmt.Errorf("Invalid Timezone: %s", appTimezone)
	}
	_, err := time.LoadLocation(appTimezone)
	if err != nil {
		return nil, fmt.Errorf("Invalid Timezone: %s. Error: %s", appTimezone, err.Error())
	}
	config.AppTimezone = appTimezone
	return config, nil
}
