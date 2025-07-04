package logmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	// Global logger instance
	globalLogger *Logger
	once         sync.Once
)

// Logger represents the main logger instance with high-performance features
type Logger struct {
	level      int32          // Atomic level for thread-safe access
	sinks      []Sink         // Output sinks
	sinksMu    sync.RWMutex   // Protects sinks slice
	buffer     *RingBuffer    // Lock-free ring buffer
	workers    []*Worker      // Background workers
	entryPool  sync.Pool      // Object pool for entries
	bufferPool sync.Pool      // Object pool for byte buffers
	shutdown   chan struct{}  // Shutdown signal
	wg         sync.WaitGroup // Wait group for workers
}

// Entry represents a log entry with structured fields
type Entry struct {
	Level     Level                  `json:"level"`
	Timestamp time.Time              `json:"timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"-"` // Don't marshal this directly
	caller    string                 // Internal field for caller info
	buffer    []byte                 // Reusable buffer for JSON marshalling
}

// LogField represents a structured logging field
type LogField struct {
	Key   string
	Value interface{}
}

// Field creates a new structured logging field
//
// Example:
//
//	logmgr.Info("User action",
//	  logmgr.Field("user_id", 12345),
//	  logmgr.Field("action", "login"),
//	)
func Field(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// Sink interface for output destinations
type Sink interface {
	// Write processes a batch of log entries
	Write(entries []*Entry) error
	// Close gracefully shuts down the sink
	Close() error
}

// Level represents log severity levels
type Level int32

const (
	// DebugLevel is used for detailed debugging information
	DebugLevel Level = iota
	// InfoLevel is used for general informational messages
	InfoLevel
	// WarnLevel is used for potentially harmful situations
	WarnLevel
	// ErrorLevel is used for error events that might still allow the application to continue
	ErrorLevel
	// FatalLevel is used for very severe error events that will lead the application to abort
	FatalLevel
)

// String returns the string representation of the level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// RingBuffer implements a lock-free ring buffer for high-performance logging
type RingBuffer struct {
	buffer   []unsafe.Pointer // Ring buffer of entry pointers
	mask     uint64           // Size mask (size must be power of 2)
	writePos uint64           // Write position (atomic)
	readPos  uint64           // Read position (atomic)
}

// NewRingBuffer creates a new ring buffer with the given size (must be power of 2)
func NewRingBuffer(size uint64) *RingBuffer {
	if size&(size-1) != 0 {
		panic("ring buffer size must be power of 2")
	}
	return &RingBuffer{
		buffer: make([]unsafe.Pointer, size),
		mask:   size - 1,
	}
}

// Push adds an entry to the ring buffer (lock-free)
func (rb *RingBuffer) Push(entry *Entry) bool {
	// Safety checks
	if rb == nil || entry == nil || rb.buffer == nil {
		return false
	}

	writePos := atomic.LoadUint64(&rb.writePos)
	readPos := atomic.LoadUint64(&rb.readPos)

	// Check if buffer is full
	if writePos-readPos >= uint64(len(rb.buffer)) {
		return false // Buffer full, drop entry
	}

	// Store entry
	atomic.StorePointer(&rb.buffer[writePos&rb.mask], unsafe.Pointer(entry))
	atomic.AddUint64(&rb.writePos, 1)
	return true
}

// Pop removes and returns entries from the ring buffer
func (rb *RingBuffer) Pop(entries []*Entry) int {
	// Safety checks
	if rb == nil || entries == nil || len(entries) == 0 || rb.buffer == nil {
		return 0
	}

	readPos := atomic.LoadUint64(&rb.readPos)
	writePos := atomic.LoadUint64(&rb.writePos)

	// Handle race condition where readPos might exceed writePos
	if writePos < readPos {
		return 0
	}

	available := writePos - readPos
	if available == 0 {
		return 0
	}

	// Safely convert uint64 to int with bounds checking
	count := int(available)
	maxInt := int(^uint(0) >> 1)    // Calculate max int value
	if available > uint64(maxInt) { // Check for int overflow
		count = maxInt // Set to max int value
	}
	if count > len(entries) {
		count = len(entries)
	}
	if count < 0 { // Additional safety check
		return 0
	}

	for i := 0; i < count; i++ {
		ptr := atomic.LoadPointer(&rb.buffer[(readPos+uint64(i))&rb.mask])
		if ptr != nil {
			entries[i] = (*Entry)(ptr)
		} else {
			// If we encounter a nil pointer, stop processing to avoid corruption
			count = i
			break
		}
	}

	if count > 0 {
		atomic.AddUint64(&rb.readPos, uint64(count))
	}
	return count
}

// Worker processes log entries in background
type Worker struct {
	id       int
	logger   *Logger
	batch    []*Entry
	shutdown chan struct{}
}

// NewWorker creates a new background worker
func NewWorker(id int, logger *Logger, batchSize int) *Worker {
	// Ensure minimum batch size to prevent slice issues
	if batchSize < 1 {
		batchSize = 256
	}
	return &Worker{
		id:       id,
		logger:   logger,
		batch:    make([]*Entry, batchSize),
		shutdown: make(chan struct{}),
	}
}

// Run starts the worker loop
func (w *Worker) Run() {
	defer w.logger.wg.Done()

	ticker := time.NewTicker(10 * time.Millisecond) // Flush every 10ms
	defer ticker.Stop()

	for {
		select {
		case <-w.shutdown:
			w.flush() // Final flush
			return
		case <-ticker.C:
			w.flush()
		}
	}
}

// flush processes available entries
func (w *Worker) flush() {
	count := w.logger.buffer.Pop(w.batch)
	if count <= 0 {
		return
	}

	// Additional safety check for slice bounds
	if count > len(w.batch) {
		count = len(w.batch)
	}

	// Take a snapshot of sinks to reduce lock contention
	w.logger.sinksMu.RLock()
	sinks := make([]Sink, len(w.logger.sinks))
	copy(sinks, w.logger.sinks)
	w.logger.sinksMu.RUnlock()

	// Process entries with each sink
	for _, sink := range sinks {
		if sink != nil && count > 0 && count <= len(w.batch) {
			if err := sink.Write(w.batch[:count]); err != nil {
				// In a production system, you might want to handle this error
				// For now, we continue to other sinks
				_ = err
			}
		}
	}

	// Return entries to pool with additional safety checks
	for i := 0; i < count && i < len(w.batch); i++ {
		if w.batch[i] != nil {
			w.logger.entryPool.Put(w.batch[i])
			w.batch[i] = nil // Clear reference
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	select {
	case <-w.shutdown:
		// Already stopped
	default:
		close(w.shutdown)
	}
}

// getLogger returns the global logger instance (singleton)
func getLogger() *Logger {
	once.Do(func() {
		globalLogger = &Logger{
			level:    int32(InfoLevel),
			buffer:   NewRingBuffer(8192),
			shutdown: make(chan struct{}),
		}

		// Initialize object pools with optimized allocations
		globalLogger.entryPool = sync.Pool{
			New: func() interface{} {
				return &Entry{
					Fields: make(map[string]interface{}, 8), // Pre-size for common case
					buffer: make([]byte, 0, 512),            // Pre-allocate 512 bytes
				}
			},
		}

		globalLogger.bufferPool = sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		}

		// Start workers (one per CPU core)
		numWorkers := runtime.NumCPU()
		globalLogger.workers = make([]*Worker, numWorkers)
		for i := 0; i < numWorkers; i++ {
			worker := NewWorker(i, globalLogger, 256)
			globalLogger.workers[i] = worker
			globalLogger.wg.Add(1)
			go worker.Run()
		}
	})
	return globalLogger
}

// SetLevel sets the global log level
//
// Example:
//
//	logmgr.SetLevel(logmgr.DebugLevel)
func SetLevel(level Level) {
	logger := getLogger()
	atomic.StoreInt32(&logger.level, int32(level))
}

// GetLevel returns the current log level
func GetLevel() Level {
	logger := getLogger()
	return Level(atomic.LoadInt32(&logger.level))
}

// AddSink adds a sink to the logger
//
// Example:
//
//	logmgr.AddSink(logmgr.DefaultConsoleSink)
//	fileSink, _ := logmgr.NewFileSink("app.log", 24*time.Hour, 100*1024*1024)
//	logmgr.AddSink(fileSink)
func AddSink(sink Sink) {
	logger := getLogger()
	logger.sinksMu.Lock()
	logger.sinks = append(logger.sinks, sink)
	logger.sinksMu.Unlock()
}

// SetSinks replaces all sinks with the provided ones
//
// Example:
//
//	logmgr.SetSinks(logmgr.DefaultConsoleSink, fileSink)
func SetSinks(sinks ...Sink) {
	logger := getLogger()
	logger.sinksMu.Lock()
	logger.sinks = make([]Sink, len(sinks))
	copy(logger.sinks, sinks)
	logger.sinksMu.Unlock()
}

// log is the internal logging function
func log(level Level, message string, fields ...LogField) {
	logger := getLogger()
	if logger == nil {
		return // Safety check
	}

	// Fast level check
	if level < Level(atomic.LoadInt32(&logger.level)) {
		return
	}

	// Safety check for buffer
	if logger.buffer == nil {
		return
	}

	// Get entry from pool with nil safety
	var entry *Entry
	if poolEntry := logger.entryPool.Get(); poolEntry != nil {
		if e, ok := poolEntry.(*Entry); ok && e != nil {
			entry = e
		}
	}

	// Always create new entry if we don't have a valid one
	if entry == nil {
		entry = &Entry{
			Fields: make(map[string]interface{}, 8),
			buffer: make([]byte, 0, 512),
		}
	}

	entry.Level = level
	entry.Timestamp = time.Now()
	entry.Message = message

	// Ensure fields map is initialized
	if entry.Fields == nil {
		entry.Fields = make(map[string]interface{}, 8)
	}

	// Always create a fresh map to avoid race conditions
	entry.Fields = make(map[string]interface{}, len(fields))

	// Populate fields
	for _, field := range fields {
		if field.Key != "" { // Only add fields with non-empty keys
			entry.Fields[field.Key] = field.Value
		}
	}

	// Try to push to buffer (non-blocking)
	if !logger.buffer.Push(entry) {
		// Buffer full, return entry to pool
		logger.entryPool.Put(entry)
	}
}

// Debug logs a message at debug level with optional structured fields
//
// Example:
//
//	logmgr.Debug("Processing request",
//	  logmgr.Field("request_id", "req-123"),
//	  logmgr.Field("user_id", 456),
//	)
func Debug(message string, fields ...LogField) {
	log(DebugLevel, message, fields...)
}

// Info logs a message at info level with optional structured fields
//
// Example:
//
//	logmgr.Info("User logged in",
//	  logmgr.Field("user_id", 12345),
//	  logmgr.Field("action", "login"),
//	)
func Info(message string, fields ...LogField) {
	log(InfoLevel, message, fields...)
}

// Warn logs a message at warn level with optional structured fields
//
// Example:
//
//	logmgr.Warn("High memory usage",
//	  logmgr.Field("memory_percent", 85.5),
//	  logmgr.Field("threshold", 80.0),
//	)
func Warn(message string, fields ...LogField) {
	log(WarnLevel, message, fields...)
}

// Error logs a message at error level with optional structured fields
//
// Example:
//
//	logmgr.Error("Database connection failed",
//	  logmgr.Field("error", "connection timeout"),
//	  logmgr.Field("host", "db.example.com"),
//	  logmgr.Field("retries", 3),
//	)
func Error(message string, fields ...LogField) {
	log(ErrorLevel, message, fields...)
}

// Fatal logs a message at fatal level with optional structured fields and exits the program
//
// Example:
//
//	logmgr.Fatal("Critical system failure",
//	  logmgr.Field("error", "out of memory"),
//	  logmgr.Field("available_memory", "0MB"),
//	)
func Fatal(message string, fields ...LogField) {
	log(FatalLevel, message, fields...)
	Shutdown() // Flush all logs
	os.Exit(1)
}

// Shutdown gracefully shuts down the logger, ensuring all logs are flushed
//
// Example:
//
//	defer logmgr.Shutdown()
func Shutdown() {
	logger := getLogger()

	// Stop all workers
	for _, worker := range logger.workers {
		worker.Stop()
	}

	// Wait for workers to finish
	logger.wg.Wait()

	// Close all sinks
	logger.sinksMu.RLock()
	for _, sink := range logger.sinks {
		if err := sink.Close(); err != nil {
			// In a real application, you might want to log this error
			// to a fallback location or handle it appropriately
			_ = err
		}
	}
	logger.sinksMu.RUnlock()
}

// MarshalJSON implements custom JSON marshaling for Entry with flattened fields
// Field ordering convention: level, timestamp, message, then custom fields in alphabetical order
func (e *Entry) MarshalJSON() ([]byte, error) {
	// Reset buffer, keeping capacity - with nil safety
	if e.buffer == nil {
		e.buffer = make([]byte, 0, 512)
	}
	e.buffer = e.buffer[:0]
	e.buffer = append(e.buffer, '{')

	// Level (always first for easy parsing)
	e.buffer = append(e.buffer, `"level":"`...)
	e.buffer = append(e.buffer, e.Level.String()...)
	e.buffer = append(e.buffer, '"')

	// Timestamp (always second for consistent ordering)
	e.buffer = append(e.buffer, `,"timestamp":"`...)
	e.buffer = append(e.buffer, e.Timestamp.Format(time.RFC3339Nano)...)
	e.buffer = append(e.buffer, '"')

	// Message (always third for consistent ordering)
	e.buffer = append(e.buffer, `,"message":`...)
	e.buffer = appendJSONString(e.buffer, e.Message)

	// Custom fields in alphabetical order for deterministic output
	if len(e.Fields) > 0 {
		// Get sorted keys
		keys := make([]string, 0, len(e.Fields))
		for key := range e.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		// Output fields in alphabetical order
		for _, key := range keys {
			value := e.Fields[key]
			e.buffer = append(e.buffer, ',')
			e.buffer = append(e.buffer, '"')
			e.buffer = append(e.buffer, key...)
			e.buffer = append(e.buffer, `":`...)
			e.buffer = appendJSONValue(e.buffer, value)
		}
	}

	e.buffer = append(e.buffer, '}')

	// Return a copy to avoid buffer reuse issues
	result := make([]byte, len(e.buffer))
	copy(result, e.buffer)
	return result, nil
}

// appendJSONString appends a JSON-encoded string to the buffer
func appendJSONString(buf []byte, s string) []byte {
	buf = append(buf, '"')
	for _, r := range s {
		switch r {
		case '"':
			buf = append(buf, `\"`...)
		case '\\':
			buf = append(buf, `\\`...)
		case '\n':
			buf = append(buf, `\n`...)
		case '\r':
			buf = append(buf, `\r`...)
		case '\t':
			buf = append(buf, `\t`...)
		default:
			if r < 32 {
				buf = append(buf, fmt.Sprintf(`\u%04x`, r)...)
			} else {
				buf = append(buf, string(r)...)
			}
		}
	}
	buf = append(buf, '"')
	return buf
}

// appendJSONValue appends a JSON-encoded value to the buffer
func appendJSONValue(buf []byte, value interface{}) []byte {
	switch v := value.(type) {
	case string:
		return appendJSONString(buf, v)
	case int:
		return append(buf, fmt.Sprintf("%d", v)...)
	case int32:
		return append(buf, fmt.Sprintf("%d", v)...)
	case int64:
		return append(buf, fmt.Sprintf("%d", v)...)
	case uint:
		return append(buf, fmt.Sprintf("%d", v)...)
	case uint32:
		return append(buf, fmt.Sprintf("%d", v)...)
	case uint64:
		return append(buf, fmt.Sprintf("%d", v)...)
	case float32:
		return append(buf, fmt.Sprintf("%g", v)...)
	case float64:
		return append(buf, fmt.Sprintf("%g", v)...)
	case bool:
		if v {
			return append(buf, "true"...)
		}
		return append(buf, "false"...)
	case nil:
		return append(buf, "null"...)
	case time.Time:
		// Fast path for time.Time
		buf = append(buf, '"')
		buf = append(buf, v.Format(time.RFC3339Nano)...)
		buf = append(buf, '"')
		return buf
	default:
		// Fallback to json.Marshal for complex types
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return append(buf, "null"...)
		}
		return append(buf, valueBytes...)
	}
}

// resetGlobalLogger resets the global logger state for testing
// This function should only be used in tests
func resetGlobalLogger() {
	if globalLogger != nil {
		// Stop existing workers if any
		for _, worker := range globalLogger.workers {
			if worker != nil {
				// Check if worker is still running before stopping
				select {
				case <-worker.shutdown:
					// Already stopped
				default:
					worker.Stop()
				}
			} else {
				// Worker is nil, decrement the wait group manually
				globalLogger.wg.Done()
			}
		}
		globalLogger.wg.Wait()

		// Close existing sinks
		globalLogger.sinksMu.RLock()
		for _, sink := range globalLogger.sinks {
			if err := sink.Close(); err != nil {
				// In tests, we might want to ignore close errors
				_ = err
			}
		}
		globalLogger.sinksMu.RUnlock()
	}

	once = sync.Once{}
	globalLogger = nil
}
