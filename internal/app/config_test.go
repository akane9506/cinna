package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var mockDBUrl string = "postgres://a:a_pswrd@localhost:5432/cinna"
var mockDeepseekAPIKey string = "deepseek_api_key"
var mockOpenaiAPIKey string = "openai_api_key"
var mockMessageEncKey = "key"
var mockMessageEncKeyID = "abcdef"
var mockTimezone = "America/Los_Angeles"

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                       string
		runtimeEnv                 string
		telegramEnv                string
		dbUrlEnv                   string
		deepseekAPIEnv             string
		openaiAPIEnv               string
		webhookUrlEnv              string
		webhookSecretEnv           string
		messageEncKeyID            string
		messageEncKey              string
		appTimezone                string
		expected                   *Config
		expectedRuntimeError       bool
		expectDBUrlError           bool
		expectTelegramError        bool
		expectWebhookUrlError      bool
		expectWebhookSecretError   bool
		expectMessageEncKeyError   bool
		expectMessageEncKeyIDError bool
		expectAppTimezoneError     bool
	}{
		{
			name:             "success in development",
			telegramEnv:      "abc",
			dbUrlEnv:         mockDBUrl,
			deepseekAPIEnv:   mockDeepseekAPIKey,
			openaiAPIEnv:     mockOpenaiAPIKey,
			messageEncKeyID:  mockMessageEncKeyID,
			messageEncKey:    mockMessageEncKey,
			webhookSecretEnv: "secret",
			appTimezone:      mockTimezone,
			expected: &Config{
				TelegramBotToken:       "abc",
				DatabaseURL:            mockDBUrl,
				DeepseekAPIKey:         mockDeepseekAPIKey,
				OpenaiAPIKey:           mockOpenaiAPIKey,
				RuntimeEnv:             "development",
				WebhookSecret:          "secret",
				MessageEncryptionKeyID: mockMessageEncKeyID,
				MessageEncryptionKey:   mockMessageEncKey,
				AppTimezone:            mockTimezone,
			},
		},
		{
			name:             "success in production",
			runtimeEnv:       "production",
			telegramEnv:      "abc",
			dbUrlEnv:         mockDBUrl,
			deepseekAPIEnv:   mockDeepseekAPIKey,
			openaiAPIEnv:     mockOpenaiAPIKey,
			messageEncKeyID:  mockMessageEncKeyID,
			messageEncKey:    mockMessageEncKey,
			webhookUrlEnv:    "https://localhost",
			webhookSecretEnv: "secret",
			appTimezone:      mockTimezone,
			expected: &Config{
				TelegramBotToken:       "abc",
				RuntimeEnv:             "production",
				DatabaseURL:            mockDBUrl,
				DeepseekAPIKey:         mockDeepseekAPIKey,
				OpenaiAPIKey:           mockOpenaiAPIKey,
				WebhookURL:             "https://localhost",
				WebhookSecret:          "secret",
				MessageEncryptionKeyID: mockMessageEncKeyID,
				MessageEncryptionKey:   mockMessageEncKey,
				AppTimezone:            mockTimezone,
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
			name:                       "missing msg enc key id in prod",
			runtimeEnv:                 "production",
			telegramEnv:                "abc",
			dbUrlEnv:                   mockDBUrl,
			deepseekAPIEnv:             mockDeepseekAPIKey,
			messageEncKey:              mockMessageEncKey,
			webhookUrlEnv:              "https://localhost",
			webhookSecretEnv:           "secret",
			expectMessageEncKeyIDError: true,
		},
		{
			name:                   "missing timezone",
			runtimeEnv:             "production",
			telegramEnv:            "abc",
			dbUrlEnv:               mockDBUrl,
			deepseekAPIEnv:         mockDeepseekAPIKey,
			messageEncKey:          mockMessageEncKey,
			messageEncKeyID:        mockMessageEncKeyID,
			webhookUrlEnv:          "https://localhost",
			webhookSecretEnv:       "secret",
			appTimezone:            "",
			expectAppTimezoneError: true,
		},
		{
			name:                   "invalid timezone",
			runtimeEnv:             "production",
			telegramEnv:            "abc",
			dbUrlEnv:               mockDBUrl,
			deepseekAPIEnv:         mockDeepseekAPIKey,
			messageEncKey:          mockMessageEncKey,
			messageEncKeyID:        mockMessageEncKeyID,
			webhookUrlEnv:          "https://localhost",
			webhookSecretEnv:       "secret",
			appTimezone:            "abc",
			expectAppTimezoneError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(goEnv, tt.runtimeEnv)
			t.Setenv(telegramBotTokenEnv, tt.telegramEnv)
			t.Setenv(webhookURLEnv, tt.webhookUrlEnv)
			t.Setenv(webhookSecretEnv, tt.webhookSecretEnv)
			t.Setenv(dbURLEnv, tt.dbUrlEnv)
			t.Setenv(deepseekKeyEnv, tt.deepseekAPIEnv)
			t.Setenv(openaiKeyEnv, tt.openaiAPIEnv)
			t.Setenv(messageEncryptionKeyEnv, tt.messageEncKey)
			t.Setenv(messageEncryptionKeyIDEnv, tt.messageEncKeyID)
			t.Setenv(appTimezoneEnv, tt.appTimezone)
			config, err := LoadConfig()
			if tt.expected == nil {
				require.Error(t, err)
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
				if tt.expectAppTimezoneError {
					assert.Contains(t, err.Error(), "Invalid Timezone")
				}
			} else {
				assert.Equal(t, config, tt.expected)
			}
		})
	}
}
