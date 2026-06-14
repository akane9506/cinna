package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                       string
		runtimeEnv                 string
		telegramEnv                string
		adminUsersEnv              string
		webhookUrlEnv              string
		webhookSecretEnv           string
		expected                   *Config
		expectedRuntimeError       bool
		expectTelegramError        bool
		expectAdminError           bool
		expectedWebhookUrlError    bool
		expectedWebhookSecretError bool
	}{
		{
			name:          "success in development",
			telegramEnv:   "abc",
			adminUsersEnv: "123,, 456, 789",
			expected: &Config{
				TelegramBotToken:  "abc",
				AllowedAdminUsers: []int64{123, 456, 789},
				RuntimeEnv:        "development",
			},
		},
		{
			name:             "success in production",
			runtimeEnv:       "production",
			telegramEnv:      "abc",
			adminUsersEnv:    "123,, 456, 789",
			webhookUrlEnv:    "https://localhost",
			webhookSecretEnv: "secret",
			expected: &Config{
				TelegramBotToken:  "abc",
				AllowedAdminUsers: []int64{123, 456, 789},
				RuntimeEnv:        "production",
				WebhookURL:        "https://localhost",
				WebhookSecret:     "secret",
			},
		},
		{
			name:                 "invalid runtime",
			runtimeEnv:           "wrong",
			expected:             nil,
			expectedRuntimeError: true,
		},
		{
			name:                "invalid telegram env",
			telegramEnv:         "",
			adminUsersEnv:       "123,, 456, 789",
			webhookUrlEnv:       "http://localhost",
			webhookSecretEnv:    "secret",
			expected:            nil,
			expectTelegramError: true,
		},
		{
			name:             "invalid admin users env",
			telegramEnv:      "abc",
			adminUsersEnv:    "name,, 456, 789",
			webhookUrlEnv:    "http://localhost",
			webhookSecretEnv: "secret",
			expected:         nil,
			expectAdminError: true,
		},
		{
			name:             "empty admin user",
			telegramEnv:      "abc",
			adminUsersEnv:    ",,",
			webhookUrlEnv:    "http://localhost",
			webhookSecretEnv: "secret",
			expected:         nil,
			expectAdminError: true,
		},
		{
			name:                    "missing webhook in production",
			runtimeEnv:              "production",
			telegramEnv:             "abc",
			adminUsersEnv:           "123,, 456, 789",
			expectedWebhookUrlError: true,
			expected:                nil,
		},
		{
			name:                       "missing websocket secret",
			runtimeEnv:                 "production",
			telegramEnv:                "abc",
			adminUsersEnv:              "123,, 456, 789",
			webhookUrlEnv:              "https://localhost",
			expected:                   nil,
			expectedWebhookSecretError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(goEnv, tt.runtimeEnv)
			t.Setenv(telegramBotTokenEnv, tt.telegramEnv)
			t.Setenv(allowedAdminUsersEnv, tt.adminUsersEnv)
			t.Setenv(webhookURLEnv, tt.webhookUrlEnv)
			t.Setenv(webhookSecretEnv, tt.webhookSecretEnv)
			config, err := LoadConfig()
			if tt.expected == nil {
				assert.Error(t, err)
				if tt.expectTelegramError {
					assert.Contains(t, err.Error(), telegramBotTokenEnv)
				}
				if tt.expectAdminError {
					assert.Contains(t, err.Error(), allowedAdminUsersEnv)
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
			} else {
				assert.Equal(t, config, tt.expected)
			}
		})
	}
}
