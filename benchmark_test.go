package logmgr

import (
	stdlog "log"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NullSink discards all log entries (for benchmarking)
type NullSink struct{}

func (ns *NullSink) Write(entries []*Entry) error {
	return nil
}

func (ns *NullSink) Close() error {
	return nil
}

// NullWriter discards all writes (for benchmarking other loggers)
type NullWriter struct{}

func (nw *NullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// Helper function for benchmark logging to avoid code duplication
func createBenchmarkLogFunc(logger *Logger) func(Level, string, ...LogField) {
	return func(level Level, message string, fields ...LogField) {
		// Fast level check
		if level < Level(atomic.LoadInt32(&logger.level)) {
			return
		}

		// Get entry from pool
		entry := logger.entryPool.Get().(*Entry)
		entry.Level = level
		entry.Timestamp = time.Now()
		entry.Message = message

		// Ensure fields map is initialized
		if entry.Fields == nil {
			entry.Fields = make(map[string]interface{})
		}

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
}

// BenchmarkLogmgr tests logmgr simple logging performance
func BenchmarkLogmgr_Simple(b *testing.B) {
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
	logFunc := createBenchmarkLogFunc(logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logFunc(InfoLevel, "benchmark message")
		}
	})

	// Cleanup
	worker.Stop()
	logger.wg.Wait()
}

// BenchmarkLogmgr_Structured tests logmgr structured logging performance
func BenchmarkLogmgr_Structured(b *testing.B) {
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
	logFunc := createBenchmarkLogFunc(logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logFunc(InfoLevel, "user action",
				Field("user_id", 12345),
				Field("action", "login"),
				Field("ip", "192.168.1.1"),
				Field("timestamp", time.Now().Unix()),
			)
		}
	})

	// Cleanup
	worker.Stop()
	logger.wg.Wait()
}

// BenchmarkZap_Simple tests Zap simple logging performance
func BenchmarkZap_Simple(b *testing.B) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{}
	config.ErrorOutputPaths = []string{}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&NullWriter{}),
		zapcore.InfoLevel,
	)

	logger := zap.New(core)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message")
		}
	})
}

// BenchmarkZap_Structured tests Zap structured logging performance
func BenchmarkZap_Structured(b *testing.B) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{}
	config.ErrorOutputPaths = []string{}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&NullWriter{}),
		zapcore.InfoLevel,
	)

	logger := zap.New(core)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("user action",
				zap.Int("user_id", 12345),
				zap.String("action", "login"),
				zap.String("ip", "192.168.1.1"),
				zap.Int64("timestamp", time.Now().Unix()),
			)
		}
	})
}

// BenchmarkLogrus_Simple tests Logrus simple logging performance
func BenchmarkLogrus_Simple(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(&NullWriter{})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message")
		}
	})
}

// BenchmarkLogrus_Structured tests Logrus structured logging performance
func BenchmarkLogrus_Structured(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(&NullWriter{})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.WithFields(logrus.Fields{
				"user_id":   12345,
				"action":    "login",
				"ip":        "192.168.1.1",
				"timestamp": time.Now().Unix(),
			}).Info("user action")
		}
	})
}

// BenchmarkStdLog_Simple tests standard library log simple logging performance
func BenchmarkStdLog_Simple(b *testing.B) {
	logger := stdlog.New(&NullWriter{}, "", stdlog.LstdFlags)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Println("benchmark message")
		}
	})
}

// BenchmarkSlog_Simple tests slog simple logging performance
func BenchmarkSlog_Simple(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&NullWriter{}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("benchmark message")
		}
	})
}

// BenchmarkSlog_Structured tests slog structured logging performance
func BenchmarkSlog_Structured(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&NullWriter{}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("user action",
				slog.Int("user_id", 12345),
				slog.String("action", "login"),
				slog.String("ip", "192.168.1.1"),
				slog.Int64("timestamp", time.Now().Unix()),
			)
		}
	})
}

// BenchmarkLogmgr_LevelFiltering tests logmgr level filtering performance
func BenchmarkLogmgr_LevelFiltering(b *testing.B) {
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
	logFunc := createBenchmarkLogFunc(logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logFunc(InfoLevel, "this should be filtered") // Below error level
		}
	})
}

// BenchmarkZap_LevelFiltering tests Zap level filtering performance
func BenchmarkZap_LevelFiltering(b *testing.B) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	config.OutputPaths = []string{}
	config.ErrorOutputPaths = []string{}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&NullWriter{}),
		zapcore.ErrorLevel,
	)

	logger := zap.New(core)
	defer logger.Sync()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("this should be filtered") // Below error level
		}
	})
}

// BenchmarkLogrus_LevelFiltering tests Logrus level filtering performance
func BenchmarkLogrus_LevelFiltering(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(&NullWriter{})
	logger.SetLevel(logrus.ErrorLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("this should be filtered") // Below error level
		}
	})
}

// BenchmarkSlog_LevelFiltering tests slog level filtering performance
func BenchmarkSlog_LevelFiltering(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&NullWriter{}, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("this should be filtered") // Below error level
		}
	})
}

// Memory allocation benchmarks
func BenchmarkLogmgr_Allocations(b *testing.B) {
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

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

	logFunc := createBenchmarkLogFunc(logger)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logFunc(InfoLevel, "benchmark message",
			Field("user_id", 12345),
			Field("action", "login"),
		)
	}

	worker.Stop()
	logger.wg.Wait()
}

func BenchmarkZap_Allocations(b *testing.B) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{}
	config.ErrorOutputPaths = []string{}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(config.EncoderConfig),
		zapcore.AddSync(&NullWriter{}),
		zapcore.InfoLevel,
	)

	logger := zap.New(core)
	defer logger.Sync()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			zap.Int("user_id", 12345),
			zap.String("action", "login"),
		)
	}
}

func BenchmarkLogrus_Allocations(b *testing.B) {
	logger := logrus.New()
	logger.SetOutput(&NullWriter{})
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(logrus.Fields{
			"user_id": 12345,
			"action":  "login",
		}).Info("benchmark message")
	}
}

func BenchmarkSlog_Allocations(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(&NullWriter{}, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			slog.Int("user_id", 12345),
			slog.String("action", "login"),
		)
	}
}

// Test concurrent logging performance under contention
func BenchmarkConcurrentLogging(b *testing.B) {
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
	}

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

	// Start multiple workers to handle the load
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		worker := NewWorker(i, logger, 256)
		logger.workers = append(logger.workers, worker)
		logger.wg.Add(1)
		go worker.Run()
	}

	logFunc := createBenchmarkLogFunc(logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logFunc(InfoLevel, "concurrent message",
				Field("goroutine", "worker"),
				Field("timestamp", time.Now().Unix()),
			)
		}
	})

	// Cleanup
	for _, worker := range logger.workers {
		worker.Stop()
	}
	logger.wg.Wait()
}
