# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
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

This function takes ownership of `rawFd`, a descriptor for a temporary file opened with `O_RDWR`. The file contains `ab` and its offset is 0.

```go
fd := iofd.NewFD(rawFd)
defer fd.Close()

dup, err := fd.Dup() // Separate descriptors, shared file offset.
if err != nil {
	return 0, err
}
defer dup.Close()

var buf [1]byte
n, err := fd.Read(buf[:]) // Reads 'a' and advances the shared offset to 1.
if err != nil {
	return 0, err
}
return dup.Write(buf[:n]) // Overwrites 'b', leaving "aa".
```

With a borrowed `rawFd`, keep the original open until `Dup` returns and close only the duplicate.

| Operation | Effect |
|-----------|------|
| `fd.Raw()` | Returns the borrowed descriptor number (`int32`). Keep `fd` open while using it. |
| `fd.Valid()` | Reports whether the stored descriptor number is nonnegative. Another goroutine can close `fd` immediately afterward. |
| `fd.SetNonblock(true)` | Sets `O_NONBLOCK` for `fd` and its duplicates. |
| `fd.SetCloexec(true)` | Sets `FD_CLOEXEC` on `fd` alone. `Dup` sets it on the new descriptor automatically. |

### Constructor Flags

| Constructor | Creation flags |
|-------------|---------------|
| `NewEventFD` | `EFD_NONBLOCK`, `EFD_CLOEXEC` |
| `NewEventFDSemaphore` | `EFD_SEMAPHORE`, `EFD_NONBLOCK`, `EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK`, `TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK`, `SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK` |
| `NewPidFDBlocking` | `0` |
| `NewMemFD` | `MFD_CLOEXEC` |
| `NewMemFDSealed` | `MFD_CLOEXEC`, `MFD_ALLOW_SEALING` |
| `NewMemFDHugeTLB` | `MFD_CLOEXEC`, `MFD_HUGETLB` |

`pidfd_open` sets `FD_CLOEXEC` even when its flags argument is `0`. `memfd_create` has no flag for nonblocking I/O.

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
- **Duplication**: Use `Dup()` or `PidFD.GetFD()` when another closeable descriptor is required
- **MappedRegion Lifetime**: `Bytes()` slice is only valid while the region is mapped

## License

MIT — see [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
