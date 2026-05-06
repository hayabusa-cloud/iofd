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

// noCopy may be added to structs which must not be copied after first use.
// This is detected by go vet.
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// FD represents one close-capable file descriptor cell.
// It stores the raw descriptor number as an int32 and uses atomic operations
// for Raw, Valid, and same-cell Close access.
//
// An FD value is small and copyable as a Go value, but copying an open FD does
// not duplicate kernel ownership. A copied FD contains the same descriptor
// number in a different Go cell; closing both cells can close an unrelated
// descriptor if the number is reused. Use Dup to create an independent
// close-capable descriptor.
//
// FD does not serialize Close against in-flight operations or borrowed raw
// descriptor users. Callers must stop and drain descriptor users before closing.
//
// Invariants:
//   - A valid FD holds a non-negative value.
//   - After Close(), the FD value becomes -1.
//   - FD access is atomic through one addressable cell.
//   - Close is idempotent for that same cell.
type FD int32

// InvalidFD represents an invalid file descriptor.
const InvalidFD FD = -1

// NewFD creates an FD from a raw file descriptor value.
// The caller is responsible for ensuring fd is valid and for transferring
// close ownership to the returned FD. If the raw descriptor is only borrowed,
// do not call Close on the returned value.
func NewFD(fd int) FD {
	return FD(fd)
}

// Raw returns the underlying file descriptor number as an int32.
// Returns -1 if the FD is invalid or closed.
//
// Raw is a borrowed observation only. It does not transfer ownership and the
// returned descriptor number must not be closed independently.
// Callers must keep the FD open while using the returned number.
//
//go:nosplit
func (fd *FD) Raw() int32 {
	return atomic.LoadInt32((*int32)(fd))
}

// Fd returns the file descriptor number as an int for interface compatibility.
// Implements PollFd interface.
//
// The returned descriptor number is borrowed and is valid only while the FD
// remains open.
//
//go:nosplit
func (fd *FD) Fd() int {
	return int(fd.Raw())
}

// Valid reports whether the file descriptor is valid (non-negative).
//
//go:nosplit
func (fd *FD) Valid() bool {
	return fd.Raw() >= 0
}

// Close closes the file descriptor owned by this FD cell.
// It is safe to call Close multiple times on the same FD cell; subsequent
// calls are no-ops.
//
// Close idempotence does not extend across copied FD values. Copying an open
// FD copies the descriptor number, not ownership. Use Dup to create a second
// descriptor that may be closed independently.
//
// Call Close only after all in-flight operations and borrowed raw descriptor
// users are drained.
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
//
// On EOF (read returns 0 bytes with no error), this returns (0, nil) rather than
// (0, io.EOF). This is intentional for low-level I/O:
//   - For SOCK_STREAM: (0, nil) indicates peer closed the connection (EOF)
//   - For SOCK_DGRAM/SOCK_SEQPACKET: (0, nil) indicates an empty message was
//     received, which is NOT EOF - the peer may send more messages
//
// Higher-level stream abstractions should translate (0, nil) to io.EOF if needed.
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

// Dup duplicates the file descriptor and returns a new close-capable FD.
// The returned FD refers to the same open file description through a distinct
// descriptor-table entry and has FD_CLOEXEC set by default.
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

// Compile-time interface assertions
var (
	_ PollFd     = (*FD)(nil)
	_ PollCloser = (*FD)(nil)
	_ Handle     = (*FD)(nil)
)
