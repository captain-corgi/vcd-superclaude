package audit

import (
	"encoding/json"
	"strings"
)

// Redactor handles redaction of sensitive data from audit payloads
type Redactor struct {
	denyKeys map[string]bool
}

// NewRedactor creates a new redactor with the specified deny keys
func NewRedactor(denyKeys []string) *Redactor {
	denyMap := make(map[string]bool)
	for _, key := range denyKeys {
		denyMap[strings.ToLower(key)] = true
	}
	return &Redactor{denyKeys: denyMap}
}

// RedactPayload redacts sensitive keys from a payload
func (r *Redactor) RedactPayload(payload map[string]interface{}) (map[string]interface{}, bool) {
	if payload == nil {
		return payload, false
	}

	redacted := make(map[string]interface{})
	hasRedacted := false

	for key, value := range payload {
		if r.shouldRedactKey(key) {
			redacted[key] = "[REDACTED]"
			hasRedacted = true
		} else {
			// Recursively redact nested objects
			if nestedMap, ok := value.(map[string]interface{}); ok {
				redactedNested, nestedRedacted := r.RedactPayload(nestedMap)
				redacted[key] = redactedNested
				if nestedRedacted {
					hasRedacted = true
				}
			} else if nestedSlice, ok := value.([]interface{}); ok {
				redactedSlice, sliceRedacted := r.redactSlice(nestedSlice)
				redacted[key] = redactedSlice
				if sliceRedacted {
					hasRedacted = true
				}
			} else {
				redacted[key] = value
			}
		}
	}

	return redacted, hasRedacted
}

// redactSlice redacts sensitive keys from a slice of interfaces
func (r *Redactor) redactSlice(slice []interface{}) ([]interface{}, bool) {
	redacted := make([]interface{}, len(slice))
	hasRedacted := false

	for i, item := range slice {
		if nestedMap, ok := item.(map[string]interface{}); ok {
			redactedNested, nestedRedacted := r.RedactPayload(nestedMap)
			redacted[i] = redactedNested
			if nestedRedacted {
				hasRedacted = true
			}
		} else {
			redacted[i] = item
		}
	}

	return redacted, hasRedacted
}

// shouldRedactKey checks if a key should be redacted
func (r *Redactor) shouldRedactKey(key string) bool {
	return r.denyKeys[strings.ToLower(key)]
}

// RedactJSON redacts sensitive keys from a JSON string
func (r *Redactor) RedactJSON(jsonStr string) (string, bool, error) {
	if jsonStr == "" {
		return jsonStr, false, nil
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		return jsonStr, false, err
	}

	redacted, hasRedacted := r.RedactPayload(payload)

	redactedJSON, err := json.Marshal(redacted)
	if err != nil {
		return jsonStr, false, err
	}

	return string(redactedJSON), hasRedacted, nil
}
