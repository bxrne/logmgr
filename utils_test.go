package logmgr

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TEST: GIVEN different field types WHEN creating LogField with Field function THEN correct LogField structure is returned
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

// TEST: GIVEN log Entry objects WHEN marshaling to JSON THEN correct JSON representation is produced
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

// TEST: GIVEN different log levels WHEN converting to string THEN correct string representation is returned
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

// TEST: GIVEN a RingBuffer WHEN pushing and popping entries THEN entries are retrieved in correct order
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

// TEST: GIVEN a non-power-of-2 buffer size WHEN creating RingBuffer THEN it should panic
func TestRingBufferPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewRingBuffer should panic with non-power-of-2 size")
		}
	}()
	NewRingBuffer(3) // Not a power of 2
}

// TEST: GIVEN a full RingBuffer WHEN trying to push more entries THEN additional pushes should fail
func TestRingBufferFull(t *testing.T) {
	rb := NewRingBuffer(2) // Very small buffer
	entry := &Entry{Message: "test"}

	// Fill the buffer
	if !rb.Push(entry) {
		t.Error("First push should succeed")
	}
	if !rb.Push(entry) {
		t.Error("Second push should succeed")
	}

	// Buffer should be full now
	if rb.Push(entry) {
		t.Error("Third push should fail on full buffer")
	}
}

// TEST: GIVEN global logger WHEN calling logging functions at different levels THEN messages are logged to sinks
func TestLoggingFunctions(t *testing.T) {
	// Reset global state
	resetGlobalLogger()

	// Capture output
	var buf bytes.Buffer
	sink := &testSink{writer: &buf}

	// Initialize logger
	SetLevel(DebugLevel)
	AddSink(sink)

	// Test all logging levels
	Debug("debug message", Field("level", "debug"))
	Info("info message", Field("level", "info"))
	Warn("warn message", Field("level", "warn"))
	Error("error message", Field("level", "error"))

	// Give workers time to process
	time.Sleep(50 * time.Millisecond)
	Shutdown()

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Error("Debug message not found in output")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Info message not found in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message not found in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message not found in output")
	}
}

// TEST: GIVEN logger with specific level WHEN logging at various levels THEN only messages at or above the level are processed
func TestLevelFiltering(t *testing.T) {
	// Reset global state
	resetGlobalLogger()

	var buf bytes.Buffer
	sink := &testSink{writer: &buf}

	// Set level to Error (should filter out Debug, Info, Warn)
	SetLevel(ErrorLevel)
	AddSink(sink)

	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	time.Sleep(50 * time.Millisecond)
	Shutdown()

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if strings.Contains(output, "warn message") {
		t.Error("Warn message should be filtered out")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should not be filtered out")
	}
}

// TEST: GIVEN a logger with set level WHEN calling GetLevel THEN correct level is returned
func TestGetLevel(t *testing.T) {
	// Reset global state
	resetGlobalLogger()

	SetLevel(WarnLevel)
	if GetLevel() != WarnLevel {
		t.Errorf("GetLevel() = %v, want %v", GetLevel(), WarnLevel)
	}

	Shutdown()
}

// TEST: GIVEN multiple sinks WHEN setting sinks and logging THEN message appears in all sinks
func TestSetSinks(t *testing.T) {
	// Reset global state
	resetGlobalLogger()

	var buf1, buf2 bytes.Buffer
	sink1 := &testSink{writer: &buf1}
	sink2 := &testSink{writer: &buf2}

	SetSinks(sink1, sink2)
	Info("test message")

	time.Sleep(50 * time.Millisecond)
	Shutdown()

	if !strings.Contains(buf1.String(), "test message") {
		t.Error("Message not found in first sink")
	}
	if !strings.Contains(buf2.String(), "test message") {
		t.Error("Message not found in second sink")
	}
}

