package logmgr

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileSink writes log entries to a file with automatic rotation support.
// It implements the Sink interface and provides both time-based and size-based rotation.
// The sink is safe for concurrent use and uses buffered I/O for optimal performance.
//
// Features:
//   - Automatic file rotation based on age and/or size
//   - Thread-safe concurrent writes
//   - Buffered I/O with 16KB buffer
//   - Automatic directory creation
//   - Timestamped rotated files
//
// Example:
//
//	fileSink, err := logmgr.NewFileSink("app.log", 24*time.Hour, 100*1024*1024)
//	if err != nil {
//	  panic(err)
//	}
//	logmgr.AddSink(fileSink)
type FileSink struct {
	filename     string        // Path to the log file
	maxAge       time.Duration // Maximum age before rotation (0 = no age limit)
	maxSize      int64         // Maximum size in bytes before rotation (0 = no size limit)
	file         *os.File      // Current log file handle
	writer       *bufio.Writer // Buffered writer for performance
	mu           sync.Mutex    // Protects concurrent access
	currentSize  int64         // Current file size in bytes
	lastRotation time.Time     // Time of last rotation
}

// NewFileSink creates a new file sink with the specified rotation parameters.
//
// Parameters:
//   - filename: Path to the log file (directories will be created if needed)
//   - maxAge: Maximum age before rotation (0 disables age-based rotation)
//   - maxSize: Maximum size in bytes before rotation (0 disables size-based rotation)
//
// The sink will rotate the file when either condition is met. Rotated files are
// renamed with a timestamp suffix (e.g., "app_2024-01-15_10-30-45.log").
//
// Example:
//
//	// Rotate daily or when file reaches 100MB
//	sink, err := logmgr.NewFileSink("logs/app.log", 24*time.Hour, 100*1024*1024)
//	if err != nil {
//	  return err
//	}
//	defer sink.Close()
func NewFileSink(filename string, maxAge time.Duration, maxSize int64) (*FileSink, error) {
	fs := &FileSink{
		filename:     filename,
		maxAge:       maxAge,
		maxSize:      maxSize,
		lastRotation: time.Now(),
	}

	if err := fs.openFile(); err != nil {
		return nil, err
	}

	return fs, nil
}

// NewDefaultFileSink creates a file sink with default settings for backward compatibility.
// Uses a default maximum size of 100MB with the specified age limit.
//
// This function will panic if the file cannot be created, making it suitable for
// initialization code where errors should be fatal.
//
// Example:
//
//	sink := logmgr.NewDefaultFileSink("app.log", 24*time.Hour)
//	logmgr.AddSink(sink)
func NewDefaultFileSink(filename string, maxAge time.Duration) *FileSink {
	fs, err := NewFileSink(filename, maxAge, 100*1024*1024) // 100MB default
	if err != nil {
		panic(fmt.Sprintf("failed to create file sink: %v", err))
	}
	return fs
}

// openFile opens the log file for writing and initializes the buffered writer.
// If the file doesn't exist, it will be created. If it exists, it will be opened
// in append mode to preserve existing log entries.
func (fs *FileSink) openFile() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(fs.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file in append mode
	file, err := os.OpenFile(fs.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Get current file size
	stat, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return fmt.Errorf("failed to get file stat: %w (close error: %v)", err, closeErr)
		}
		return err
	}

	fs.file = file
	fs.writer = bufio.NewWriterSize(file, 16384) // 16KB buffer
	fs.currentSize = stat.Size()

	return nil
}

// shouldRotate checks if the file should be rotated based on age or size limits.
// Returns true if either the maximum age or maximum size has been exceeded.
func (fs *FileSink) shouldRotate() bool {
	now := time.Now()

	// Check age-based rotation
	if fs.maxAge > 0 && now.Sub(fs.lastRotation) >= fs.maxAge {
		return true
	}

	// Check size-based rotation
	if fs.maxSize > 0 && fs.currentSize >= fs.maxSize {
		return true
	}

	return false
}

