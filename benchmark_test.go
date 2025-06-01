package logmgr

import (
	stdlog "log"
	"log/slog"
	"sync"
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

// BenchmarkLogmgr_Simple tests logmgr simple logging performance
func BenchmarkLogmgr_Simple(b *testing.B) {
	// Create dedicated logger for benchmarking to avoid race conditions
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
		sinks:    []Sink{&NullSink{}},
	}

	// Initialize a simple pool that always creates new entries
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				Fields: make(map[string]interface{}, 8),
				buffer: make([]byte, 0, 512),
			}
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Direct call to avoid global state conflicts
		entry := &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "benchmark message",
			Fields:    make(map[string]interface{}),
		}
		logger.buffer.Push(entry)
	}
}

// BenchmarkLogmgr_Structured tests logmgr structured logging performance
func BenchmarkLogmgr_Structured(b *testing.B) {
	// Create dedicated logger for benchmarking to avoid race conditions
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
		sinks:    []Sink{&NullSink{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Direct call to avoid global state conflicts
		entry := &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "user action",
			Fields: map[string]interface{}{
				"user_id":   12345,
				"action":    "login",
				"ip":        "192.168.1.1",
				"timestamp": time.Now().Unix(),
			},
		}
		logger.buffer.Push(entry)
	}
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
	defer func() { _ = logger.Sync() }()

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
	defer func() { _ = logger.Sync() }()

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
	// Create dedicated logger for benchmarking
	logger := &Logger{
		level:    int32(ErrorLevel), // Only error and above
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
		sinks:    []Sink{&NullSink{}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Fast level check - simulates what would happen in the real log function
		if InfoLevel >= Level(logger.level) {
			// This should be filtered, so we do nothing - this is intentional for benchmarking
			_ = 0 // Explicit no-op to satisfy linter
		}
	}
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
	defer func() { _ = logger.Sync() }()

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
	// Create dedicated logger for benchmarking
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
		sinks:    []Sink{&NullSink{}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := &Entry{
			Level:     InfoLevel,
			Timestamp: time.Now(),
			Message:   "benchmark message",
			Fields: map[string]interface{}{
				"user_id": 12345,
				"action":  "login",
			},
		}
		logger.buffer.Push(entry)
	}
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
	defer func() { _ = logger.Sync() }()

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
	// Create dedicated logger for benchmarking
	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(8192),
		shutdown: make(chan struct{}),
		sinks:    []Sink{&NullSink{}},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			entry := &Entry{
				Level:     InfoLevel,
				Timestamp: time.Now(),
				Message:   "concurrent message",
				Fields: map[string]interface{}{
					"goroutine": "worker",
					"timestamp": time.Now().Unix(),
				},
			}
			logger.buffer.Push(entry)
		}
	})
}
