package security

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testKeyID  = "test"
	testKey    = "SknWqc4U2KduI/myleg9aHF1XV0U/iG/kdn/i1QwP14="
	testUserID = 123456
)

func TestCreateMessageCipher(t *testing.T) {
	tests := []struct {
		name        string
		activeKeyID string
		encodedKeys map[string]string
		expectedErr bool
	}{
		{
			name:        "success",
			activeKeyID: testKeyID,
			encodedKeys: map[string]string{testKeyID: testKey},
		},
		{
			name:        "failed with invalid key id",
			activeKeyID: "",
			encodedKeys: map[string]string{testKeyID: testKey},
			expectedErr: true,
		},
		{
			name:        "non-existing key id",
			activeKeyID: "invalid key",
			encodedKeys: map[string]string{testKeyID: testKey},
			expectedErr: true,
		},
		{
			name:        "invalid key 1",
			activeKeyID: testKeyID,
			encodedKeys: map[string]string{
				testKeyID: "abcde",
				"empty":   "",
			},
			expectedErr: true,
		},
		{
			name:        "invalid key 2",
			activeKeyID: testKeyID,
			encodedKeys: map[string]string{"empty": ""},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			messageCipher, err := NewMessageCipher(tt.activeKeyID, tt.encodedKeys)
			if !tt.expectedErr {
				assert.Equal(t, messageCipher.activeKeyID, tt.activeKeyID)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	keys := map[string]string{testKeyID: testKey}
	messageCipher, err := NewMessageCipher(testKeyID, keys)
	assert.Nil(t, err)
	tests := []struct {
		name                 string
		text                 string
		errorEncrypt         bool
		mockEncryptedMessage string
		errorDecrypt         bool
	}{
		{
			name: "success",
			text: "this is a test message",
		},
		{
			name: "success_mandarin",
			text: "这是一条测试信息",
		},
		{
			name:         "failed encryption",
			text:         "",
			errorEncrypt: true,
		},
		{
			name:                 "failed decryption-invalid msg",
			mockEncryptedMessage: "a:b:c",
			errorDecrypt:         true,
		},
		{
			name:                 "failed decryption-invalid key",
			mockEncryptedMessage: fmt.Sprintf("%s:%s:c:d", envelopePrefix, envelopeVersion),
			errorDecrypt:         true,
		},
		{
			name:                 "failed decryption-short payload",
			mockEncryptedMessage: fmt.Sprintf("%s:%s:%s:d", envelopePrefix, envelopeVersion, testKeyID),
			errorDecrypt:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			encrypted, err := messageCipher.Encrypt(testUserID, tt.text)
			if tt.errorEncrypt {
				assert.Error(t, err)
				return
			}
			decryptMsg := encrypted
			if tt.errorDecrypt {
				decryptMsg = tt.mockEncryptedMessage
			}
			decrypted, err := messageCipher.Decrypt(testUserID, decryptMsg)
			if tt.errorDecrypt {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, tt.text, decrypted)
		})
	}
}
