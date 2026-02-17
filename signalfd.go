// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"unsafe"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// SignalFD represents a Linux signalfd file descriptor.
// It provides a file descriptor for accepting signals synchronously,
// enabling signal handling via poll/epoll/io_uring.
//
// SignalFD is created with SFD_NONBLOCK and SFD_CLOEXEC by default.
//
// Invariants:
//   - The caller must block the signals with sigprocmask before using signalfd.
//   - Each Read returns exactly one SignalInfo structure (128 bytes).
type SignalFD struct {
	_    noCopy
	fd   FD
	mask SigSet
}

// SigSet represents a signal set for signalfd operations.
// On Linux amd64, this is a 64-bit mask where bit N represents signal N+1.
//
// Limitation: This implementation supports signals 1-64 only.
// Real-time signals beyond SIGRTMIN+32 (signal 64) are not supported.
// For most applications, standard signals (1-31) are sufficient.
type SigSet uint64

// Signal constants matching Linux signal numbers.
const (
	SIGHUP    = 1
	SIGINT    = 2
	SIGQUIT   = 3
	SIGILL    = 4
	SIGTRAP   = 5
	SIGABRT   = 6
	SIGBUS    = 7
	SIGFPE    = 8
	SIGKILL   = 9
	SIGUSR1   = 10
	SIGSEGV   = 11
	SIGUSR2   = 12
	SIGPIPE   = 13
	SIGALRM   = 14
	SIGTERM   = 15
	SIGSTKFLT = 16
	SIGCHLD   = 17
	SIGCONT   = 18
	SIGSTOP   = 19
	SIGTSTP   = 20
	SIGTTIN   = 21
	SIGTTOU   = 22
	SIGURG    = 23
	SIGXCPU   = 24
	SIGXFSZ   = 25
	SIGVTALRM = 26
	SIGPROF   = 27
	SIGWINCH  = 28
	SIGIO     = 29
	SIGPWR    = 30
	SIGSYS    = 31
)

// Add adds a signal to the set.
// If sig is outside the valid range (1-64), the call is a no-op.
func (s *SigSet) Add(sig int) {
	if sig < 1 || sig > 64 {
		return
	}
	*s |= 1 << (sig - 1)
}

// Del removes a signal from the set.
// If sig is outside the valid range (1-64), the call is a no-op.
func (s *SigSet) Del(sig int) {
	if sig < 1 || sig > 64 {
		return
	}
	*s &^= 1 << (sig - 1)
}

// Has reports whether the signal is in the set.
// Returns false if sig is outside the valid range (1-64).
func (s SigSet) Has(sig int) bool {
	if sig < 1 || sig > 64 {
		return false
	}
	return s&(1<<(sig-1)) != 0
}

// Empty reports whether the set contains no signals.
func (s SigSet) Empty() bool {
	return s == 0
}

// SignalInfo contains information about a received signal.
// This structure matches struct signalfd_siginfo from the Linux kernel.
type SignalInfo struct {
	Signo    uint32   // Signal number
	Errno    int32    // Error number (unused)
	Code     int32    // Signal code
	PID      uint32   // PID of sender
	UID      uint32   // UID of sender
	FD       int32    // File descriptor (SIGIO)
	TID      uint32   // Kernel timer ID (POSIX timers)
	Band     uint32   // Band event (SIGIO)
	Overrun  uint32   // Overrun count (POSIX timers)
	Trapno   uint32   // Trap number
	Status   int32    // Exit status or signal (SIGCHLD)
	Int      int32    // Integer sent by sigqueue
	Ptr      uint64   // Pointer sent by sigqueue
	Utime    uint64   // User CPU time (SIGCHLD)
	Stime    uint64   // System CPU time (SIGCHLD)
	Addr     uint64   // Fault address (SIGILL, SIGFPE, SIGSEGV, SIGBUS)
	AddrLsb  uint16   // LSB of address (SIGBUS)
	_        uint16   // Padding
	Syscall  int32    // Syscall number (SIGSYS)
	CallAddr uint64   // Syscall instruction address (SIGSYS)
	Arch     uint32   // Architecture (SIGSYS)
	_        [28]byte // Padding to 128 bytes
}

