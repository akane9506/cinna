package core

import (
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeIntent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected Intent
	}{
		{"valid shopping - lower", "shopping", IntentShopping},
		{"valid shopping - upper", "SHOPPING", IntentShopping},
		{"valid feedback - mixed", "Feedback", IntentFeedback},
		{"valid other - upper", "OTHER", IntentOther},
		{"invalid failed", "invalid", IntentOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			output := normalizeIntent(tt.input)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestNormalizeMethod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected DBMethod
		error    error
	}{
		{"success_lower", "add", MethodAdd, nil},
		{"success_upper", "REMOVE", MethodRemove, nil},
		{"success_mixed", "Modify", MethodModify, nil},
		{"failed invalid", "some", "", errors.New("invalid method type: SOME")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			output, err := normalizeMethod(tt.input)
			if tt.error == nil {
				assert.Equal(t, tt.expected, output)
				assert.Nil(t, tt.error, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tt.error, err)
			}
		})
	}
}

func TestNormalizeShoppingCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected ShoppingCategory
	}{
		{"valid - lower", "grocery", ShoppingGrocery},
		{"valid - upper", "PHARMACY", ShoppingPharmacy},
		{"valid - mixed", "Pet", ShoppingPet},
		{"invalid", "invalid", ShoppingOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			output := normalizeShoppingCategory(tt.input)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestOrganizeInputMessage(t *testing.T) {
	t.Parallel()
	inputSystemMsg := "[INPUT SYSTEM]"
	tests := []struct {
		name          string
		input         []*schema.Message
		numSystemMsgs int
	}{
		{
			name: "with system message",
			input: []*schema.Message{
				schema.SystemMessage(inputSystemMsg),
				schema.AssistantMessage("assistant message", nil),
				schema.UserMessage("user message"),
			},
			numSystemMsgs: 1,
		},
		{
			name: "with more input system messages",
			input: []*schema.Message{
				schema.SystemMessage(inputSystemMsg),
				schema.AssistantMessage("assistant message", nil),
				schema.SystemMessage(inputSystemMsg),
				schema.UserMessage("user message"),
				schema.SystemMessage(inputSystemMsg),
			},
			numSystemMsgs: 3,
		},
		{
			name: "without system message",
			input: []*schema.Message{
				schema.AssistantMessage("assistant message", nil),
				schema.UserMessage("user message"),
			},
			numSystemMsgs: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			currSystemPrompt := "new system prompt"
			output := organizeInputMessage(tt.input, currSystemPrompt)
			for _, msg := range output {
				if msg.Role == schema.System {
					assert.NotEqual(t, msg.Content, inputSystemMsg)
				}
			}
			assert.Equal(t, len(tt.input)-tt.numSystemMsgs+1, len(output))
		})
	}
}

func TestOrganizeInputMessageWithoutSensitiveInfo(t *testing.T) {
	t.Parallel()
	inputSystemMsg := "[INPUT SYSTEM]"
	tests := []struct {
		name             string
		input            []*schema.Message
		numSystemMsgs    int
		numSensitiveMsgs int
	}{
		{
			name: "with system message and sensitive message",
			input: []*schema.Message{
				schema.SystemMessage(inputSystemMsg),
				schema.AssistantMessage(SENSITIVE_PREFIX+"assistant message", nil),
				schema.UserMessage("user message"),
			},
			numSystemMsgs:    1,
			numSensitiveMsgs: 1,
		},
		{
			name: "with more input system and sensitive messages",
			input: []*schema.Message{
				schema.SystemMessage(inputSystemMsg),
				schema.AssistantMessage("assistant message", nil),
				schema.SystemMessage(inputSystemMsg),
				schema.UserMessage("user message"),
				schema.UserMessage(SENSITIVE_PREFIX + "user message"),
				schema.SystemMessage(inputSystemMsg),
				schema.AssistantMessage(SENSITIVE_PREFIX+"assistant message", nil),
			},
			numSystemMsgs:    3,
			numSensitiveMsgs: 2,
		},
		{
			name: "without system and sensitive message",
			input: []*schema.Message{
				schema.AssistantMessage("assistant message", nil),
				schema.UserMessage("user message"),
			},
			numSystemMsgs:    0,
			numSensitiveMsgs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			currSystemPrompt := "new system prompt"
			output := organizeInputMessageWithoutSensitiveInfo(tt.input, currSystemPrompt)
			for _, msg := range output {
				if msg.Role == schema.System {
					assert.NotEqual(t, msg.Content, inputSystemMsg)
				}
				assert.NotContains(t, msg.Content, SENSITIVE_PREFIX)
			}
			assert.Equal(t, len(tt.input)-tt.numSystemMsgs-tt.numSensitiveMsgs+1, len(output))
		})
	}
}

func TestOrganizeCompressionMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		input           []*schema.Message
		prompt          string
		expectedHistory string
	}{
		{
			name: "formats user and assistant messages",
			input: []*schema.Message{
				schema.UserMessage("Hello"),
				schema.AssistantMessage("Hi, how can I help?", nil),
				schema.UserMessage("Tell me a joke"),
			},
			prompt: "Compress the conversation",
			expectedHistory: "[USER]\n" +
				"Hello\n\n" +
				"[ASSISTANT]\n" +
				"Hi, how can I help?\n\n" +
				"[USER]\n" +
				"Tell me a joke\n\n",
		},
		{
			name: "skips system and sensitive messages",
			input: []*schema.Message{
				schema.SystemMessage("Original system message"),
				schema.UserMessage("Visible user message"),
				schema.AssistantMessage(SENSITIVE_PREFIX+"hidden assistant message", nil),
				schema.UserMessage(SENSITIVE_PREFIX + "hidden user message"),
				schema.AssistantMessage("Visible assistant message", nil),
			},
			prompt: "Compress the conversation",
			expectedHistory: "[USER]\n" +
				"Visible user message\n\n" +
				"[ASSISTANT]\n" +
				"Visible assistant message\n\n",
		},
		{
			name: "only skips sensitive prefix at the beginning",
			input: []*schema.Message{
				schema.UserMessage("This message contains " + SENSITIVE_PREFIX + " in the middle"),
				schema.AssistantMessage(SENSITIVE_PREFIX+"hidden message", nil),
			},
			prompt: "Compress the conversation",
			expectedHistory: "[USER]\n" +
				"This message contains " + SENSITIVE_PREFIX + " in the middle\n\n",
		},
		{
			name:            "empty input",
			input:           nil,
			prompt:          "Compress the conversation",
			expectedHistory: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			output := organizeCompressionMessage(tt.input, tt.prompt)

			if assert.Len(t, output, 2) {
				assert.Equal(t, schema.System, output[0].Role)
				assert.Equal(t, tt.prompt, output[0].Content)

				assert.Equal(t, schema.User, output[1].Role)
				assert.Equal(t, tt.expectedHistory, output[1].Content)
			}
		})
	}
}
