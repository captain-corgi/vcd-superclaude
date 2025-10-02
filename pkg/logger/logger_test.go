package logger

import (
	"testing"
)

func TestNewFromConfig(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		format      string
		outputs     []string
		filePath    string
		development bool
		shouldPanic bool
	}{
		{
			name:        "valid config",
			level:       "info",
			format:      "json",
			outputs:     []string{"stdout"},
			filePath:    "",
			development: false,
			shouldPanic: false,
		},
		{
			name:        "console format",
			level:       "debug",
			format:      "console",
			outputs:     []string{"stdout"},
			filePath:    "",
			development: true,
			shouldPanic: false,
		},
		{
			name:        "file output",
			level:       "warn",
			format:      "json",
			outputs:     []string{"file"},
			filePath:    "test.log",
			development: false,
			shouldPanic: false,
		},
		{
			name:        "multiple outputs",
			level:       "error",
			format:      "json",
			outputs:     []string{"stdout", "file"},
			filePath:    "test.log",
			development: false,
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			logger := NewFromConfig(tt.level, tt.format, tt.outputs, tt.filePath, tt.development)

			if logger == nil {
				t.Error("Expected logger to be created")
			}
		})
	}
}
