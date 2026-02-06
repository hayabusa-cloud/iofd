// ©Hayabusa Cloud Co., Ltd. 2026. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

/*
Package iofd provides minimal file descriptor abstractions and specialized
Linux handles for the Go ecosystem. It serves as the canonical handle
abstraction for high-performance I/O systems.

# Design Principles

Zero-Overhead: All kernel interactions use code.hybscloud.com/zcall exclusively,
bypassing Go's standard library syscall hooks. This eliminates runtime scheduler
overhead in hot paths.

Atomic Lifecycle: FD uses atomic operations for concurrent-safe access. Close()
is idempotent and can be called multiple times safely.

Non-Blocking Default: Constructors default to O_NONBLOCK | O_CLOEXEC. Operations
return iox.ErrWouldBlock when they would block.

# Supported Architectures

The core FD type works on all Unix platforms. Specialized handles (EventFD,
TimerFD, PidFD, MemFD, SignalFD) require Linux.

	Platform         FD Core   EventFD   TimerFD   PidFD   MemFD   SignalFD
	Linux/amd64        ✓         ✓         ✓        ✓       ✓        ✓
	Linux/arm64        ✓         ✓         ✓        ✓       ✓        ✓
	Darwin/arm64       ✓         -         -        -       -        -
	FreeBSD/amd64      ✓         -         -        -       -        -

# Usage

Basic EventFD usage:

	efd, err := iofd.NewEventFD(0)
	if err != nil {
		return err
	}
	defer efd.Close()

	// Signal from one goroutine
	efd.Signal(1)

	// Wait in another
	val, err := efd.Wait()

Timer example:

	tfd, err := iofd.NewTimerFD()
	if err != nil {
		return err
	}
	defer tfd.Close()

	// Arm for 100ms one-shot
	tfd.ArmDuration(100*time.Millisecond, 0)

	// Poll or use with epoll/io_uring, then read expirations
	count, err := tfd.Expirations()

# Safety Considerations

Atomic Operations: All FD access uses atomic load/store. Multiple goroutines
can safely call Valid(), Raw(), and Close() concurrently.

Valid Check: Always check Valid() or handle ErrClosed before performing
operations on potentially closed descriptors.

Close Idempotency: Close() can be called multiple times safely. After Close(),
Raw() returns -1 and operations return ErrClosed.

MappedRegion Lifetime: When using MemFD.Mmap(), the returned MappedRegion's
Bytes() slice is only valid while the region is mapped. Call Unmap() when done.

# Dependencies

This package depends on:
  - code.hybscloud.com/zcall — Zero-overhead syscalls
  - code.hybscloud.com/iox — Semantic errors (ErrWouldBlock)
*/
package iofd
