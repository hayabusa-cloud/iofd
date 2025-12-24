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
- **Specialized Handles**: Linux-specific `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD`
- **Cross-Platform Core**: Base `FD` operations work on Linux, Darwin, and FreeBSD

## Installation

```bash
go get code.hybscloud.com/iofd
```

## Quick Start

```go
efd, _ := iofd.NewEventFD(0)
efd.Signal(1)
val, _ := efd.Wait() // val == 1
efd.Close()
```

## API

### Core Types

| Type | Description |
|------|-------------|
| `FD` | Universal file descriptor with atomic operations |
| `EventFD` | Linux eventfd for inter-thread signaling |
| `TimerFD` | Linux timerfd for high-resolution timers |
| `PidFD` | Linux pidfd for race-free process management |
| `MemFD` | Linux memfd for anonymous memory-backed files |
| `SignalFD` | Linux signalfd for synchronous signal handling |

### Interfaces

| Interface | Methods | Description |
|-----------|---------|-------------|
| `PollFd` | `Fd() int` | Pollable file descriptor |
| `PollCloser` | `Fd()`, `Close()` | Closeable pollable descriptor |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Full I/O handle |
| `Signaler` | `Signal()`, `Wait()` | Signaling mechanism |
| `Timer` | `Arm()`, `Disarm()`, `Read()` | Timer handle |

### FD Operations

```go
// Create FD from raw descriptor
fd := iofd.NewFD(rawFd)

// Atomic operations
fd.Raw()           // Get raw int32 value
fd.Valid()         // Check if valid (non-negative)
fd.Close()         // Idempotent close

// I/O operations
fd.Read(buf)       // Read bytes
fd.Write(buf)      // Write bytes

// Descriptor flags
fd.SetNonblock(true)   // Set O_NONBLOCK
fd.SetCloexec(true)    // Set FD_CLOEXEC
fd.Dup()               // Duplicate with CLOEXEC
```

## Platform Support

| Platform | FD Core | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|----------|---------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Note**: Specialized handles (`EventFD`, `TimerFD`, etc.) are Linux-specific kernel primitives. On Darwin and FreeBSD, only the core `FD` type is available.

## License

MIT — see [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
