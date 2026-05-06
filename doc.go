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

Allocation Discipline: Fixed-size EventFD, TimerFD, and SignalFD success paths
keep syscall argument storage stack-backed and do not allocate heap memory.
Caller-provided Into methods keep result storage caller-owned.

Atomic Lifecycle: FD uses atomic operations through one addressable descriptor
cell. Close() is idempotent for that same cell. Copying an open FD does not
duplicate kernel ownership; use Dup() for an independent close-capable
descriptor.

Non-Blocking Constructors: EventFD, TimerFD, SignalFD, and NewPidFD create
non-blocking descriptors by default. MemFD is created close-on-exec but has no
creation-time nonblocking flag, and NewPidFDBlocking intentionally creates a
blocking pidfd. Non-blocking operations return iox.ErrWouldBlock when they
would block.

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

Atomic Operations: FD access uses atomic load/store through one addressable FD
cell. Atomicity protects same-cell state transitions; it does not make
descriptor-number reuse safe if Close races with in-flight operations.

Valid Check: Always check Valid() or handle ErrClosed before performing
operations on potentially closed descriptors.

Close Idempotency: Close() can be called multiple times safely on the same FD
cell. Do not close copied FD values; they are not independent owners. After
Close(), Raw() returns -1 and operations return ErrClosed.

Close Ordering: Call Close only after all in-flight operations and borrowed raw
descriptor users are drained. The package does not add hidden synchronization
around kernel descriptor reuse.

MappedRegion Lifetime: When using MemFD.Mmap(), the returned MappedRegion's
Bytes() slice is only valid while the region is mapped. Call Unmap() when done.

# Dependencies

This package depends on:
  - code.hybscloud.com/zcall — Zero-overhead syscalls
  - code.hybscloud.com/iox — Semantic errors (ErrWouldBlock)
*/
package iofd
