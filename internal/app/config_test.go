package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name                string
		telegramEnv         string
		adminUsersEnv       string
		expected            *Config
		expectTelegramError bool
		expectAdminError    bool
	}{
		{
			name:          "success",
			telegramEnv:   "abc",
			adminUsersEnv: "123,, 456, 789",
			expected: &Config{
				TelegramBotToken:  "abc",
				AllowedAdminUsers: []int64{123, 456, 789},
			},
			expectTelegramError: false,
			expectAdminError:    false,
		},
		{
			name:                "invalid telegram env",
			telegramEnv:         "",
			adminUsersEnv:       "123,, 456, 789",
			expected:            nil,
			expectTelegramError: true,
			expectAdminError:    false,
		},
		{
			name:                "invalid admin users env",
			telegramEnv:         "abc",
			adminUsersEnv:       "name,, 456, 789",
			expected:            nil,
			expectTelegramError: false,
			expectAdminError:    true,
		},
		{
			name:                "empty admin user",
			telegramEnv:         "abc",
			adminUsersEnv:       ",,",
			expected:            nil,
			expectTelegramError: false,
			expectAdminError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(telegramBotTokenEnv, tt.telegramEnv)
			t.Setenv(allowedAdminUsersEnv, tt.adminUsersEnv)
			config, err := LoadConfig()
			if tt.expectAdminError || tt.expectTelegramError {
				assert.Error(t, err)
				if tt.expectTelegramError {
					assert.Contains(t, err.Error(), telegramBotTokenEnv)
				}
				if tt.expectAdminError {
					assert.Contains(t, err.Error(), allowedAdminUsersEnv)
				}
			} else {
				assert.Equal(t, config, tt.expected)
			}
		})
	}
}
