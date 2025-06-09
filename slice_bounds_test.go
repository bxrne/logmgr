package logmgr

import (
	"sync"
	"testing"
	"time"
)

// TestSliceBoundsProtection tests that the worker flush function properly handles
// edge cases that could cause slice bounds out of range panics
func TestSliceBoundsProtection(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	// Test 1: Worker with nil logger
	t.Run("nil_logger", func(t *testing.T) {
		worker := &Worker{
			id:       0,
			logger:   nil,
			batch:    make([]*Entry, 10),
			shutdown: make(chan struct{}),
		}
		
		// Should not panic
		worker.flush()
	})

	// Test 2: Worker with nil batch
	t.Run("nil_batch", func(t *testing.T) {
		logger := &Logger{
			level:    int32(InfoLevel),
			buffer:   NewRingBuffer(8),
			shutdown: make(chan struct{}),
		}
		
		worker := &Worker{
			id:       0,
			logger:   logger,
			batch:    nil,
			shutdown: make(chan struct{}),
		}
		
		// Should not panic
		worker.flush()
	})

	// Test 3: Worker with zero-length batch
	t.Run("zero_length_batch", func(t *testing.T) {
		logger := &Logger{
			level:    int32(InfoLevel),
			buffer:   NewRingBuffer(8),
			shutdown: make(chan struct{}),
		}
		
		worker := &Worker{
			id:       0,
			logger:   logger,
			batch:    make([]*Entry, 0), // Zero length
			shutdown: make(chan struct{}),
		}
		
		// Should not panic
		worker.flush()
	})

	// Test 4: Concurrent shutdown stress test
	t.Run("concurrent_shutdown", func(t *testing.T) {
		logger := &Logger{
			level:    int32(InfoLevel),
			buffer:   NewRingBuffer(64),
			shutdown: make(chan struct{}),
		}
		
		logger.entryPool = sync.Pool{
			New: func() interface{} {
				return &Entry{Fields: make(map[string]interface{})}
			},
		}
		
		// Add some test sinks
		sink := &boundsTestSink{}
		logger.sinks = []Sink{sink}
		
		// Create worker
		worker := NewWorker(0, logger, 10)
		if worker == nil {
			t.Fatal("NewWorker returned nil")
		}
		
		// Fill buffer with entries
		for i := 0; i < 50; i++ {
			entry := &Entry{
				Level:     InfoLevel,
				Timestamp: time.Now(),
				Message:   "test message",
				Fields:    make(map[string]interface{}),
			}
			logger.buffer.Push(entry)
		}
		
		// Start multiple goroutines that call flush concurrently
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					worker.flush() // Should not panic
					time.Sleep(time.Microsecond)
				}
			}()
		}
		
		// Stop worker while flushes are happening
		go func() {
			time.Sleep(time.Millisecond)
			worker.Stop()
		}()
		
		wg.Wait()
	})

	// Test 5: RingBuffer with nil entries slice
	t.Run("ringbuffer_nil_entries", func(t *testing.T) {
		rb := NewRingBuffer(8)
		count := rb.Pop(nil) // Pass nil slice
		if count != 0 {
			t.Errorf("Expected count 0 for nil entries, got %d", count)
		}
	})

	// Test 6: RingBuffer with zero-length entries slice
	t.Run("ringbuffer_zero_length_entries", func(t *testing.T) {
		rb := NewRingBuffer(8)
		entries := make([]*Entry, 0)
		count := rb.Pop(entries) // Pass zero-length slice
		if count != 0 {
			t.Errorf("Expected count 0 for zero-length entries, got %d", count)
		}
	})

	// Test 7: NewWorker with invalid parameters
	t.Run("newworker_invalid_params", func(t *testing.T) {
		// Nil logger
		worker := NewWorker(0, nil, 10)
		if worker != nil {
			t.Error("Expected nil worker for nil logger")
		}
		
		// Zero batch size
		logger := &Logger{
			level:    int32(InfoLevel),
			buffer:   NewRingBuffer(8),
			shutdown: make(chan struct{}),
		}
		worker = NewWorker(0, logger, 0)
		if worker == nil {
			t.Error("Expected non-nil worker for zero batch size")
		} else if len(worker.batch) != 256 { // Should default to 256
			t.Errorf("Expected batch size 256 for zero input, got %d", len(worker.batch))
		}
		
		// Negative batch size
		worker = NewWorker(0, logger, -5)
		if worker == nil {
			t.Error("Expected non-nil worker for negative batch size")
		} else if len(worker.batch) != 256 { // Should default to 256
			t.Errorf("Expected batch size 256 for negative input, got %d", len(worker.batch))
		}
	})

	// Test 8: Worker Run with nil worker
	t.Run("worker_run_nil_checks", func(t *testing.T) {
		// This should not panic even with nil worker
		var worker *Worker
		worker = &Worker{
			id:       0,
			logger:   nil, // Nil logger
			batch:    make([]*Entry, 10),
			shutdown: make(chan struct{}),
		}
		
		// Should exit immediately without panic
		worker.Run()
	})

	// Test 9: Worker Stop with nil worker
	t.Run("worker_stop_nil_checks", func(t *testing.T) {
		var worker *Worker
		worker = nil
		worker.Stop() // Should not panic
		
		// Test with valid worker
		worker = &Worker{
			shutdown: make(chan struct{}),
		}
		worker.Stop() // Should not panic
		worker.Stop() // Second call should not panic
	})
}

