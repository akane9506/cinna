package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var mockDBUrl string = "postgres://a:a_pswrd@localhost:5432/cinna"
var mockDeepseekAPIKey string = "deepseek_api_key"
var mockMessageEncKey = "key"
var mockMessageEncKeyID = "abcdef"

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                       string
		runtimeEnv                 string
		telegramEnv                string
		dbUrlEnv                   string
		deepseekAPIEnv             string
		webhookUrlEnv              string
		webhookSecretEnv           string
		messageEncKeyID            string
		messageEncKey              string
		expected                   *Config
		expectedRuntimeError       bool
		expectDBUrlError           bool
		expectTelegramError        bool
		expectWebhookUrlError      bool
		expectWebhookSecretError   bool
		expectMessageEncKeyError   bool
		expectMessageEncKeyIDError bool
	}{
		{
			name:            "success in development",
			telegramEnv:     "abc",
			dbUrlEnv:        mockDBUrl,
			deepseekAPIEnv:  mockDeepseekAPIKey,
			messageEncKeyID: mockMessageEncKeyID,
			messageEncKey:   mockMessageEncKey,
			expected: &Config{
				TelegramBotToken:       "abc",
				DatabaseURL:            mockDBUrl,
				DeepseekAPIKey:         mockDeepseekAPIKey,
				RuntimeEnv:             "development",
				MessageEncryptionKeyID: mockMessageEncKeyID,
				MessageEncryptionKey:   mockMessageEncKey,
			},
		},
		{
			name:             "success in production",
			runtimeEnv:       "production",
			telegramEnv:      "abc",
			dbUrlEnv:         mockDBUrl,
			deepseekAPIEnv:   mockDeepseekAPIKey,
			messageEncKeyID:  mockMessageEncKeyID,
			messageEncKey:    mockMessageEncKey,
			webhookUrlEnv:    "https://localhost",
			webhookSecretEnv: "secret",
			expected: &Config{
				TelegramBotToken:       "abc",
				RuntimeEnv:             "production",
				DatabaseURL:            mockDBUrl,
				DeepseekAPIKey:         mockDeepseekAPIKey,
				WebhookURL:             "https://localhost",
				WebhookSecret:          "secret",
				MessageEncryptionKeyID: mockMessageEncKeyID,
				MessageEncryptionKey:   mockMessageEncKey,
			},
		},
		{
			name:                 "invalid runtime",
			dbUrlEnv:             mockDBUrl,
			runtimeEnv:           "wrong",
			expectedRuntimeError: true,
		},
		{
			name:             "missing db url",
			telegramEnv:      "",
			webhookUrlEnv:    "http://localhost",
			webhookSecretEnv: "secret",
			expectDBUrlError: true,
		},
		{
			name:                "invalid telegram env",
			telegramEnv:         "",
			dbUrlEnv:            mockDBUrl,
			webhookUrlEnv:       "http://localhost",
			webhookSecretEnv:    "secret",
			expectTelegramError: true,
		},
		{
			name:                  "missing webhook in production",
			dbUrlEnv:              mockDBUrl,
			runtimeEnv:            "production",
			telegramEnv:           "abc",
			expectWebhookUrlError: true,
		},
		{
			name:                     "missing websocket secret",
			dbUrlEnv:                 mockDBUrl,
			runtimeEnv:               "production",
			telegramEnv:              "abc",
			webhookUrlEnv:            "https://localhost",
			expectWebhookSecretError: true,
		},
		{
			name:                     "missing msg enc key in prod",
			runtimeEnv:               "production",
			telegramEnv:              "abc",
			dbUrlEnv:                 mockDBUrl,
			deepseekAPIEnv:           mockDeepseekAPIKey,
			messageEncKeyID:          mockMessageEncKeyID,
			webhookUrlEnv:            "https://localhost",
			webhookSecretEnv:         "secret",
			expectMessageEncKeyError: true,
		},
		{
			name:                       "missing msg enc key in prod",
			runtimeEnv:                 "production",
			telegramEnv:                "abc",
			dbUrlEnv:                   mockDBUrl,
			deepseekAPIEnv:             mockDeepseekAPIKey,
			messageEncKey:              mockMessageEncKey,
			webhookUrlEnv:              "https://localhost",
			webhookSecretEnv:           "secret",
			expectMessageEncKeyIDError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(goEnv, tt.runtimeEnv)
			t.Setenv(telegramBotTokenEnv, tt.telegramEnv)
			t.Setenv(webhookURLEnv, tt.webhookUrlEnv)
			t.Setenv(webhookSecretEnv, tt.webhookSecretEnv)
			t.Setenv(dbURLEnv, tt.dbUrlEnv)
			t.Setenv(deepseekEnv, tt.deepseekAPIEnv)
			t.Setenv(messageEncryptionKeyEnv, tt.messageEncKey)
			t.Setenv(messageEncryptionKeyIDEnv, tt.messageEncKeyID)
			config, err := LoadConfig()
			if tt.expected == nil {
				assert.Error(t, err)
				if tt.expectTelegramError {
					assert.Contains(t, err.Error(), telegramBotTokenEnv)
				}
				if tt.expectWebhookUrlError {
					assert.Contains(t, err.Error(), webhookURLEnv)
				}
				if tt.expectWebhookSecretError {
					assert.Contains(t, err.Error(), webhookSecretEnv)
				}
				if tt.expectedRuntimeError {
					assert.Contains(t, err.Error(), goEnv)
				}
				if tt.expectDBUrlError {
					assert.Contains(t, err.Error(), dbURLEnv)
				}
				if tt.expectMessageEncKeyError {
					assert.Contains(t, err.Error(), messageEncryptionKeyEnv)
				}
				if tt.expectMessageEncKeyIDError {
					assert.Contains(t, err.Error(), messageEncryptionKeyIDEnv)
				}
			} else {
				assert.Equal(t, config, tt.expected)
			}
		})
	}
}
