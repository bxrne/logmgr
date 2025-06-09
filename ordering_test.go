package logmgr

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestJSONFieldOrdering tests that JSON output has consistent field ordering
func TestJSONFieldOrdering(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	// Create an entry with multiple fields
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Message:   "test message",
		Fields:    make(map[string]interface{}),
	}

	// Add fields in non-alphabetical order to test sorting
	entry.Fields["zebra"] = "last"
	entry.Fields["alpha"] = "first"
	entry.Fields["beta"] = "second"
	entry.Fields["delta"] = "third"

	// Marshal to JSON
	jsonBytes, err := entry.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	jsonStr := string(jsonBytes)
	t.Logf("JSON output: %s", jsonStr)

	// Verify the ordering: level, timestamp, message, then custom fields alphabetically
	expectedOrder := []string{
		`"level":"info"`,
		`"timestamp":"2023-01-01T12:00:00Z"`,
		`"message":"test message"`,
		`"alpha":"first"`,
		`"beta":"second"`,
		`"delta":"third"`,
		`"zebra":"last"`,
	}

	// Check that each field appears in the correct order
	lastIndex := -1
	for i, expectedField := range expectedOrder {
		index := strings.Index(jsonStr, expectedField)
		if index == -1 {
			t.Errorf("Expected field not found: %s", expectedField)
			continue
		}
		if index <= lastIndex {
			t.Errorf("Field %d (%s) appears before field %d, violating ordering", i, expectedField, i-1)
		}
		lastIndex = index
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Errorf("Generated JSON is invalid: %v", err)
	}

	// Verify all expected fields are present
	if parsed["level"] != "info" {
		t.Errorf("Expected level 'info', got %v", parsed["level"])
	}
	if parsed["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", parsed["message"])
	}
	if parsed["alpha"] != "first" {
		t.Errorf("Expected alpha 'first', got %v", parsed["alpha"])
	}
}

// TestJSONFieldOrderingConsistency tests that field ordering is consistent across multiple calls
func TestJSONFieldOrderingConsistency(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	entry := &Entry{
		Level:     WarnLevel,
		Timestamp: time.Now(),
		Message:   "consistency test",
		Fields:    make(map[string]interface{}),
	}

	// Add many fields to increase chance of inconsistency if not properly sorted
	fields := map[string]interface{}{
		"user_id":    12345,
		"action":     "login",
		"ip_address": "192.168.1.1",
		"browser":    "Chrome",
		"version":    "1.0.0",
		"feature":    "auth",
		"success":    true,
		"duration":   123.45,
		"retry":      false,
		"session":    "sess-123",
	}

	for k, v := range fields {
		entry.Fields[k] = v
	}

	// Marshal multiple times and verify consistency
	var firstJSON string
	for i := 0; i < 10; i++ {
		jsonBytes, err := entry.MarshalJSON()
		if err != nil {
			t.Fatalf("Failed to marshal entry on iteration %d: %v", i, err)
		}

		jsonStr := string(jsonBytes)
		if i == 0 {
			firstJSON = jsonStr
		} else if jsonStr != firstJSON {
			t.Errorf("JSON output inconsistent on iteration %d", i)
			t.Logf("First:   %s", firstJSON)
			t.Logf("Current: %s", jsonStr)
			break
		}
	}
}

// TestJSONFieldOrderingWithoutCustomFields tests ordering when no custom fields are present
func TestJSONFieldOrderingWithoutCustomFields(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	entry := &Entry{
		Level:     ErrorLevel,
		Timestamp: time.Date(2023, 6, 15, 10, 30, 45, 123456789, time.UTC),
		Message:   "error occurred",
		Fields:    make(map[string]interface{}),
	}

	jsonBytes, err := entry.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	jsonStr := string(jsonBytes)
	t.Logf("JSON output: %s", jsonStr)

	// Should have exactly three fields in order
	expectedOrder := []string{
		`"level":"error"`,
		`"timestamp":"2023-06-15T10:30:45.123456789Z"`,
		`"message":"error occurred"`,
	}

	lastIndex := -1
	for i, expectedField := range expectedOrder {
		index := strings.Index(jsonStr, expectedField)
		if index == -1 {
			t.Errorf("Expected field not found: %s", expectedField)
			continue
		}
		if index <= lastIndex {
			t.Errorf("Field %d (%s) appears before field %d", i, expectedField, i-1)
		}
		lastIndex = index
	}

	// Should not contain any other fields
	fieldCount := strings.Count(jsonStr, `":`)
	if fieldCount != 3 {
		t.Errorf("Expected exactly 3 fields, found %d", fieldCount)
	}
}