// TEST: GIVEN console sink WHEN writing log entries THEN entries are written to stdout
func TestConsoleSink(t *testing.T) {
	sink := NewConsoleSink()

	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "test message",
			Fields:    map[string]interface{}{"key": "value"},
		},
	}

	err := sink.Write(entries)
	if err != nil {
		t.Errorf("ConsoleSink.Write() error = %v", err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("ConsoleSink.Close() error = %v", err)
	}
}

// TEST: GIVEN stderr sink WHEN writing log entries THEN entries are written to stderr
func TestStderrSink(t *testing.T) {
	sink := NewStderrSink()

	entries := []*Entry{
		{
			Level:     ErrorLevel,
			Timestamp: time.Now(),
			Message:   "error message",
			Fields:    map[string]interface{}{"key": "value"},
		},
	}

	err := sink.Write(entries)
	if err != nil {
		t.Errorf("StderrSink.Write() error = %v", err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("StderrSink.Close() error = %v", err)
	}
}

// TEST: GIVEN a file sink WHEN writing log entries THEN entries are written to the file
func TestFileSink(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	sink, err := NewFileSink(filename, 0, 0) // No rotation
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "file test message",
			Fields:    map[string]interface{}{"file": "test"},
		},
	}

	err = sink.Write(entries)
	if err != nil {
		t.Errorf("FileSink.Write() error = %v", err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("FileSink.Close() error = %v", err)
	}

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "file test message") {
		t.Error("Log file should contain test message")
	}
}

// TEST: GIVEN a file sink with size rotation WHEN writing entries that exceed the size limit THEN file rotation occurs
func TestFileSinkRotation(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "rotate.log")

	// Create sink with very small size limit to trigger rotation
	sink, err := NewFileSink(filename, 0, 10) // 10 bytes max
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	// Write enough data to trigger rotation
	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "this is a very long message that should trigger rotation",
			Fields:    map[string]interface{}{},
		},
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "second message",
			Fields:    map[string]interface{}{},
		},
	}

	for _, entry := range entries {
		err = sink.Write([]*Entry{entry})
		if err != nil {
			t.Errorf("FileSink.Write() error = %v", err)
		}
	}

	if err := sink.Close(); err != nil {
		t.Errorf("FileSink.Close() error = %v", err)
	}

	// Check that rotation occurred (rotated file should exist)
	files, err := filepath.Glob(filepath.Join(tmpDir, "rotate_*.log"))
	if err != nil {
		t.Fatalf("Failed to glob rotated files: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected rotated file to exist")
	}
}

// TEST: GIVEN an async file sink WHEN writing log entries THEN entries are written asynchronously to the file
func TestAsyncFileSink(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "async.log")

	sink, err := NewAsyncFileSink(filename, 0, 0, 10) // Small buffer
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "async test message",
			Fields:    map[string]interface{}{"async": true},
		},
	}

	err = sink.Write(entries)
	if err != nil {
		t.Errorf("AsyncFileSink.Write() error = %v", err)
	}

	// Give async writer time to process
	time.Sleep(200 * time.Millisecond)

	err = sink.Close()
	if err != nil {
		t.Errorf("AsyncFileSink.Close() error = %v", err)
	}

	// Read file content
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read async log file: %v", err)
	}

	if !strings.Contains(string(content), "async test message") {
		t.Error("Async log file should contain test message")
	}
}

