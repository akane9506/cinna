package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIntent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected Intent
	}{
		{"valid shopping - lower", "shopping", IntentShopping},
		{"valid shopping - upper", "SHOPPING", IntentShopping},
		{"valid feedback - mixed", "Feedback", IntentFeedback},
		{"valid other - upper", "OTHER", IntentOther},
		{"invalid failed", "invalid", IntentFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			output := normalizeIntent(tt.input)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestNormalizeMethod(t *testing.T) {
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
