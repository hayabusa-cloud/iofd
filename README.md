# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/iofd)](https://goreportcard.com/report/github.com/hayabusa-cloud/iofd)
[![Codecov](https://codecov.io/gh/hayabusa-cloud/iofd/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/iofd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Universal file descriptor abstractions for Unix systems in Go.

Language: **English** | [简体中文](./README.zh-CN.md) | [Español](./README.es.md) | [日本語](./README.ja.md) | [Français](./README.fr.md)

## Overview

`iofd` provides minimal file descriptor abstractions and specialized Linux handles for the Go ecosystem. It serves as the canonical handle abstraction for high-performance I/O systems.

### Key Features

- **Zero Overhead**: All kernel interactions via `zcall` assembly, bypassing Go's syscall hooks
- **Zero-Allocation Hot Paths**: Fixed-size `EventFD`, `TimerFD`, and `SignalFD` success paths keep syscall arguments stack-backed
- **Specialized Handles**: Linux-specific `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD`
- **Cross-Platform Core**: Base `FD` operations work on Linux, Darwin, and FreeBSD
- **Explicit Ownership**: `FD` close idempotence applies to one descriptor cell; close after users are drained and use `Dup` for an independent close owner

## Installation

```bash
go get code.hybscloud.com/iofd
```

## Quick Start

### EventFD Signaling

```go
efd, _ := iofd.NewEventFD(0)
defer efd.Close()

efd.Signal(1)
val, _ := efd.Wait() // val == 1
```

### TimerFD

```go
tfd, _ := iofd.NewTimerFD()
defer tfd.Close()

// One-shot timer at 100ms
tfd.ArmDuration(100*time.Millisecond, 0)
// ... poll/epoll/io_uring wait ...
count, _ := tfd.Expirations() // count == 1
```

### Error Handling

```go
_, err := efd.Wait()
if errors.Is(err, iox.ErrWouldBlock) {
    // Non-blocking, no data available - retry later
} else if errors.Is(err, iofd.ErrClosed) {
    // FD was closed
} else if err != nil {
    // Other error
}
```

## API

### Core Types

| Type | Description |
|------|-------------|
| `FD` | File descriptor cell with atomic same-cell lifecycle operations |
| `EventFD` | Linux eventfd for inter-thread signaling |
| `TimerFD` | Linux timerfd for high-resolution timers |
| `PidFD` | Linux pidfd for race-free process management |
| `MemFD` | Linux memfd for anonymous memory-backed files |
| `MappedRegion` | Memory-mapped region for zero-copy access |
| `SignalFD` | Linux signalfd for synchronous signal handling |

### Interfaces

| Interface | Methods | Description |
|-----------|---------|-------------|
| `PollFd` | `Fd() int` | Pollable file descriptor |
| `PollCloser` | `Fd()`, `Close()` | Closeable pollable descriptor |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Full I/O handle |
| `Signaler` | `Signal()`, `Wait()` | Signaling mechanism |
| `Timer` | `Arm()`, `Disarm()` | Timer handle |

### FD Operations

```go
// Create FD from raw descriptor
fd := iofd.NewFD(rawFd)
// NewFD takes close ownership. Do not close copied FD values;
// close only after users are drained. Use fd.Dup() for an
// independent descriptor owner.

// Atomic operations
fd.Raw()           // Get raw int32 value
fd.Valid()         // Check if valid (non-negative)
fd.Close()         // Same-cell close after drain

// I/O operations
fd.Read(buf)       // Read bytes
fd.Write(buf)      // Write bytes

// Descriptor flags
fd.SetNonblock(true)   // Set O_NONBLOCK
fd.SetCloexec(true)    // Set FD_CLOEXEC
fd.Dup()               // Duplicate with CLOEXEC
```

### Constructor Flags

| Constructor | Default flags |
|-------------|---------------|
| `NewEventFD`, `NewEventFDSemaphore` | `EFD_NONBLOCK | EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK | TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK | SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK`; close-on-exec is set by the kernel |
| `NewPidFDBlocking` | Blocking pidfd; close-on-exec is still set by the kernel |
| `NewMemFD`, `NewMemFDSealed`, `NewMemFDHugeTLB` | `MFD_CLOEXEC` plus memfd-specific flags; no creation-time nonblocking flag exists |

### MemFD Memory Mapping

```go
// Create memfd and set size
mfd, _ := iofd.NewMemFD("buffer")
mfd.Truncate(4096)

// Map for zero-copy access
region, _ := mfd.Mmap(4096, iofd.PROT_READ|iofd.PROT_WRITE)
data := region.Bytes()  // []byte backed by shared memory
copy(data, []byte("hello"))

// Cleanup
region.Unmap()
mfd.Close()
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                      │
├─────────────────────────────────────────────────────────┤
│  EventFD │ TimerFD │ MemFD │ PidFD │ SignalFD │   FD   │
├─────────────────────────────────────────────────────────┤
│                        iofd                              │
├─────────────────────────────────────────────────────────┤
│                       zcall                              │
│                (zero-overhead syscalls)                  │
├─────────────────────────────────────────────────────────┤
│                   Linux Kernel                           │
└─────────────────────────────────────────────────────────┘
```

## Platform Support

| Platform | FD Core | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|----------|---------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Note**: Specialized handles (`EventFD`, `TimerFD`, etc.) are Linux-specific kernel primitives. On Darwin and FreeBSD, only the core `FD` type is available.

## Safety Considerations

- **Atomic Operations**: `Raw`, `Valid`, and same-cell `Close` use atomic access; callers still drain users before `Close()`
- **Ownership**: `Close()` is idempotent for the same `FD` cell; copied open `FD` values are not independent owners
- **Close Ordering**: Call `Close()` only after in-flight operations and borrowed raw descriptor users are drained
- **Valid Check**: Use `Valid()` before operations on potentially closed descriptors
- **Duplication**: Use `Dup()` or `PidFD.GetFD()` when another closeable descriptor is required
- **MappedRegion Lifetime**: `Bytes()` slice is only valid while the region is mapped

## License

MIT — see [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
