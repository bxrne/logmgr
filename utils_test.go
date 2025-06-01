package logmgr

import (
	"reflect"
	"testing"
	"time"
)

func TestField(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    interface{}
		expected LogField
	}{
		{
			name:     "string field",
			key:      "message",
			value:    "test message",
			expected: LogField{Key: "message", Value: "test message"},
		},
		{
			name:     "integer field",
			key:      "count",
			value:    42,
			expected: LogField{Key: "count", Value: 42},
		},
		{
			name:     "float field",
			key:      "percentage",
			value:    85.5,
			expected: LogField{Key: "percentage", Value: 85.5},
		},
		{
			name:     "boolean field",
			key:      "enabled",
			value:    true,
			expected: LogField{Key: "enabled", Value: true},
		},
		{
			name:     "nil field",
			key:      "optional",
			value:    nil,
			expected: LogField{Key: "optional", Value: nil},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Field(tt.key, tt.value)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Field() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEntryJSONMarshal(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)

	tests := []struct {
		name     string
		entry    *Entry
		expected string
	}{
		{
			name: "entry without fields",
			entry: &Entry{
				Level:     InfoLevel,
				Timestamp: timestamp,
				Message:   "test message",
				Fields:    map[string]interface{}{},
			},
			expected: `{"level":"info","timestamp":"2024-01-15T10:30:45.123456789Z","message":"test message"}`,
		},
		{
			name: "entry with single field",
			entry: &Entry{
				Level:     ErrorLevel,
				Timestamp: timestamp,
				Message:   "error occurred",
				Fields: map[string]interface{}{
					"error_code": "E001",
				},
			},
			expected: `{"level":"error","timestamp":"2024-01-15T10:30:45.123456789Z","message":"error occurred","error_code":"E001"}`,
		},
		{
			name: "entry with multiple fields",
			entry: &Entry{
				Level:     WarnLevel,
				Timestamp: timestamp,
				Message:   "warning message",
				Fields: map[string]interface{}{
					"user_id": 12345,
					"action":  "login",
					"success": true,
				},
			},
			// Note: field order in JSON may vary due to map iteration
			expected: `{"level":"warn","timestamp":"2024-01-15T10:30:45.123456789Z","message":"warning message"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.entry.MarshalJSON()
			if err != nil {
				t.Errorf("MarshalJSON() error = %v", err)
				return
			}

			resultStr := string(result)

			// For entries with multiple fields, just check that it starts correctly
			// since field order in maps is not guaranteed
			if tt.name == "entry with multiple fields" {
				if !contains(resultStr, `"level":"warn"`) ||
					!contains(resultStr, `"timestamp":"2024-01-15T10:30:45.123456789Z"`) ||
					!contains(resultStr, `"message":"warning message"`) ||
					!contains(resultStr, `"user_id":12345`) ||
					!contains(resultStr, `"action":"login"`) ||
					!contains(resultStr, `"success":true`) {
					t.Errorf("MarshalJSON() = %v, missing expected fields", resultStr)
				}
			} else {
				if resultStr != tt.expected {
					t.Errorf("MarshalJSON() = %v, want %v", resultStr, tt.expected)
				}
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{Level(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Level.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(4) // Small buffer for testing

	// Test empty buffer
	entries := make([]*Entry, 10)
	count := rb.Pop(entries)
	if count != 0 {
		t.Errorf("Pop() on empty buffer = %v, want 0", count)
	}

	// Test push and pop
	entry1 := &Entry{Message: "test1"}
	entry2 := &Entry{Message: "test2"}

	if !rb.Push(entry1) {
		t.Error("Push() failed on non-full buffer")
	}
	if !rb.Push(entry2) {
		t.Error("Push() failed on non-full buffer")
	}

	count = rb.Pop(entries)
	if count != 2 {
		t.Errorf("Pop() = %v, want 2", count)
	}

	if entries[0].Message != "test1" || entries[1].Message != "test2" {
		t.Error("Pop() returned incorrect entries")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