// TestSliceBoundsRaceCondition specifically tests for race conditions that could
// cause slice bounds errors during high-load scenarios
func TestSliceBoundsRaceCondition(t *testing.T) {
	resetGlobalLogger()
	defer resetGlobalLogger()

	logger := &Logger{
		level:    int32(InfoLevel),
		buffer:   NewRingBuffer(1024),
		shutdown: make(chan struct{}),
	}
	
	logger.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{Fields: make(map[string]interface{})}
		},
	}
	
	// Add test sink
	sink := &boundsTestSink{}
	logger.sinks = []Sink{sink}
	
	// Create multiple workers
	numWorkers := 5
	workers := make([]*Worker, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = NewWorker(i, logger, 50)
		if workers[i] == nil {
			t.Fatalf("NewWorker %d returned nil", i)
		}
	}
	
	// Start producers that fill the buffer
	var producerWG sync.WaitGroup
	stopProducers := make(chan struct{})
	
	for i := 0; i < 3; i++ {
		producerWG.Add(1)
		go func(id int) {
			defer producerWG.Done()
			for {
				select {
				case <-stopProducers:
					return
				default:
					entry := &Entry{
						Level:     InfoLevel,
						Timestamp: time.Now(),
						Message:   "race test message",
						Fields:    make(map[string]interface{}),
					}
					logger.buffer.Push(entry)
				}
			}
		}(i)
	}
	
	// Start consumers (workers) that flush the buffer
	var consumerWG sync.WaitGroup
	stopConsumers := make(chan struct{})
	
	for i := 0; i < numWorkers; i++ {
		consumerWG.Add(1)
		go func(worker *Worker) {
			defer consumerWG.Done()
			for {
				select {
				case <-stopConsumers:
					worker.flush() // Final flush
					return
				default:
					worker.flush() // Should not panic
					time.Sleep(time.Microsecond)
				}
			}
		}(workers[i])
	}
	
	// Let it run for a short time
	time.Sleep(100 * time.Millisecond)
	
	// Stop producers first
	close(stopProducers)
	producerWG.Wait()
	
	// Stop consumers
	close(stopConsumers)
	consumerWG.Wait()
	
	// Stop all workers
	for _, worker := range workers {
		worker.Stop()
	}
}

// boundsTestSink is a simple sink for testing that doesn't do any actual I/O
type boundsTestSink struct {
	entries [][]*Entry
	mu      sync.Mutex
}

func (s *boundsTestSink) Write(entries []*Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Make a copy to avoid slice sharing issues
	batch := make([]*Entry, len(entries))
	copy(batch, entries)
	s.entries = append(s.entries, batch)
	return nil
}

func (s *boundsTestSink) Close() error {
	return nil
}

func (s *boundsTestSink) GetEntries() [][]*Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.entries
}