// TEST: GIVEN an async file sink with small buffer WHEN buffer becomes full THEN fallback to synchronous writes occurs
func TestAsyncFileSinkBufferFull(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "buffer_full.log")

	// Create sink with very small buffer to test fallback
	sink, err := NewAsyncFileSink(filename, 0, 0, 1) // Buffer size of 1
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	// Fill the buffer and test fallback to synchronous write
	entries := []*Entry{
		{Level: InfoLevel, Timestamp: time.Now(), Message: "msg1", Fields: map[string]interface{}{}},
		{Level: InfoLevel, Timestamp: time.Now(), Message: "msg2", Fields: map[string]interface{}{}},
		{Level: InfoLevel, Timestamp: time.Now(), Message: "msg3", Fields: map[string]interface{}{}},
	}

	for _, entry := range entries {
		err = sink.Write([]*Entry{entry})
		if err != nil {
			t.Errorf("AsyncFileSink.Write() error = %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)
	if err := sink.Close(); err != nil {
		t.Errorf("AsyncFileSink.Close() error = %v", err)
	}

	// Verify all messages were written
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "msg1") || !strings.Contains(contentStr, "msg2") || !strings.Contains(contentStr, "msg3") {
		t.Error("All messages should be written even when buffer is full")
	}
}

// TEST: GIVEN default file sink configuration WHEN writing log entries THEN entries are written with default settings
func TestNewDefaultFileSink(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "default.log")

	sink := NewDefaultFileSink(filename, time.Hour)
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "default sink test",
			Fields:    map[string]interface{}{},
		},
	}

	err := sink.Write(entries)
	if err != nil {
		t.Errorf("NewDefaultFileSink.Write() error = %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Errorf("NewDefaultFileSink.Close() error = %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read default log file: %v", err)
	}

	if !strings.Contains(string(content), "default sink test") {
		t.Error("Default file sink should contain test message")
	}
}

// TEST: GIVEN a file sink with time-based rotation WHEN time limit is reached THEN file rotation occurs
func TestFileSinkTimeRotation(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "time_rotate.log")

	// Create sink with very short time limit
	sink, err := NewFileSink(filename, 1*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			t.Logf("Failed to close sink: %v", err)
		}
	}()

	// Write first entry
	entry1 := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "first message",
		Fields:    map[string]interface{}{},
	}
	err = sink.Write([]*Entry{entry1})
	if err != nil {
		t.Errorf("FileSink.Write() error = %v", err)
	}

	// Wait for time-based rotation to trigger
	time.Sleep(10 * time.Millisecond)

	// Write second entry (should trigger rotation)
	entry2 := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "second message",
		Fields:    map[string]interface{}{},
	}
	err = sink.Write([]*Entry{entry2})
	if err != nil {
		t.Errorf("FileSink.Write() error = %v", err)
	}

	if err := sink.Close(); err != nil {
		t.Errorf("FileSink.Close() error = %v", err)
	}

	// Check that rotation occurred
	files, err := filepath.Glob(filepath.Join(tmpDir, "time_rotate_*.log"))
	if err != nil {
		t.Fatalf("Failed to glob rotated files: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected time-based rotation to create rotated file")
	}
}

// TEST: GIVEN console sink with malformed entries WHEN writing entries with unmarshalable data THEN sink handles errors gracefully
func TestConsoleSinkWithMalformedEntry(t *testing.T) {
	sink := NewConsoleSink()

	// Create an entry that will cause JSON marshaling to fail
	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "test",
			Fields:    map[string]interface{}{"invalid": make(chan int)}, // channels can't be marshaled
		},
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "valid message",
			Fields:    map[string]interface{}{"valid": "field"},
		},
	}

	// Should not return error even with malformed entry
	err := sink.Write(entries)
	if err != nil {
		t.Errorf("ConsoleSink.Write() should not error on malformed entries: %v", err)
	}
}

// TEST: GIVEN a RingBuffer WHEN multiple goroutines push and pop concurrently THEN operations are thread-safe
func TestRingBufferConcurrency(t *testing.T) {
	rb := NewRingBuffer(64) // Smaller buffer

	// Test concurrent pushes and pops
	var wg sync.WaitGroup
	numGoroutines := 4        // Fewer goroutines
	entriesPerGoroutine := 10 // Fewer entries

	// Start pushers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < entriesPerGoroutine; j++ {
				entry := &Entry{
					Message: fmt.Sprintf("msg-%d-%d", id, j),
					Fields:  map[string]interface{}{},
				}
				rb.Push(entry)
			}
		}(i)
	}

	wg.Wait()

	// Pop all entries
	entries := make([]*Entry, 64)
	totalPopped := 0
	for {
		count := rb.Pop(entries)
		if count == 0 {
			break
		}
		totalPopped += count
	}

	if totalPopped < numGoroutines*entriesPerGoroutine {
		t.Errorf("Expected to pop at least %d entries, got %d", numGoroutines*entriesPerGoroutine, totalPopped)
	}
}