// signalInfoSize is the size of SignalInfo in bytes.
const signalInfoSize = 128

// Compile-time size check for SignalInfo.
// Must match struct signalfd_siginfo from the Linux kernel.
var _ [signalInfoSize]byte = [unsafe.Sizeof(SignalInfo{})]byte{}

// NewSignalFD creates a new signalfd monitoring the given signal set.
// The signalfd is created with SFD_NONBLOCK | SFD_CLOEXEC flags.
//
// The caller should block the signals in the set using sigprocmask
// before creating the signalfd to prevent default signal handling.
func NewSignalFD(mask SigSet) (*SignalFD, error) {
	return newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
}

func newSignalFD(mask SigSet, flags uintptr) (*SignalFD, error) {
	// signalfd4 expects the sigset_t size, which is 8 bytes on amd64
	fd, errno := zcall.Signalfd4(
		^uintptr(0), // -1: create new fd
		unsafe.Pointer(&mask),
		unsafe.Sizeof(mask),
		flags,
	)
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &SignalFD{fd: FD(fd), mask: mask}, nil
}

// Fd returns the underlying file descriptor.
// Implements PollFd interface.
//
//go:nosplit
func (s *SignalFD) Fd() int {
	return s.fd.Fd()
}

// Close closes the signalfd.
// Implements PollCloser interface.
func (s *SignalFD) Close() error {
	return s.fd.Close()
}

// Valid reports whether the signalfd is still valid.
//
//go:nosplit
func (s *SignalFD) Valid() bool {
	return s.fd.Valid()
}

// Raw returns the raw file descriptor for use in tight loops.
// The caller must ensure the SignalFD remains valid while using the raw fd.
//
//go:nosplit
func (s *SignalFD) Raw() int32 {
	return s.fd.Raw()
}

// Read reads signal information into the provided buffer.
// Implements io.Reader interface.
// p must be at least 128 bytes.
// Returns iox.ErrWouldBlock if no signal is pending.
func (s *SignalFD) Read(p []byte) (int, error) {
	if len(p) < signalInfoSize {
		return 0, ErrInvalidParam
	}
	raw := s.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Read(uintptr(raw), p[:signalInfoSize])
	if errno != 0 {
		if errno == uintptr(zcall.EAGAIN) {
			return 0, iox.ErrWouldBlock
		}
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// ReadInto reads signal information into a caller-provided SignalInfo.
// Returns iox.ErrWouldBlock if no signal is pending.
// This is the zero-allocation variant for use in hot paths.
//
// Postcondition: On success, info contains the next pending signal.
func (s *SignalFD) ReadInto(info *SignalInfo) error {
	raw := s.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(info)), signalInfoSize)
	_, errno := zcall.Read(uintptr(raw), buf)
	if errno != 0 {
		if errno == uintptr(zcall.EAGAIN) {
			return iox.ErrWouldBlock
		}
		return errFromErrno(errno)
	}
	return nil
}

// SetMask updates the signal mask monitored by this signalfd.
func (s *SignalFD) SetMask(mask SigSet) error {
	raw := s.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	_, errno := zcall.Signalfd4(
		uintptr(raw),
		unsafe.Pointer(&mask),
		unsafe.Sizeof(mask),
		0, // flags are ignored when updating
	)
	if errno != 0 {
		return errFromErrno(errno)
	}
	s.mask = mask
	return nil
}

// Mask returns the current signal mask.
func (s *SignalFD) Mask() SigSet {
	return s.mask
}

// signalfd flags
const (
	SFD_CLOEXEC  = 0x80000
	SFD_NONBLOCK = 0x800
)

// Compile-time interface assertions
var (
	_ PollFd     = (*SignalFD)(nil)
	_ PollCloser = (*SignalFD)(nil)
	_ Reader     = (*SignalFD)(nil)
)
