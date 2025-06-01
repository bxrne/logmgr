package main

import (
	"time"

	"github.com/bxrne/logmgr"
)

func main() {
	// Initialize the logger
	logmgr.SetLevel(logmgr.DebugLevel) // Set the desired log level (optional)

	// Add sinks for output
	logmgr.AddSink(logmgr.DefaultConsoleSink) // Console output

	// Add file sink with rotation (24 hours, 100MB max)
	fileSink, err := logmgr.NewFileSink("app.log", 24*time.Hour, 100*1024*1024)
	if err != nil {
		panic(err)
	}

	// Add file sink to the logger
	logmgr.AddSink(fileSink)

	// Log a debug message
	logmgr.Debug("This is a debug message")

	// Log an info message with structured fields
	logmgr.Info("User logged in",
		logmgr.Field("user_id", 12345),
		logmgr.Field("action", "login"),
		logmgr.Field("ip", "192.168.1.1"),
	)

	// Log a warning message with structured fields
	logmgr.Warn("High memory usage",
		logmgr.Field("memory_percent", 85.5),
		logmgr.Field("threshold", 80.0),
	)

	// Log an error message with structured fields
	logmgr.Error("Database connection failed",
		logmgr.Field("error", "connection timeout"),
		logmgr.Field("host", "db.example.com"),
		logmgr.Field("port", 5432),
		logmgr.Field("retries", 3),
	)

	// Fatal would call os.Exit(1) after flushing logs
	logmgr.Fatal("Critical system failure",
		logmgr.Field("error", "out of memory"),
		logmgr.Field("available_memory", "0MB"),
	)

	// Gracefully shutdown to flush all logs
	logmgr.Shutdown()
}