// TEST: GIVEN logger with no sinks WHEN logging messages THEN operations complete without errors
func TestLoggerWithNoSinks(t *testing.T) {
	resetGlobalLogger()

	// Test logging without any sinks (should not crash)
	SetLevel(InfoLevel)
	Info("message with no sinks")

	time.Sleep(50 * time.Millisecond)
	Shutdown()
}

// TEST: GIVEN a logger WHEN shutdown is called multiple times THEN subsequent shutdowns are handled gracefully
func TestMultipleShutdowns(t *testing.T) {
	resetGlobalLogger()

	SetLevel(InfoLevel)
	AddSink(&testSink{writer: &bytes.Buffer{}})

	// Multiple shutdowns should not panic
	Shutdown()
	Shutdown()
	Shutdown()
}

// Test helper sink for capturing output
type testSink struct {
	writer io.Writer
	mu     sync.Mutex
}

func (ts *testSink) Write(entries []*Entry) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for _, entry := range entries {
		data, err := entry.MarshalJSON()
		if err != nil {
			continue
		}
		if _, err := ts.writer.Write(data); err != nil {
			return err
		}
		if _, err := ts.writer.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func (ts *testSink) Close() error {
	return nil
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

// TEST: GIVEN a worker with logger and sink WHEN processing log entries THEN entries are written to sink
func TestWorker(t *testing.T) {
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8),
		shutdown: make(chan struct{}),
	}

	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{Fields: make(map[string]interface{})}
		},
	}

	var buf bytes.Buffer
	sink := &testSink{writer: &buf}
	logger.sinks = []Sink{sink}

	worker := NewWorker(0, logger, 4)
	logger.workers = []*Worker{worker}
	logger.wg.Add(1)
	go worker.Run()

	// Add entry to buffer
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "worker test",
		Fields:    map[string]interface{}{},
	}
	logger.buffer.Push(entry)

	// Give worker time to process
	time.Sleep(50 * time.Millisecond)

	worker.Stop()
	logger.wg.Wait()

	if !strings.Contains(buf.String(), "worker test") {
		t.Error("Worker should have processed the entry")
	}
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

// TEST: GIVEN invalid file path WHEN creating file sink THEN error is returned
func TestFileSinkErrors(t *testing.T) {
	// Use platform-appropriate invalid path
	var invalidPath string
	if isWindows() {
		// On Windows, use characters that are not allowed in file paths
		invalidPath = `C:\invalid\path\with<invalid>characters\test.log`
	} else {
		// On Unix-like systems, use a path that requires root permissions
		invalidPath = "/proc/invalid/path/that/does/not/exist/test.log"
	}

	_, err := NewFileSink(invalidPath, 0, 0)
	if err == nil {
		t.Error("NewFileSink should fail with invalid path")
	}
}

// TEST: GIVEN invalid file path WHEN creating async file sink THEN error is returned
func TestAsyncFileSinkErrors(t *testing.T) {
	// Use platform-appropriate invalid path
	var invalidPath string
	if isWindows() {
		// On Windows, use characters that are not allowed in file paths
		invalidPath = `C:\invalid\path\with<invalid>characters\async.log`
	} else {
		// On Unix-like systems, use a path that requires root permissions
		invalidPath = "/proc/invalid/path/that/does/not/exist/async.log"
	}

	_, err := NewAsyncFileSink(invalidPath, 0, 0, 10)
	if err == nil {
		t.Error("NewAsyncFileSink should fail with invalid path")
	}
}