// rotate performs file rotation by closing the current file, renaming it with
// a timestamp, and opening a new file. The rotated file format is:
// "basename_YYYY-MM-DD_HH-MM-SS.ext"
//
// Example: "app.log" becomes "app_2024-01-15_10-30-45.log"
func (fs *FileSink) rotate() error {
	// Close current file
	if fs.writer != nil {
		if err := fs.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush writer during rotation: %w", err)
		}
	}
	if fs.file != nil {
		if err := fs.file.Close(); err != nil {
			return fmt.Errorf("failed to close file during rotation: %w", err)
		}
	}

	// Rename current file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	ext := filepath.Ext(fs.filename)
	base := fs.filename[:len(fs.filename)-len(ext)]
	rotatedName := fmt.Sprintf("%s_%s%s", base, timestamp, ext)

	// Only rename if the file exists and has content
	if stat, err := os.Stat(fs.filename); err == nil && stat.Size() > 0 {
		if err := os.Rename(fs.filename, rotatedName); err != nil {
			return fmt.Errorf("failed to rename file during rotation: %w", err)
		}
	}

	// Open new file
	fs.lastRotation = time.Now()
	fs.currentSize = 0

	return fs.openFile()
}

// Write writes a batch of log entries to the file in JSON format.
// Each entry is written as a single line of JSON followed by a newline.
// This method is safe for concurrent use and will automatically rotate
// the file if rotation conditions are met.
//
// The method will skip any entries that fail to marshal to JSON rather than
// failing the entire batch. File rotation is checked before writing the batch.
//
// Example output in file:
//
//	{"level":"info","timestamp":"2024-01-15T10:30:45.123Z","message":"User logged in","user_id":12345}
//	{"level":"error","timestamp":"2024-01-15T10:30:46.456Z","message":"Database error","error":"connection timeout"}
func (fs *FileSink) Write(entries []*Entry) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Check if sink has been closed
	if fs.writer == nil || fs.file == nil {
		return fmt.Errorf("cannot write to closed file sink")
	}

	// Check if rotation is needed
	if fs.shouldRotate() {
		if err := fs.rotate(); err != nil {
			return err
		}
	}

	// Pre-allocate buffer for the entire batch to reduce system calls
	var batchBuffer []byte
	if len(entries) > 0 {
		// Estimate buffer size: ~200 bytes per entry average
		batchBuffer = make([]byte, 0, len(entries)*200)
	}

	// Write entries to batch buffer
	for _, entry := range entries {
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
		n, err := fs.writer.Write(batchBuffer)
		if err != nil {
			return err
		}
		fs.currentSize += int64(n)
	}

	// Flush the buffer
	return fs.writer.Flush()
}

// Close flushes any remaining buffered data and closes the file sink.
// This method should be called during application shutdown to ensure all
// log entries are written and the file is properly closed.
// It's safe to call Close multiple times.
//
// Example:
//
//	defer fileSink.Close()
func (fs *FileSink) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	var err error

	if fs.writer != nil {
		if flushErr := fs.writer.Flush(); flushErr != nil {
			err = fmt.Errorf("failed to flush writer during close: %w", flushErr)
		}
		fs.writer = nil
	}

	if fs.file != nil {
		if closeErr := fs.file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		fs.file = nil
	}

	return err
}

// AsyncFileSink is a high-performance asynchronous file sink that writes log entries
// in a background goroutine. This provides maximum performance by decoupling log
// writing from the application's critical path.
//
// Features:
//   - Non-blocking writes with configurable buffer
//   - Background writer goroutine
//   - Automatic fallback to synchronous writes when buffer is full
//   - All FileSink features (rotation, buffering, etc.)
//
// Example:
//
//	asyncSink, err := logmgr.NewAsyncFileSink("app.log", 24*time.Hour, 100*1024*1024, 1000)
//	if err != nil {
//	  panic(err)
//	}
//	defer asyncSink.Close()
//	logmgr.AddSink(asyncSink)
type AsyncFileSink struct {
	*FileSink                // Embedded FileSink for actual file operations
	buffer    chan []*Entry  // Channel buffer for async writes
	done      chan struct{}  // Shutdown signal
	wg        sync.WaitGroup // Wait group for background goroutine
	closed    bool           // Track if sink has been closed
	mu        sync.Mutex     // Protect closed flag
}

