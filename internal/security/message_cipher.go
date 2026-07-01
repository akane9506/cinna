package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	envelopePrefix  = "enc"
	envelopeVersion = "v1"
	aes256KeySize   = 32 // AES-256, requires exactly a 32-byte secret key
)

type MessageCipher struct {
	activeKeyID string
	keys        map[string]cipher.AEAD // a map of available keys ready for use
}

func NewMessageCipher(activeKeyID string, encodedKeys map[string]string) (*MessageCipher, error) {
	if activeKeyID == "" || strings.Contains(activeKeyID, ":") { // : will break our sealing logic
		return nil, errors.New("invalid active encryption key ID")
	}
	keys := make(map[string]cipher.AEAD, len(encodedKeys))
	for keyID, encodedKey := range encodedKeys {
		aead, err := parseAEAD(keyID, encodedKey)
		if err != nil {
			return nil, err
		}
		keys[keyID] = aead
	}
	// safety check, ensure cipher with action key actually exists
	if _, exists := keys[activeKeyID]; !exists {
		return nil, fmt.Errorf("active encryption key %q was not configured", activeKeyID)
	}
	return &MessageCipher{activeKeyID: activeKeyID, keys: keys}, nil
}

// encrypts message
func (c *MessageCipher) Encrypt(userID int64, plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("cannot encrypt an empty message")
	}
	aead := c.keys[c.activeKeyID]
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("error generating encryption nonce: %w", err)
	}
	// actual encryption
	// 1st nonce: Destination buffer, force go to write the cipher text directly after the nonce byte in memory
	// 2nd nonce: The actual initialization vector used for encryption math
	payload := aead.Seal(nonce, nonce, []byte(plaintext), messageAAD(userID))
	// encode the combines [nonce + ciphered text + auth tag] into an URL-safe base64 string
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	return fmt.Sprintf(
		"%s:%s:%s:%s", envelopePrefix, envelopeVersion, c.activeKeyID, encodedPayload), nil
}

// decrypts message
func (c *MessageCipher) Decrypt(userID int64, encodedMessage string) (string, error) {
	parts := strings.SplitN(encodedMessage, ":", 4)
	if len(parts) != 4 || parts[0] != envelopePrefix || parts[1] != envelopeVersion {
		return "", errors.New("invalid encrypted message format")
	}
	keyID, encodedPayload := parts[2], parts[3]
	aead, exists := c.keys[keyID]
	if !exists {
		return "", fmt.Errorf("unknown encryption key ID %q", keyID)
	}
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return "", errors.New("invalid encrypted message encoding")
	}
	nonceSize := aead.NonceSize()
	if len(payload) < nonceSize+aead.Overhead() {
		return "", errors.New("encrypted message is too short")
	}
	nonce, cipherText := payload[:nonceSize], payload[nonceSize:]
	plainText, err := aead.Open(nil, nonce, cipherText, messageAAD(userID))
	if err != nil {
		return "", errors.New("message authentication failed")
	}
	return string(plainText), nil
}

// ========== helper functions ==========

func parseAEAD(keyID, encodedKey string) (cipher.AEAD, error) {
	rawKey, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedKey))
	if err != nil {
		return nil, fmt.Errorf("error decoding encryption key %q: %w", keyID, err)
	}
	// strictly enforce AES-256
	if len(rawKey) != aes256KeySize {
		return nil, fmt.Errorf("encryption key %q must be exactly %d bytes", keyID, aes256KeySize)
	}
	// create the base AES cipher block compiler
	block, err := aes.NewCipher(rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher for key %q: %w", keyID, err)
	}
	// wrap the AES block in GCM to get authenticated encryption (AEAD)
	return cipher.NewGCM(block)
}

// additional authentication data
func messageAAD(userID int64) []byte {
	return strconv.AppendInt(nil, userID, 10) // format int64 into a slice without string allocations
}
