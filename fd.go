// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build unix

package iofd

import (
	"sync/atomic"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// FD represents a file descriptor as a universal handle.
// It wraps an int32 and provides atomic operations for safe concurrent access.
//
// Invariants:
//   - A valid FD holds a non-negative value.
//   - After Close(), the FD value becomes -1.
//   - FD is safe for concurrent use; Close() is idempotent.
type FD int32

// InvalidFD represents an invalid file descriptor.
const InvalidFD FD = -1

// NewFD creates a new FD from a raw file descriptor value.
// The caller is responsible for ensuring fd is valid.
func NewFD(fd int) FD {
	return FD(fd)
}

// Raw returns the underlying file descriptor as an int32.
// Returns -1 if the FD is invalid or closed.
func (fd *FD) Raw() int32 {
	return atomic.LoadInt32((*int32)(fd))
}

// Fd returns the file descriptor as an int for interface compatibility.
// Implements PollFd interface.
func (fd *FD) Fd() int {
	return int(fd.Raw())
}

// Valid reports whether the file descriptor is valid (non-negative).
func (fd *FD) Valid() bool {
	return fd.Raw() >= 0
}

// Close closes the file descriptor.
// It is safe to call Close multiple times; subsequent calls are no-ops.
// Returns nil if already closed.
//
// Postcondition: fd.Raw() == -1
func (fd *FD) Close() error {
	// Atomically swap to -1 to prevent double-close
	old := atomic.SwapInt32((*int32)(fd), -1)
	if old < 0 {
		return nil // Already closed
	}
	errno := zcall.Close(uintptr(old))
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// Read reads up to len(p) bytes from the file descriptor.
// Returns iox.ErrWouldBlock if the fd is non-blocking and no data is available.
func (fd *FD) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	raw := fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Read(uintptr(raw), p)
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// Write writes len(p) bytes to the file descriptor.
// Returns iox.ErrWouldBlock if the fd is non-blocking and cannot accept data.
func (fd *FD) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	raw := fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Write(uintptr(raw), p)
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// SetNonblock sets or clears the O_NONBLOCK flag on the file descriptor.
func (fd *FD) SetNonblock(nonblock bool) error {
	raw := fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	// Get current flags
	flags, errno := zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_GETFL, 0, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	// Modify flags
	if nonblock {
		flags |= O_NONBLOCK
	} else {
		flags &^= O_NONBLOCK
	}
	// Set new flags
	_, errno = zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_SETFL, flags, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// SetCloexec sets or clears the FD_CLOEXEC flag on the file descriptor.
func (fd *FD) SetCloexec(cloexec bool) error {
	raw := fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	// Get current flags
	flags, errno := zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_GETFD, 0, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	// Modify flags
	if cloexec {
		flags |= FD_CLOEXEC
	} else {
		flags &^= FD_CLOEXEC
	}
	// Set new flags
	_, errno = zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_SETFD, flags, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// Dup duplicates the file descriptor.
// The new FD has FD_CLOEXEC set by default.
func (fd *FD) Dup() (FD, error) {
	raw := fd.Raw()
	if raw < 0 {
		return InvalidFD, ErrClosed
	}
	// Use fcntl F_DUPFD_CLOEXEC for atomic dup with CLOEXEC.
	// This is portable across all architectures and platforms.
	newfd, errno := zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_DUPFD_CLOEXEC, 0, 0)
	if errno != 0 {
		return InvalidFD, errFromErrno(errno)
	}
	return FD(newfd), nil
}

// errFromErrno converts a zcall errno to a semantic error.
func errFromErrno(errno uintptr) error {
	if errno == 0 {
		return nil
	}
	e := zcall.Errno(errno)
	switch e {
	case zcall.EAGAIN:
		return iox.ErrWouldBlock
	case zcall.EBADF:
		return ErrClosed
	case zcall.EINVAL:
		return ErrInvalidParam
	case zcall.EINTR:
		return ErrInterrupted
	case zcall.ENOMEM:
		return ErrNoMemory
	case zcall.EACCES, zcall.EPERM:
		return ErrPermission
	default:
		return e
	}
}
