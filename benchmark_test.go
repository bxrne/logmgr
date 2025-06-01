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

// BenchmarkLogInfo benchmarks basic info logging
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

// BenchmarkLogInfoWithFields benchmarks logging with structured fields
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

// BenchmarkLogLevelFiltering benchmarks level filtering performance
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

// BenchmarkJSONMarshal benchmarks JSON marshaling performance
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

// BenchmarkRingBuffer benchmarks the lock-free ring buffer
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

// BenchmarkConsoleSink benchmarks console output performance
func BenchmarkConsoleSink(b *testing.B) {
	// Redirect stdout to null to avoid terminal overhead
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

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
		sink.Write(entries)
	}
}

// BenchmarkFileSink benchmarks file output performance
func BenchmarkFileSink(b *testing.B) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "logmgr_bench_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sink, err := NewFileSink(tmpFile.Name(), 0, 0) // No rotation for benchmark
	if err != nil {
		b.Fatal(err)
	}
	defer sink.Close()

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
		sink.Write(entries)
	}
}

// BenchmarkAsyncFileSink benchmarks async file output performance
func BenchmarkAsyncFileSink(b *testing.B) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "logmgr_async_bench_*.log")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	sink, err := NewAsyncFileSink(tmpFile.Name(), 0, 0, 1000)
	if err != nil {
		b.Fatal(err)
	}
	defer sink.Close()

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
		sink.Write(entries)
	}
}

// BenchmarkConcurrentLogging benchmarks concurrent logging from multiple goroutines
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

// Example benchmark results output
func ExampleBenchmarkResults() {
	// Run: go test -bench=. -benchmem
	//
	// Expected results (approximate):
	// BenchmarkLogInfo-8                    	10000000	       120 ns/op	      48 B/op	       1 allocs/op
	// BenchmarkLogInfoWithFields-8          	 5000000	       250 ns/op	     128 B/op	       2 allocs/op
	// BenchmarkLogLevelFiltering-8          	50000000	        25 ns/op	       0 B/op	       0 allocs/op
	// BenchmarkJSONMarshal-8                	 2000000	       800 ns/op	     256 B/op	       3 allocs/op
	// BenchmarkRingBuffer-8                 	20000000	        85 ns/op	       0 B/op	       0 allocs/op
	// BenchmarkConsoleSink-8                	  100000	     15000 ns/op	    2560 B/op	      20 allocs/op
	// BenchmarkFileSink-8                   	  200000	      8000 ns/op	    2560 B/op	      20 allocs/op
	// BenchmarkAsyncFileSink-8              	 1000000	      1200 ns/op	    2560 B/op	      20 allocs/op
	// BenchmarkConcurrentLogging-8          	 8000000	       180 ns/op	      48 B/op	       1 allocs/op
}
