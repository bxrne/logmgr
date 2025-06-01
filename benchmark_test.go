package logmgr

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// NullSink discards all log entries (for benchmarking)
type NullSink struct{}

func (ns *NullSink) Write(entries []*Entry) error {
	return nil
}

func (ns *NullSink) Close() error {
	return nil
}

// TEST: GIVEN a logger with null sink WHEN logging info messages repeatedly THEN benchmark measures throughput
func BenchmarkLogInfo(b *testing.B) {
	// Create a standalone logger for benchmarking
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

	// Initialize object pools properly
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				Fields: make(map[string]interface{}),
			}
		},
	}

	logger.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	// Add null sink to avoid I/O overhead
	logger.sinks = []Sink{&NullSink{}}

	// Start one worker for processing
	worker := NewWorker(0, logger, 256)
	logger.workers = []*Worker{worker}
	logger.wg.Add(1)
	go worker.Run()

	// Create a local log function that uses our logger
	logFunc := func(level Level, message string, fields ...LogField) {
		// Fast level check
		if level < Level(atomic.LoadInt32(&logger.level)) {
			return
		}

		// Get entry from pool
		entry := logger.entryPool.Get().(*Entry)
		entry.Level = level
		entry.Timestamp = time.Now()
		entry.Message = message

		// Clear and populate fields
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		for _, field := range fields {
			entry.Fields[field.Key] = field.Value
		}

		// Try to push to buffer (non-blocking)
		if !logger.buffer.Push(entry) {
			// Buffer full, return entry to pool
			logger.entryPool.Put(entry)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logFunc(InfoLevel, "benchmark message")
	}

	// Cleanup
	worker.Stop()
	logger.wg.Wait()
}

// TEST: GIVEN a logger with null sink WHEN logging info messages with fields repeatedly THEN benchmark measures structured logging throughput
func BenchmarkLogInfoWithFields(b *testing.B) {
	// Create a standalone logger for benchmarking
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

	// Initialize object pools properly
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				Fields: make(map[string]interface{}),
			}
		},
	}

	logger.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	logger.sinks = []Sink{&NullSink{}}

	worker := NewWorker(0, logger, 256)
	logger.workers = []*Worker{worker}
	logger.wg.Add(1)
	go worker.Run()

	// Create a local log function that uses our logger
	logFunc := func(level Level, message string, fields ...LogField) {
		// Fast level check
		if level < Level(atomic.LoadInt32(&logger.level)) {
			return
		}

		// Get entry from pool
		entry := logger.entryPool.Get().(*Entry)
		entry.Level = level
		entry.Timestamp = time.Now()
		entry.Message = message

		// Clear and populate fields
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		for _, field := range fields {
			entry.Fields[field.Key] = field.Value
		}

		// Try to push to buffer (non-blocking)
		if !logger.buffer.Push(entry) {
			// Buffer full, return entry to pool
			logger.entryPool.Put(entry)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logFunc(InfoLevel, "user action",
			Field("user_id", 12345),
			Field("action", "login"),
			Field("ip", "192.168.1.1"),
			Field("timestamp", time.Now().Unix()),
		)
	}

	// Cleanup
	worker.Stop()
	logger.wg.Wait()
}

// TEST: GIVEN a logger with high level filtering WHEN logging low-level messages repeatedly THEN benchmark measures filtering performance
func BenchmarkLogLevelFiltering(b *testing.B) {
	// Create a standalone logger for benchmarking
	logger := &Logger{
		level:    int32(ErrorLevel), // Only error and above
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

	// Initialize object pools properly
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				Fields: make(map[string]interface{}),
			}
		},
	}

	logger.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	logger.sinks = []Sink{&NullSink{}}

	// Create a local log function that uses our logger
	logFunc := func(level Level, message string, fields ...LogField) {
		// Fast level check
		if level < Level(atomic.LoadInt32(&logger.level)) {
			return
		}

		// Get entry from pool
		entry := logger.entryPool.Get().(*Entry)
		entry.Level = level
		entry.Timestamp = time.Now()
		entry.Message = message

		// Clear and populate fields
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		for _, field := range fields {
			entry.Fields[field.Key] = field.Value
		}

		// Try to push to buffer (non-blocking)
		if !logger.buffer.Push(entry) {
			// Buffer full, return entry to pool
			logger.entryPool.Put(entry)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// These should be filtered out quickly
		logFunc(DebugLevel, "debug message")
		logFunc(InfoLevel, "info message")
		logFunc(WarnLevel, "warn message")
	}
}

