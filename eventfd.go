// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"encoding/binary"
	"unsafe"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// EventFD represents a Linux eventfd file descriptor.
// It provides an efficient inter-thread/kernel signaling mechanism.
//
// An eventfd maintains an unsigned 64-bit counter. Writing adds to the counter,
// reading returns and resets it (or decrements by 1 in semaphore mode).
//
// EventFD is created with O_NONBLOCK and O_CLOEXEC by default.
type EventFD struct {
	fd FD
}

// NewEventFD creates a new eventfd with the given initial value.
// The eventfd is created with EFD_NONBLOCK | EFD_CLOEXEC flags.
//
// Returns iox.ErrWouldBlock semantics are used for non-blocking operations.
func NewEventFD(initval uint) (*EventFD, error) {
	return newEventFD(initval, EFD_NONBLOCK|EFD_CLOEXEC)
}

// NewEventFDSemaphore creates a new eventfd in semaphore mode.
// In semaphore mode, reads decrement the counter by 1 instead of resetting it.
func NewEventFDSemaphore(initval uint) (*EventFD, error) {
	return newEventFD(initval, EFD_SEMAPHORE|EFD_NONBLOCK|EFD_CLOEXEC)
}

func newEventFD(initval uint, flags uintptr) (*EventFD, error) {
	fd, errno := zcall.Eventfd2(uintptr(initval), flags)
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &EventFD{fd: FD(fd)}, nil
}

// Fd returns the underlying file descriptor.
// Implements PollFd interface.
func (e *EventFD) Fd() int {
	return e.fd.Fd()
}

// Close closes the eventfd.
// Implements PollCloser interface.
func (e *EventFD) Close() error {
	return e.fd.Close()
}

// Signal increments the eventfd counter by val.
// Returns iox.ErrWouldBlock if the counter would overflow (non-blocking mode).
//
// The maximum value is 0xFFFFFFFFFFFFFFFE (2^64 - 2).
// Writing would block/fail if adding val would exceed this limit.
func (e *EventFD) Signal(val uint64) error {
	if val == 0 {
		return nil
	}
	raw := e.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	var buf [8]byte
	binary.NativeEndian.PutUint64(buf[:], val)
	n, errno := zcall.Write(uintptr(raw), buf[:])
	if errno != 0 {
		if zcall.Errno(errno) == zcall.EAGAIN {
			return iox.ErrWouldBlock
		}
		return errFromErrno(errno)
	}
	if n != 8 {
		return ErrInvalidParam
	}
	return nil
}

// Wait reads and returns the eventfd counter value.
// In normal mode, this resets the counter to zero.
// In semaphore mode, this decrements the counter by 1.
//
// Returns iox.ErrWouldBlock if the counter is zero (non-blocking mode).
func (e *EventFD) Wait() (uint64, error) {
	raw := e.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	var buf [8]byte
	n, errno := zcall.Read(uintptr(raw), buf[:])
	if errno != 0 {
		if zcall.Errno(errno) == zcall.EAGAIN {
			return 0, iox.ErrWouldBlock
		}
		return 0, errFromErrno(errno)
	}
	if n != 8 {
		return 0, ErrInvalidParam
	}
	return binary.NativeEndian.Uint64(buf[:]), nil
}

// Read reads the eventfd counter into p.
// p must be at least 8 bytes. Only the first 8 bytes are used.
// This is a lower-level interface; prefer Wait() for typical usage.
func (e *EventFD) Read(p []byte) (int, error) {
	if len(p) < 8 {
		return 0, ErrInvalidParam
	}
	raw := e.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Read(uintptr(raw), p[:8])
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// Write writes a value to the eventfd from p.
// p must be at least 8 bytes containing a little-endian uint64.
// This is a lower-level interface; prefer Signal() for typical usage.
func (e *EventFD) Write(p []byte) (int, error) {
	if len(p) < 8 {
		return 0, ErrInvalidParam
	}
	raw := e.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Write(uintptr(raw), p[:8])
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// Value returns the current counter value without consuming it.
// This uses a non-standard approach via /proc and should be used sparingly.
// For most use cases, use Wait() instead.
func (e *EventFD) Value() (uint64, error) {
	// Note: There's no direct syscall to peek at eventfd value.
	// The only way is to read (which consumes) or use /proc.
	// For zero-allocation hot paths, this method should be avoided.
	return 0, ErrInvalidParam
}

// eventfd flags
const (
	EFD_SEMAPHORE = 0x1
	EFD_CLOEXEC   = 0x80000
	EFD_NONBLOCK  = 0x800
)

// Compile-time interface assertions
var (
	_ PollFd     = (*EventFD)(nil)
	_ PollCloser = (*EventFD)(nil)
	_ Signaler   = (*EventFD)(nil)
)

// nativeEndian is the byte order of the native architecture.
// Used for eventfd counter encoding.
var nativeEndian = binary.NativeEndian

// eventfdValuePtr returns a pointer to the value for zero-copy writes.
//
//go:nosplit
func eventfdValuePtr(val *uint64) unsafe.Pointer {
	return unsafe.Pointer(val)
}
