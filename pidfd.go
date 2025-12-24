// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"code.hybscloud.com/zcall"
)

// PidFD represents a Linux pidfd file descriptor.
// It provides a stable handle to a process that avoids PID reuse races.
//
// A pidfd becomes readable when the process terminates, making it
// suitable for polling via epoll and io_uring.
//
// Invariants:
//   - The pidfd refers to a specific process instance, not just a PID.
//   - After the process exits, the pidfd remains valid for signal/wait operations.
type PidFD struct {
	fd  FD
	pid int
}

// NewPidFD creates a new pidfd for the specified process ID.
// The pidfd is created with PIDFD_NONBLOCK flag.
//
// Returns an error if the process does not exist or if pidfd is not supported.
func NewPidFD(pid int) (*PidFD, error) {
	return newPidFD(pid, PIDFD_NONBLOCK)
}

// NewPidFDBlocking creates a new pidfd for the specified process ID
// without the PIDFD_NONBLOCK flag.
func NewPidFDBlocking(pid int) (*PidFD, error) {
	return newPidFD(pid, 0)
}

func newPidFD(pid int, flags uintptr) (*PidFD, error) {
	if pid <= 0 {
		return nil, ErrInvalidParam
	}
	fd, errno := zcall.PidfdOpen(uintptr(pid), flags)
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &PidFD{fd: FD(fd), pid: pid}, nil
}

// Fd returns the underlying file descriptor.
// Implements PollFd interface.
func (p *PidFD) Fd() int {
	return p.fd.Fd()
}

// Close closes the pidfd.
// Implements PollCloser interface.
func (p *PidFD) Close() error {
	return p.fd.Close()
}

// PID returns the process ID this pidfd refers to.
// Note: The PID value may be reused by a new process after the original exits,
// but the pidfd still refers to the original process instance.
func (p *PidFD) PID() int {
	return p.pid
}

// SendSignal sends a signal to the process.
// This is race-free with respect to PID reuse.
//
// sig is the signal number to send (e.g., SIGTERM, SIGKILL).
// Returns nil on success.
func (p *PidFD) SendSignal(sig int) error {
	raw := p.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	errno := zcall.PidfdSendSignal(uintptr(raw), uintptr(sig), nil, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// GetFD duplicates a file descriptor from the target process.
// targetFD is the file descriptor number in the target process.
//
// This operation requires appropriate privileges (CAP_SYS_PTRACE or
// being in the same user namespace with PTRACE_MODE_ATTACH_REALCREDS).
//
// Returns a new FD in the current process that refers to the same
// open file description as targetFD in the target process.
func (p *PidFD) GetFD(targetFD int) (FD, error) {
	raw := p.fd.Raw()
	if raw < 0 {
		return InvalidFD, ErrClosed
	}
	newfd, errno := zcall.PidfdGetfd(uintptr(raw), uintptr(targetFD), 0)
	if errno != 0 {
		return InvalidFD, errFromErrno(errno)
	}
	return FD(newfd), nil
}

// Valid reports whether the pidfd is still valid.
func (p *PidFD) Valid() bool {
	return p.fd.Valid()
}

// pidfd flags
const (
	PIDFD_NONBLOCK = 0x800
)

// Compile-time interface assertions
var (
	_ PollFd     = (*PidFD)(nil)
	_ PollCloser = (*PidFD)(nil)
)
