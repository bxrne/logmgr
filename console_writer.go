// Package logmgr provides high-performance structured logging for Go applications.
package logmgr

import (
	"bufio"
	"os"
	"sync"
)

// ConsoleSink writes log entries to stdout in JSON format with high performance buffering.
// It implements the Sink interface and is safe for concurrent use.
//
// Example:
//
//	sink := logmgr.NewConsoleSink()
//	logmgr.AddSink(sink)
type ConsoleSink struct {
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewConsoleSink creates a new console sink that writes to stdout.
// The sink uses an 8KB buffer for optimal performance.
//
// Example:
//
//	consoleSink := logmgr.NewConsoleSink()
//	logmgr.AddSink(consoleSink)
func NewConsoleSink() *ConsoleSink {
	return &ConsoleSink{
		writer: bufio.NewWriterSize(os.Stdout, 8192), // 8KB buffer
	}
}

// Write writes a batch of log entries to stdout in JSON format.
// Each entry is written as a single line of JSON followed by a newline.
// This method is safe for concurrent use.
//
// The method will skip any entries that fail to marshal to JSON rather than
// failing the entire batch.
//
// Example output:
//
//	{"level":"info","timestamp":"2024-01-15T10:30:45.123Z","message":"User logged in","user_id":12345}
//	{"level":"error","timestamp":"2024-01-15T10:30:46.456Z","message":"Database error","error":"connection timeout"}
func (cs *ConsoleSink) Write(entries []*Entry) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Pre-allocate buffer for the entire batch to reduce system calls
	var batchBuffer []byte
	if len(entries) > 0 {
		// Estimate buffer size: ~200 bytes per entry average
		batchBuffer = make([]byte, 0, len(entries)*200)
	}

	for _, entry := range entries {
		// Marshal to JSON
		data, err := entry.MarshalJSON()
		if err != nil {
			continue // Skip malformed entries
		}

		// Append to batch buffer with newline
		batchBuffer = append(batchBuffer, data...)
		batchBuffer = append(batchBuffer, '\n')
	}

	// Write the entire batch in one system call
	if len(batchBuffer) > 0 {
		if _, err := cs.writer.Write(batchBuffer); err != nil {
			return err
		}
	}

	// Flush the buffer
	return cs.writer.Flush()
}

// Close flushes any remaining buffered data and closes the console sink.
// This method should be called during application shutdown to ensure all
// log entries are written.
//
// Example:
//
//	defer consoleSink.Close()
func (cs *ConsoleSink) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.writer.Flush()
}

// DefaultConsoleSink is a pre-configured console sink instance that can be used immediately.
// This is the most common way to add console logging to your application.
//
// Example:
//
//	logmgr.AddSink(logmgr.DefaultConsoleSink)
var DefaultConsoleSink = NewConsoleSink()

// StderrSink writes log entries to stderr in JSON format.
// This is useful for separating error logs from regular output or when stdout
// is used for application data.
//
// Example:
//
//	stderrSink := logmgr.NewStderrSink()
//	logmgr.AddSink(stderrSink)
type StderrSink struct {
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewStderrSink creates a new stderr sink that writes to stderr.
// The sink uses an 8KB buffer for optimal performance.
//
// Example:
//
//	stderrSink := logmgr.NewStderrSink()
//	logmgr.AddSink(stderrSink)
func NewStderrSink() *StderrSink {
	return &StderrSink{
		writer: bufio.NewWriterSize(os.Stderr, 8192),
	}
}

// Write writes a batch of log entries to stderr in JSON format.
// Each entry is written as a single line of JSON followed by a newline.
// This method is safe for concurrent use.
//
// The method will skip any entries that fail to marshal to JSON rather than
// failing the entire batch.
func (ss *StderrSink) Write(entries []*Entry) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Pre-allocate buffer for the entire batch to reduce system calls
	var batchBuffer []byte
	if len(entries) > 0 {
		// Estimate buffer size: ~200 bytes per entry average
		batchBuffer = make([]byte, 0, len(entries)*200)
	}

	for _, entry := range entries {
		data, err := entry.MarshalJSON()
		if err != nil {
			continue
		}

		// Append to batch buffer with newline
		batchBuffer = append(batchBuffer, data...)
		batchBuffer = append(batchBuffer, '\n')
	}

	// Write the entire batch in one system call
	if len(batchBuffer) > 0 {
		if _, err := ss.writer.Write(batchBuffer); err != nil {
			return err
		}
	}

	return ss.writer.Flush()
}

// Close flushes any remaining buffered data and closes the stderr sink.
// This method should be called during application shutdown to ensure all
// log entries are written.
func (ss *StderrSink) Close() error {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.writer.Flush()
}
