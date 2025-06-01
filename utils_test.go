package logmgr

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
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

func TestRingBufferPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewRingBuffer should panic with non-power-of-2 size")
		}
	}()
	NewRingBuffer(3) // Not a power of 2
}

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

// Test logging functions
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

func TestGetLevel(t *testing.T) {
	// Reset global state
	resetGlobalLogger()

	SetLevel(WarnLevel)
	if GetLevel() != WarnLevel {
		t.Errorf("GetLevel() = %v, want %v", GetLevel(), WarnLevel)
	}

	Shutdown()
}

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

// Test console sink
func TestConsoleSink(t *testing.T) {
	testStreamSink(t, "console", NewConsoleSink(), &os.Stdout, "test message", "Console output should contain test message")
}

func TestStderrSink(t *testing.T) {
	testStreamSink(t, "stderr", NewStderrSink(), &os.Stderr, "error message", "Stderr output should contain error message")
}

// Helper function to test stream sinks (console and stderr)
func testStreamSink(t *testing.T, name string, sink Sink, stream **os.File, message, expectation string) {
	// Redirect stream to capture output
	oldStream := *stream
	r, w, _ := os.Pipe()
	*stream = w

	// Use appropriate level based on sink type
	level := InfoLevel
	if name == "stderr" {
		level = ErrorLevel
	}

	entries := []*Entry{
		{
			Level:     level,
			Timestamp: time.Now(),
			Message:   message,
			Fields:    map[string]interface{}{"key": "value"},
		},
	}

	err := sink.Write(entries)
	if err != nil {
		t.Errorf("%sSink.Write() error = %v", name, err)
	}

	err = sink.Close()
	if err != nil {
		t.Errorf("%sSink.Close() error = %v", name, err)
	}

	// Restore stream
	if err := w.Close(); err != nil {
		t.Logf("Failed to close pipe writer: %v", err)
	}
	*stream = oldStream

	// Read captured output
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Logf("Failed to copy from pipe reader: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Logf("Failed to close pipe reader: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, message) {
		t.Error(expectation)
	}
}

// Test file sink
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

func TestLoggerWithNoSinks(t *testing.T) {
	resetGlobalLogger()

	// Test logging without any sinks (should not crash)
	SetLevel(InfoLevel)
	Info("message with no sinks")

	time.Sleep(50 * time.Millisecond)
	Shutdown()
}

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

func TestFileSinkErrors(t *testing.T) {
	// Test creating file sink with invalid directory
	_, err := NewFileSink("/invalid/path/that/does/not/exist/test.log", 0, 0)
	if err == nil {
		t.Error("NewFileSink should fail with invalid path")
	}
}

func TestAsyncFileSinkErrors(t *testing.T) {
	// Test creating async file sink with invalid directory
	_, err := NewAsyncFileSink("/invalid/path/that/does/not/exist/async.log", 0, 0, 10)
	if err == nil {
		t.Error("NewAsyncFileSink should fail with invalid path")
	}
}

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

func TestStderrSinkBufferFlush(t *testing.T) {
	// Test that stderr sink flushes buffer properly
	sink := NewStderrSink()

	// Create many entries to test buffer flushing
	entries := make([]*Entry, 100)
	for i := range entries {
		entries[i] = &Entry{
			Level:     ErrorLevel,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("error %d", i),
			Fields:    map[string]interface{}{"index": i},
		}
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
