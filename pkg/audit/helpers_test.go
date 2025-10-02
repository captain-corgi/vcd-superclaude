package audit

import (
	"net/http"
	"testing"
)

func TestExtractActorInfo(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected ActorInfo
	}{
		{
			name: "with user headers",
			headers: map[string]string{
				"X-User-Id":    "user123",
				"X-User-Email": "user@example.com",
			},
			expected: ActorInfo{
				ID:    "user123",
				Email: "user@example.com",
			},
		},
		{
			name:    "without user headers",
			headers: map[string]string{},
			expected: ActorInfo{
				ID:    "system",
				Email: "",
			},
		},
		{
			name: "with only user id",
			headers: map[string]string{
				"X-User-Id": "user123",
			},
			expected: ActorInfo{
				ID:    "user123",
				Email: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := ExtractActorInfo(req)
			if result.ID != tt.expected.ID {
				t.Errorf("expected ID=%s, got %s", tt.expected.ID, result.ID)
			}
			if result.Email != tt.expected.Email {
				t.Errorf("expected Email=%s, got %s", tt.expected.Email, result.Email)
			}
		})
	}
}

func TestExtractSourceIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name: "with X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name: "with X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "fallback to RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.1:8080",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := ExtractSourceIP(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGenerateEventID(t *testing.T) {
	id1 := GenerateEventID()
	id2 := GenerateEventID()

	if id1 == id2 {
		t.Error("Generated event IDs should be unique")
	}

	if len(id1) != 32 {
		t.Errorf("Expected event ID length 32, got %d", len(id1))
	}
}
