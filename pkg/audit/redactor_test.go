package audit

import (
	"reflect"
	"testing"
)

func TestRedactor_RedactPayload(t *testing.T) {
	redactor := NewRedactor([]string{"password", "token", "secret"})

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
		redacted bool
	}{
		{
			name: "no sensitive data",
			input: map[string]interface{}{
				"username": "john",
				"email":    "john@example.com",
			},
			expected: map[string]interface{}{
				"username": "john",
				"email":    "john@example.com",
			},
			redacted: false,
		},
		{
			name: "password redaction",
			input: map[string]interface{}{
				"username": "john",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"username": "john",
				"password": "[REDACTED]",
			},
			redacted: true,
		},
		{
			name: "token redaction",
			input: map[string]interface{}{
				"user_id": "123",
				"token":   "abc123",
			},
			expected: map[string]interface{}{
				"user_id": "123",
				"token":   "[REDACTED]",
			},
			redacted: true,
		},
		{
			name: "nested object redaction",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name":     "john",
					"password": "secret123",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name":     "john",
					"password": "[REDACTED]",
				},
			},
			redacted: true,
		},
		{
			name: "case insensitive redaction",
			input: map[string]interface{}{
				"PASSWORD": "secret123",
				"Token":    "abc123",
			},
			expected: map[string]interface{}{
				"PASSWORD": "[REDACTED]",
				"Token":    "[REDACTED]",
			},
			redacted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, redacted := redactor.RedactPayload(tt.input)

			if redacted != tt.redacted {
				t.Errorf("expected redacted=%v, got %v", tt.redacted, redacted)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
