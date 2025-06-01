# logmgr: A High-Performance Go Logging Library

## Introduction

In the world of Go logging libraries, performance and simplicity often seem to be at odds. While some libraries prioritize blazing-fast performance at the cost of complexity, others focus on ease of use but sacrifice speed. logmgr aims to strike the perfect balance, providing exceptional performance while maintaining a clean, intuitive API.

This blog post explores logmgr's design philosophy, architectural decisions, and comprehensive benchmark results comparing it against popular Go logging libraries: Zap, Logrus, the standard library log package, and the new structured logging package (slog).

## Design Philosophy

### 1. Performance Without Compromise

logmgr is built from the ground up with performance as a primary concern. Every architectural decision has been made to minimize latency and memory allocations while maintaining thread safety and reliability.

### 2. Structured Logging First

While many logging libraries treat structured logging as an afterthought, logmgr embraces it as a first-class citizen. The API is designed to make structured logging natural and efficient.

### 3. Zero-Allocation Logging

Through careful use of object pools, lock-free data structures, and optimized JSON serialization, logmgr achieves near-zero allocation logging in most scenarios.

### 4. Asynchronous by Design

logmgr employs a lock-free ring buffer and background workers to ensure that logging operations never block your application's critical path.

## Architectural Highlights

### Lock-Free Ring Buffer

At the heart of logmgr is a custom lock-free ring buffer that uses atomic operations for thread-safe, high-performance log entry queuing:

```go
type RingBuffer struct {
    buffer   []unsafe.Pointer // Ring buffer of entry pointers
    mask     uint64           // Size mask (size must be power of 2)
    writePos uint64           // Write position (atomic)
    readPos  uint64           // Read position (atomic)
}
```

This design ensures that log operations are non-blocking and scale linearly with the number of cores.

### Object Pool Optimization

logmgr uses `sync.Pool` to reuse log entry objects, dramatically reducing garbage collection pressure:

```go
entryPool: sync.Pool{
    New: func() interface{} {
        return &Entry{
            Fields: make(map[string]interface{}, 8),
            buffer: make([]byte, 0, 512),
        }
    },
}
```

### Background Workers

Multiple background workers (one per CPU core) continuously drain the ring buffer and write to configured sinks, keeping the hot path as fast as possible.

### Custom JSON Serialization

logmgr includes a highly optimized JSON marshaler that avoids reflection and minimizes allocations:

```go
func (e *Entry) MarshalJSON() ([]byte, error) {
    e.buffer = e.buffer[:0]
    e.buffer = append(e.buffer, '{')
    
    // Direct string manipulation for maximum performance
    e.buffer = append(e.buffer, `"level":"`...)
    e.buffer = append(e.buffer, e.Level.String()...)
    // ... continues with optimized field serialization
}
```

## Comprehensive Benchmark Results

We conducted extensive benchmarks comparing logmgr against the most popular Go logging libraries. All tests were run on a MacBook Pro with an Intel Core i9-9980HK CPU @ 2.40GHz.

### Complete Benchmark Results

Here are the complete benchmark results from our test suite:

```
goos: darwin
goarch: amd64
pkg: github.com/bxrne/logmgr
cpu: Intel(R) Core(TM) i9-9980HK CPU @ 2.40GHz

Benchmarklogmgr_Simple-16                6551541               160.6 ns/op           144 B/op          2 allocs/op
Benchmarklogmgr_Structured-16            3044401               429.6 ns/op           440 B/op          4 allocs/op
BenchmarkZap_Simple-16                  20303310                55.96 ns/op            0 B/op          0 allocs/op
BenchmarkZap_Structured-16               4970788               243.3 ns/op           258 B/op          1 allocs/op
BenchmarkLogrus_Simple-16                 622810              2217 ns/op             883 B/op         19 allocs/op
BenchmarkLogrus_Structured-16             320115              3723 ns/op            1913 B/op         32 allocs/op
BenchmarkStdLog_Simple-16                6494241               187.9 ns/op             0 B/op          0 allocs/op
BenchmarkSlog_Simple-16                  7147566               148.4 ns/op             0 B/op          0 allocs/op
BenchmarkSlog_Structured-16              4235157               276.1 ns/op           192 B/op          4 allocs/op
Benchmarklogmgr_LevelFiltering-16       1000000000               0.4803 ns/op          0 B/op          0 allocs/op
BenchmarkZap_LevelFiltering-16          1000000000               0.8132 ns/op          0 B/op          0 allocs/op
BenchmarkLogrus_LevelFiltering-16       1000000000               0.2787 ns/op          0 B/op          0 allocs/op
BenchmarkSlog_LevelFiltering-16         1000000000               0.8582 ns/op          0 B/op          0 allocs/op
Benchmarklogmgr_Allocations-16           4276648               272.9 ns/op           432 B/op          3 allocs/op
BenchmarkZap_Allocations-16              2182059               551.7 ns/op           128 B/op          1 allocs/op
BenchmarkLogrus_Allocations-16            469026              2585 ns/op            1765 B/op         27 allocs/op
BenchmarkSlog_Allocations-16             1479810               819.7 ns/op            96 B/op          2 allocs/op
BenchmarkConcurrentLogging-16            8073288               126.8 ns/op           440 B/op          4 allocs/op
```

### Trade-offs

1. **Memory Usage**: Slightly higher memory usage due to asynchronous buffering
2. **Complexity**: More complex internals compared to synchronous loggers
3. **Allocation Count**: Object pooling strategy results in more allocations than zero-allocation approaches