// NewAsyncFileSink creates a new asynchronous file sink with the specified parameters.
//
// Parameters:
//   - filename: Path to the log file
//   - maxAge: Maximum age before rotation (0 disables age-based rotation)
//   - maxSize: Maximum size in bytes before rotation (0 disables size-based rotation)
//   - bufferSize: Size of the internal channel buffer for async writes
//
// The background writer goroutine is started automatically and will process
// log entries every 100ms or when entries are available.
//
// Example:
//
//	// High-performance async sink with 1000-entry buffer
//	sink, err := logmgr.NewAsyncFileSink("logs/app.log", 24*time.Hour, 100*1024*1024, 1000)
//	if err != nil {
//	  return err
//	}
//	defer sink.Close()
func NewAsyncFileSink(filename string, maxAge time.Duration, maxSize int64, bufferSize int) (*AsyncFileSink, error) {
	fs, err := NewFileSink(filename, maxAge, maxSize)
	if err != nil {
		return nil, err
	}

	afs := &AsyncFileSink{
		FileSink: fs,
		buffer:   make(chan []*Entry, bufferSize),
		done:     make(chan struct{}),
	}

	// Start background writer
	afs.wg.Add(1)
	go afs.writerLoop()

	return afs, nil
}

// writerLoop runs the background writer goroutine that processes log entries
// from the channel buffer. It flushes entries every 100ms or when available.
func (afs *AsyncFileSink) writerLoop() {
	defer afs.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // Flush every 100ms
	defer ticker.Stop()

	for {
		select {
		case entries := <-afs.buffer:
			// Log write errors but don't crash the writer loop
			if err := afs.FileSink.Write(entries); err != nil {
				// In a real application, you might want to log this error
				// to a fallback location or handle it appropriately
				_ = err
			}
		case <-ticker.C:
			// Periodic flush (handled by FileSink.Write)
		case <-afs.done:
			// Drain remaining entries
			for {
				select {
				case entries := <-afs.buffer:
					if err := afs.FileSink.Write(entries); err != nil {
						// Log write errors but continue draining
						_ = err
					}
				default:
					return
				}
			}
		}
	}
}

// Write writes log entries to the async buffer for background processing.
// This method is non-blocking and returns immediately in most cases.
//
// If the internal buffer is full, the method falls back to synchronous writing
// to prevent blocking the application. A copy of the entries is made since
// the original slice may be reused by the caller.
//
// This method is safe for concurrent use.
func (afs *AsyncFileSink) Write(entries []*Entry) error {
	afs.mu.Lock()
	closed := afs.closed
	afs.mu.Unlock()

	if closed {
		return fmt.Errorf("cannot write to closed async file sink")
	}

	// Make a copy of entries since they might be reused
	entriesCopy := make([]*Entry, len(entries))
	copy(entriesCopy, entries)

	select {
	case afs.buffer <- entriesCopy:
		return nil
	default:
		// Buffer full, write synchronously as fallback
		return afs.FileSink.Write(entries)
	}
}

// Close gracefully shuts down the async file sink by stopping the background
// writer and ensuring all buffered entries are written to disk.
//
// This method will block until all pending writes are completed, ensuring
// no log entries are lost during shutdown. It's safe to call Close multiple times.
//
// Example:
//
//	defer asyncSink.Close()
func (afs *AsyncFileSink) Close() error {
	afs.mu.Lock()
	defer afs.mu.Unlock()

	// Check if already closed
	if afs.closed {
		return nil
	}
	afs.closed = true

	close(afs.done)
	afs.wg.Wait()
	return afs.FileSink.Close()
}