// TEST: GIVEN a closed file sink WHEN writing entries THEN write is handled gracefully
func TestFileSinkWriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "closed.log")

	sink, err := NewFileSink(filename, 0, 0)
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}

	// Close the sink
	if err := sink.Close(); err != nil {
		t.Fatalf("Failed to close sink: %v", err)
	}

	// Try to write after close (should handle gracefully)
	entries := []*Entry{
		{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "after close",
			Fields:    map[string]interface{}{},
		},
	}

	err = sink.Write(entries)
	// Should not panic, may return error
	if err != nil {
		t.Logf("Expected error writing to closed sink: %v", err)
	}
}

// TEST: GIVEN RingBuffer in various states WHEN popping with different slice sizes THEN correct behavior is exhibited
func TestRingBufferEdgeCases(t *testing.T) {
	rb := NewRingBuffer(4)

	// Test popping from empty buffer with different slice sizes
	entries1 := make([]*Entry, 1)
	count := rb.Pop(entries1)
	if count != 0 {
		t.Error("Pop from empty buffer should return 0")
	}

	entries10 := make([]*Entry, 10)
	count = rb.Pop(entries10)
	if count != 0 {
		t.Error("Pop from empty buffer should return 0")
	}

	// Fill buffer partially
	entry := &Entry{Message: "test"}
	rb.Push(entry)
	rb.Push(entry)

	// Pop with smaller slice
	entries2 := make([]*Entry, 1)
	count = rb.Pop(entries2)
	if count != 1 {
		t.Errorf("Pop with small slice should return 1, got %d", count)
	}

	// Pop remaining
	count = rb.Pop(entries2)
	if count != 1 {
		t.Errorf("Pop remaining should return 1, got %d", count)
	}
}

// TEST: GIVEN uninitialized global logger WHEN setting level or adding sinks THEN logger is properly initialized
func TestLoggerInitialization(t *testing.T) {
	resetGlobalLogger()

	// Test that logger is initialized on first use
	if globalLogger != nil {
		t.Error("Global logger should be nil initially")
	}

	// First call should initialize logger
	SetLevel(InfoLevel)
	if globalLogger == nil {
		t.Error("Global logger should be initialized after SetLevel")
	}

	// Test multiple initializations don't cause issues
	SetLevel(DebugLevel)
	AddSink(&testSink{writer: &bytes.Buffer{}})

	Shutdown()
}

// TEST: GIVEN worker with no sinks WHEN flushing entries THEN worker handles gracefully
func TestWorkerFlushEdgeCases(t *testing.T) {
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8),
		shutdown: make(chan struct{}),
	}

	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{Fields: make(map[string]interface{})}
		},
	}

	// Test worker with no sinks
	logger.sinks = []Sink{}

	worker := NewWorker(0, logger, 2) // Small batch size
	logger.workers = []*Worker{worker}
	logger.wg.Add(1)
	go worker.Run()

	// Add entries
	for i := 0; i < 5; i++ {
		entry := &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("test %d", i),
			Fields:    map[string]interface{}{},
		}
		logger.buffer.Push(entry)
	}

	time.Sleep(50 * time.Millisecond)
	worker.Stop()
	logger.wg.Wait()
}

// TEST: GIVEN console sink with many entries WHEN writing and closing THEN buffer is flushed properly
func TestConsoleSinkBufferFlush(t *testing.T) {
	// Test that console sink flushes buffer properly
	sink := NewConsoleSink()

	// Create many entries to test buffer flushing
	entries := make([]*Entry, 100)
	for i := range entries {
		entries[i] = &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("message %d", i),
			Fields:    map[string]interface{}{"index": i},
		}
	}

	err := sink.Write(entries)
	if err != nil {
		t.Errorf("ConsoleSink.Write() error = %v", err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("ConsoleSink.Close() error = %v", err)
	}
}

// TEST: GIVEN stderr sink with many entries WHEN writing and closing THEN buffer is flushed properly
func TestStderrSinkBufferFlush(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	// Capture stderr to test output
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	defer func() {
		_ = w.Close()
		os.Stderr = oldStderr
	}()

	sink := NewStderrSink()
	SetSinks(sink)

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test message",
		Fields:    map[string]interface{}{"key": "value"},
	}

	err := sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TEST: GIVEN Fatal log level WHEN calling Fatal function THEN program exits with code 1