// TestJSONFieldOrderingWithSpecialCharacters tests ordering with special characters in field names
func TestJSONFieldOrderingWithSpecialCharacters(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	entry := &Entry{
		Level:     DebugLevel,
		Timestamp: time.Now(),
		Message:   "special chars test",
		Fields:    make(map[string]interface{}),
	}

	// Add fields with special characters that should still sort alphabetically
	entry.Fields["_private"] = "underscore"
	entry.Fields["HTTP_STATUS"] = 200
	entry.Fields["api.version"] = "2.0"
	entry.Fields["user-agent"] = "test-agent"
	entry.Fields["123numeric"] = "starts with number"

	jsonBytes, err := entry.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	jsonStr := string(jsonBytes)
	t.Logf("JSON output: %s", jsonStr)

	// Verify alphabetical ordering of custom fields
	expectedCustomFieldOrder := []string{
		`"123numeric"`,
		`"HTTP_STATUS"`,
		`"_private"`,
		`"api.version"`,
		`"user-agent"`,
	}

	lastIndex := -1
	for i, expectedField := range expectedCustomFieldOrder {
		index := strings.Index(jsonStr, expectedField)
		if index == -1 {
			t.Errorf("Expected field not found: %s", expectedField)
			continue
		}
		if index <= lastIndex {
			t.Errorf("Custom field %d (%s) appears before field %d", i, expectedField, i-1)
		}
		lastIndex = index
	}
}

// TestJSONFieldOrderingWithDifferentValueTypes tests ordering with various value types
func TestJSONFieldOrderingWithDifferentValueTypes(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "value types test",
		Fields:    make(map[string]interface{}),
	}

	// Add fields with different value types
	entry.Fields["string_field"] = "text value"
	entry.Fields["int_field"] = 42
	entry.Fields["float_field"] = 3.14159
	entry.Fields["bool_field"] = true
	entry.Fields["nil_field"] = nil
	entry.Fields["array_field"] = []string{"a", "b", "c"}
	entry.Fields["map_field"] = map[string]string{"nested": "value"}

	jsonBytes, err := entry.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to marshal entry: %v", err)
	}

	jsonStr := string(jsonBytes)
	t.Logf("JSON output: %s", jsonStr)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Errorf("Generated JSON is invalid: %v", err)
	}

	// Verify alphabetical ordering of field names (regardless of value type)
	expectedFieldOrder := []string{
		"array_field",
		"bool_field",
		"float_field",
		"int_field",
		"map_field",
		"nil_field",
		"string_field",
	}

	// Extract field positions from JSON
	fieldPositions := make(map[string]int)
	for _, field := range expectedFieldOrder {
		quotedField := `"` + field + `"`
		index := strings.Index(jsonStr, quotedField)
		if index == -1 {
			t.Errorf("Expected field not found: %s", field)
			continue
		}
		fieldPositions[field] = index
	}

	// Verify ordering
	for i := 1; i < len(expectedFieldOrder); i++ {
		prevField := expectedFieldOrder[i-1]
		currField := expectedFieldOrder[i]

		prevPos, prevExists := fieldPositions[prevField]
		currPos, currExists := fieldPositions[currField]

		if prevExists && currExists && prevPos >= currPos {
			t.Errorf("Field %s (pos %d) should come before %s (pos %d)",
				prevField, prevPos, currField, currPos)
		}
	}
}
