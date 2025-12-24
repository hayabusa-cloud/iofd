// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package iofd provides minimal file descriptor abstractions and specialized
// Linux handles for the Go ecosystem. It serves as the common denominator for
// kernel resource lifecycle management.
//
// All kernel interactions use code.hybscloud.com/zcall exclusively,
// bypassing Go's standard library syscall hooks for zero-overhead operation.
package iofd

// PollFd represents a pollable file descriptor.
// Any resource that can be monitored for I/O readiness implements this interface.
type PollFd interface {
	// Fd returns the underlying file descriptor as an integer.
	// The returned value is valid only while the resource is open.
	Fd() int
}

// PollCloser extends PollFd with the ability to close the resource.
type PollCloser interface {
	PollFd
	// Close releases the underlying file descriptor.
	// After Close returns, Fd() behavior is undefined.
	Close() error
}

// Reader is an interface for reading from a file descriptor.
type Reader interface {
	// Read reads up to len(p) bytes into p.
	// Returns the number of bytes read and any error encountered.
	// Returns iox.ErrWouldBlock if the resource is not ready.
	Read(p []byte) (n int, err error)
}

// Writer is an interface for writing to a file descriptor.
type Writer interface {
	// Write writes len(p) bytes from p.
	// Returns the number of bytes written and any error encountered.
	// Returns iox.ErrWouldBlock if the resource is not ready.
	Write(p []byte) (n int, err error)
}

// ReadWriter combines Reader and Writer interfaces.
type ReadWriter interface {
	Reader
	Writer
}

// Handle represents a generic kernel handle with full I/O capabilities.
type Handle interface {
	PollCloser
	ReadWriter
}

// Signaler is an interface for signaling mechanisms like eventfd.
type Signaler interface {
	PollCloser
	// Signal increments the eventfd counter by the given value.
	// Returns iox.ErrWouldBlock if the counter would overflow.
	Signal(val uint64) error
	// Wait reads and resets the eventfd counter.
	// Returns iox.ErrWouldBlock if the counter is zero.
	Wait() (uint64, error)
}

// Timer is an interface for timer handles like timerfd.
type Timer interface {
	PollCloser
	// Arm sets the timer to expire after the given duration.
	// If interval is non-zero, the timer repeats with that interval.
	Arm(initial, interval int64) error
	// Disarm stops the timer.
	Disarm() error
	// Read reads the number of expirations since the last read.
	// Returns iox.ErrWouldBlock if no expirations have occurred.
	Read() (uint64, error)
}
