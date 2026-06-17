package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var mockDBUrl string = "postgres://a:a_pswrd@localhost:5432/cinna"
var mockDeepseekAPIKey string = "deepseek_api_key"

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                       string
		runtimeEnv                 string
		telegramEnv                string
		dbUrlEnv                   string
		deepseekAPIEnv             string
		webhookUrlEnv              string
		webhookSecretEnv           string
		expected                   *Config
		expectedRuntimeError       bool
		expectDBUrlError           bool
		expectTelegramError        bool
		expectedWebhookUrlError    bool
		expectedWebhookSecretError bool
	}{
		{
			name:           "success in development",
			telegramEnv:    "abc",
			dbUrlEnv:       mockDBUrl,
			deepseekAPIEnv: mockDeepseekAPIKey,
			expected: &Config{
				TelegramBotToken: "abc",
				DatabaseURL:      mockDBUrl,
				DeepseekAPIKey:   mockDeepseekAPIKey,
				RuntimeEnv:       "development",
			},
		},
		{
			name:             "success in production",
			runtimeEnv:       "production",
			telegramEnv:      "abc",
			dbUrlEnv:         mockDBUrl,
			deepseekAPIEnv:   mockDeepseekAPIKey,
			webhookUrlEnv:    "https://localhost",
			webhookSecretEnv: "secret",
			expected: &Config{
				TelegramBotToken: "abc",
				RuntimeEnv:       "production",
				DatabaseURL:      mockDBUrl,
				DeepseekAPIKey:   mockDeepseekAPIKey,
				WebhookURL:       "https://localhost",
				WebhookSecret:    "secret",
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
			name:                    "missing webhook in production",
			dbUrlEnv:                mockDBUrl,
			runtimeEnv:              "production",
			telegramEnv:             "abc",
			expectedWebhookUrlError: true,
		},
		{
			name:                       "missing websocket secret",
			dbUrlEnv:                   mockDBUrl,
			runtimeEnv:                 "production",
			telegramEnv:                "abc",
			webhookUrlEnv:              "https://localhost",
			expectedWebhookSecretError: true,
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
			config, err := LoadConfig()
			if tt.expected == nil {
				assert.Error(t, err)
				if tt.expectTelegramError {
					assert.Contains(t, err.Error(), telegramBotTokenEnv)
				}
				if tt.expectedWebhookUrlError {
					assert.Contains(t, err.Error(), webhookURLEnv)
				}
				if tt.expectedWebhookSecretError {
					assert.Contains(t, err.Error(), webhookSecretEnv)
				}
				if tt.expectedRuntimeError {
					assert.Contains(t, err.Error(), goEnv)
				}
				if tt.expectDBUrlError {
					assert.Contains(t, err.Error(), dbURLEnv)
				}
			} else {
				assert.Equal(t, config, tt.expected)
			}
		})
	}
}