// TEST: GIVEN log entries with various fields WHEN marshaling to JSON repeatedly THEN benchmark measures JSON serialization performance
func BenchmarkJSONMarshal(b *testing.B) {
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "benchmark message",
		Fields: map[string]interface{}{
			"user_id": 12345,
			"action":  "login",
			"ip":      "192.168.1.1",
			"latency": 45.67,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := entry.MarshalJSON()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TEST: GIVEN a RingBuffer WHEN pushing entries in parallel THEN benchmark measures lock-free buffer performance
func BenchmarkRingBuffer(b *testing.B) {
	rb := NewRingBuffer(8192)
	entry := &Entry{
		Level:     InfoLevel,
		Timestamp: time.Now(),
		Message:   "test message",
		Fields:    make(map[string]interface{}),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rb.Push(entry)
		}
	})
}

// TEST: GIVEN console sink with redirected output WHEN writing batches of entries THEN benchmark measures console output performance
func BenchmarkConsoleSink(b *testing.B) {
	// Redirect stdout to null to avoid terminal overhead
	oldStdout := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0644)
	if err != nil {
		b.Fatal(err)
	}
	os.Stdout = devNull
	defer func() {
		os.Stdout = oldStdout
		if err := devNull.Close(); err != nil {
			b.Logf("Failed to close devNull: %v", err)
		}
	}()

	sink := NewConsoleSink()
	entries := make([]*Entry, 10)

	for i := range entries {
		entries[i] = &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "benchmark message",
			Fields: map[string]interface{}{
				"batch_id": i,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sink.Write(entries); err != nil {
			b.Fatal(err)
		}
	}
}

// TEST: GIVEN file sink with temp file WHEN writing batches of entries THEN benchmark measures file output performance
func BenchmarkFileSink(b *testing.B) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "logmgr_bench_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			b.Logf("Failed to remove temp file: %v", err)
		}
	}()
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	sink, err := NewFileSink(tmpFile.Name(), 0, 0) // No rotation for benchmark
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			b.Logf("Failed to close sink: %v", err)
		}
	}()

	entries := make([]*Entry, 10)
	for i := range entries {
		entries[i] = &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "benchmark message",
			Fields: map[string]interface{}{
				"batch_id": i,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sink.Write(entries); err != nil {
			b.Fatal(err)
		}
	}
}

// TEST: GIVEN async file sink with buffer WHEN writing batches of entries THEN benchmark measures async file output performance
func BenchmarkAsyncFileSink(b *testing.B) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "logmgr_async_bench_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			b.Logf("Failed to remove temp file: %v", err)
		}
	}()
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	sink, err := NewAsyncFileSink(tmpFile.Name(), 0, 0, 1000)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := sink.Close(); err != nil {
			b.Logf("Failed to close sink: %v", err)
		}
	}()

	entries := make([]*Entry, 10)
	for i := range entries {
		entries[i] = &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "benchmark message",
			Fields: map[string]interface{}{
				"batch_id": i,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sink.Write(entries); err != nil {
			b.Fatal(err)
		}
	}
}

// TEST: GIVEN logger with multiple workers WHEN logging concurrently from multiple goroutines THEN benchmark measures concurrent logging performance
func BenchmarkConcurrentLogging(b *testing.B) {
	// Create a standalone logger for benchmarking
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

	// Initialize object pools properly
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				Fields: make(map[string]interface{}),
			}
		},
	}

	logger.bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 0, 1024)
		},
	}

	logger.sinks = []Sink{&NullSink{}}

	// Start multiple workers
	numWorkers := 4
	logger.workers = make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(i, logger, 256)
		logger.workers[i] = worker
		logger.wg.Add(1)
		go worker.Run()
	}

	// Create a local log function that uses our logger
	logFunc := func(level Level, message string, fields ...LogField) {
		// Fast level check
		if level < Level(atomic.LoadInt32(&logger.level)) {
			return
		}

		// Get entry from pool
		entry := logger.entryPool.Get().(*Entry)
		entry.Level = level
		entry.Timestamp = time.Now()
		entry.Message = message

		// Clear and populate fields
		for k := range entry.Fields {
			delete(entry.Fields, k)
		}
		for _, field := range fields {
			entry.Fields[field.Key] = field.Value
		}

		// Try to push to buffer (non-blocking)
		if !logger.buffer.Push(entry) {
			// Buffer full, return entry to pool
			logger.entryPool.Put(entry)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logFunc(InfoLevel, "concurrent message",
			Field("goroutine_id", "test"),
			Field("counter", 1),
		)
	}

	// Cleanup
	for _, worker := range logger.workers {
		worker.Stop()
	}
	logger.wg.Wait()
}