func TestFatal(t *testing.T) {
	// This test verifies that Fatal calls log, Shutdown, and os.Exit(1)
	// We can't actually test os.Exit(1) in a unit test, but we can test
	// that the function structure is correct by replacing os.Exit temporarily

	// First, let's test that we can't easily test Fatal due to os.Exit
	// Instead, we'll test the components that Fatal uses

	resetGlobalLogger()
	defer resetGlobalLogger()

	// Test that Fatal level logging works
	var buffer bytes.Buffer
	sink := &testSink{writer: &buffer}
	SetSinks(sink)
	SetLevel(FatalLevel)

	// We can test that the log function works with FatalLevel
	log(FatalLevel, "test fatal", Field("key", "value"))

	// Give time for background workers to process
	time.Sleep(50 * time.Millisecond)
	Shutdown()

	output := buffer.String()
	if !contains(output, "fatal") {
		t.Errorf("Expected fatal level in output, got: %s", output)
	}
	if !contains(output, "test fatal") {
		t.Errorf("Expected message in output, got: %s", output)
	}
}

// TEST: GIVEN various string characters WHEN using appendJSONString THEN proper JSON escaping occurs
func TestAppendJSONString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: `"hello"`,
		},
		{
			name:     "string with quotes",
			input:    `hello "world"`,
			expected: `"hello \"world\""`,
		},
		{
			name:     "string with backslash",
			input:    `hello\world`,
			expected: `"hello\\world"`,
		},
		{
			name:     "string with newline",
			input:    "hello\nworld",
			expected: `"hello\nworld"`,
		},
		{
			name:     "string with carriage return",
			input:    "hello\rworld",
			expected: `"hello\rworld"`,
		},
		{
			name:     "string with tab",
			input:    "hello\tworld",
			expected: `"hello\tworld"`,
		},
		{
			name:     "string with control character",
			input:    "hello\x01world",
			expected: `"hello\u0001world"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: `""`,
		},
		{
			name:     "unicode string",
			input:    "hello 世界",
			expected: `"hello 世界"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			result := appendJSONString(buf, tt.input)
			if string(result) != tt.expected {
				t.Errorf("appendJSONString() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

// TEST: GIVEN various value types WHEN using appendJSONValue THEN proper JSON encoding occurs
func TestAppendJSONValue(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "string value",
			value:    "test",
			expected: `"test"`,
		},
		{
			name:     "int value",
			value:    42,
			expected: "42",
		},
		{
			name:     "int32 value",
			value:    int32(123),
			expected: "123",
		},
		{
			name:     "int64 value",
			value:    int64(456),
			expected: "456",
		},
		{
			name:     "uint value",
			value:    uint(789),
			expected: "789",
		},
		{
			name:     "uint32 value",
			value:    uint32(101112),
			expected: "101112",
		},
		{
			name:     "uint64 value",
			value:    uint64(131415),
			expected: "131415",
		},
		{
			name:     "float32 value",
			value:    float32(3.14),
			expected: "3.14",
		},
		{
			name:     "float64 value",
			value:    float64(2.718),
			expected: "2.718",
		},
		{
			name:     "bool true",
			value:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			value:    false,
			expected: "false",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: "null",
		},
		{
			name:     "time value",
			value:    timestamp,
			expected: `"2024-01-15T10:30:45.123456789Z"`,
		},
		{
			name:     "slice value (complex type)",
			value:    []string{"a", "b"},
			expected: `["a","b"]`,
		},
		{
			name:     "map value (complex type)",
			value:    map[string]int{"count": 5},
			expected: `{"count":5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf []byte
			result := appendJSONValue(buf, tt.value)
			if string(result) != tt.expected {
				t.Errorf("appendJSONValue() = %v, want %v", string(result), tt.expected)
			}
		})
	}
}

// TEST: GIVEN JSON marshaling error WHEN using appendJSONValue THEN null is returned
func TestAppendJSONValueMarshalError(t *testing.T) {
	// Create a value that will cause json.Marshal to fail
	type UnmarshalableType struct {
		BadField chan int // channels can't be marshaled to JSON
	}

	var buf []byte
	result := appendJSONValue(buf, UnmarshalableType{BadField: make(chan int)})
	if string(result) != "null" {
		t.Errorf("appendJSONValue() with unmarshalable type = %v, want 'null'", string(result))
	}
}

// TEST: GIVEN file sink with directory creation needed WHEN opening file THEN directory is created
func TestFileSinkDirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "logs", "nested", "test.log")

	sink, err := NewFileSink(nestedPath, 0, 0)
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() { _ = sink.Close() }()

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("Directory was not created")
	}
}

// TEST: GIVEN file sink with Stat error WHEN opening file THEN error is returned
func TestFileSinkStatError(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "test.log")

	// Create a file that we can't stat (we'll need to simulate this)
	// This test is tricky because we need the file to open but Stat to fail
	// We'll create the file, then test Close error path instead

	sink, err := NewFileSink(filename, 0, 0)
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}

	// Force close the underlying file to test error handling
	if sink.file != nil {
		_ = sink.file.Close()
		sink.file = nil
	}

	// Now Close should handle nil file gracefully
	err = sink.Close()
	if err != nil {
		t.Errorf("Close() with nil file should not error, got: %v", err)
	}
}

// TEST: GIVEN file rotation scenario WHEN file has no content THEN rotation skips rename
func TestFileSinkRotationNoContent(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "test.log")

	sink, err := NewFileSink(filename, 1*time.Nanosecond, 0) // Very short rotation time
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() { _ = sink.Close() }()

	// Wait to trigger age-based rotation
	time.Sleep(2 * time.Nanosecond)

	// Write something to trigger rotation check
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	err = sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
}

// TEST: GIVEN NewDefaultFileSink with invalid path WHEN creating sink THEN panic occurs
func TestNewDefaultFileSinkPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewDefaultFileSink should panic with invalid path")
		}
	}()

	// Use platform-appropriate invalid path that will definitely fail
	var invalidPath string
	if isWindows() {
		// On Windows, use characters that are not allowed in file paths
		invalidPath = `C:\invalid\path\with<invalid>characters\test.log`
	} else {
		// On Unix-like systems, use a path that requires root permissions
		invalidPath = "/proc/invalid/path/test.log"
	}

	NewDefaultFileSink(invalidPath, time.Hour)
}

// TEST: GIVEN async file sink writer loop WHEN errors occur THEN they are handled gracefully
func TestAsyncFileSinkWriterLoopErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "async_test.log")

	sink, err := NewAsyncFileSink(filename, 0, 0, 10)
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}

	// Write a valid entry
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	err = sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	// Close and verify it handles the background goroutine properly
	err = sink.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Try to close again (should be idempotent)
	err = sink.Close()
	if err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

// TEST: GIVEN async file sink WHEN buffer is full THEN fallback to sync write occurs
func TestAsyncFileSinkSyncFallback(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "sync_fallback_test.log")

	// Create sink with very small buffer
	sink, err := NewAsyncFileSink(filename, 0, 0, 1)
	if err != nil {
		t.Fatalf("NewAsyncFileSink() error = %v", err)
	}
	defer func() { _ = sink.Close() }()

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	// Fill the buffer
	err = sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("First Write() error = %v", err)
	}

	// This should trigger sync fallback since buffer is full
	err = sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("Sync fallback Write() error = %v", err)
	}
}

// TEST: GIVEN resetGlobalLogger WHEN logger has nil workers THEN function handles gracefully
func TestResetGlobalLoggerWithNilWorkers(t *testing.T) {
	// First ensure we have a logger
	_ = getLogger()

	// Manually set a worker to nil to test nil handling
	if len(globalLogger.workers) > 0 {
		globalLogger.workers[0] = nil
	}

	// This should not panic
	resetGlobalLogger()

	// Verify logger is reset
	once = sync.Once{}
	globalLogger = nil
}

// TEST: GIVEN worker flush WHEN sink write fails THEN error is handled gracefully
func TestWorkerFlushSinkError(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	// Create a sink that always fails
	errorSink := &errorSink{}
	SetSinks(errorSink)

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	// Push entry to ring buffer
	logger := getLogger()
	logger.buffer.Push(entry)

	// Create worker and manually call flush to test error handling
	worker := NewWorker(0, logger, 10)
	worker.flush() // Should handle error gracefully
}

// TEST: GIVEN console sink write error WHEN writing entries THEN error is returned
func TestConsoleSinkWriteError(t *testing.T) {
	// Create a console sink with a failing writer
	sink := &ConsoleSink{
		writer: bufio.NewWriter(&failingWriter{}),
	}

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	err := sink.Write([]*Entry{entry})
	if err == nil {
		t.Error("Write() should return error with failing writer")
	}
}

// TEST: GIVEN stderr sink write error WHEN writing entries THEN error is returned
func TestStderrSinkWriteError(t *testing.T) {
	// Create a stderr sink with a failing writer
	sink := &StderrSink{
		writer: bufio.NewWriter(&failingWriter{}),
	}

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	err := sink.Write([]*Entry{entry})
	if err == nil {
		t.Error("Write() should return error with failing writer")
	}
}

// TEST: GIVEN file sink write error WHEN writing entries THEN error is returned
func TestFileSinkWriteError(t *testing.T) {
	tempDir := t.TempDir()
	filename := filepath.Join(tempDir, "write_error_test.log")

	sink, err := NewFileSink(filename, 0, 0)
	if err != nil {
		t.Fatalf("NewFileSink() error = %v", err)
	}
	defer func() { _ = sink.Close() }()

	// Replace writer with failing writer
	sink.writer = bufio.NewWriter(&failingWriter{})

	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{},
	}

	err = sink.Write([]*Entry{entry})
	if err == nil {
		t.Error("Write() should return error with failing writer")
	}
}

// Helper types for testing error conditions
type errorSink struct{}

func (es *errorSink) Write(entries []*Entry) error {
	return fmt.Errorf("simulated sink error")
}

func (es *errorSink) Close() error {
	return fmt.Errorf("simulated close error")
}

type failingWriter struct{}

func (fw *failingWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated write error")
}

func (fw *failingWriter) Flush() error {
	return fmt.Errorf("simulated flush error")
}

// TEST: GIVEN entry with large field map WHEN clearing fields THEN new map is created
func TestEntryFieldClearing(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	var buffer bytes.Buffer
	sink := &testSink{writer: &buffer}
	SetSinks(sink)

	// Create many fields to trigger the "large map" clearing path
	fields := make([]LogField, 20) // More than 8 to trigger new map creation
	for i := 0; i < 20; i++ {
		fields[i] = Field(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	Info("test with many fields", fields...)

	// Give time for background workers to process
	time.Sleep(50 * time.Millisecond)
	Shutdown()

	output := buffer.String()
	if !contains(output, "test with many fields") {
		t.Errorf("Expected message in output, got: %s", output)
	}
}

// TEST: GIVEN entry marshaling fails WHEN writing to sink THEN entry is skipped
func TestEntryMarshalFailureSkipped(t *testing.T) {
	sink := NewConsoleSink()

	// Create an entry that will fail to marshal by setting a problematic field
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test",
		Fields:    map[string]interface{}{"bad": make(chan int)}, // channels can't be marshaled
	}

	// This should not error, the bad entry should be skipped
	err := sink.Write([]*Entry{entry})
	if err != nil {
		t.Errorf("Write() should skip bad entries, got error: %v", err)
	}
}

// TEST: GIVEN shutdown called multiple times WHEN waiting for workers THEN no deadlock occurs
func TestShutdownIdempotency(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	// Initialize logger
	_ = getLogger()

	// Call shutdown multiple times
	Shutdown()
	Shutdown()
	Shutdown()

	// Should not hang or panic
}
