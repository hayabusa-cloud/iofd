// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"encoding/binary"
	"time"
	"unsafe"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// TimerFD represents a Linux timerfd file descriptor.
// It provides a high-resolution timer that can be monitored via poll/epoll/io_uring.
//
// TimerFD is created with TFD_NONBLOCK and TFD_CLOEXEC by default.
type TimerFD struct {
	fd FD
}

// NewTimerFD creates a new timerfd using CLOCK_MONOTONIC.
// The timer is initially disarmed.
func NewTimerFD() (*TimerFD, error) {
	return newTimerFD(CLOCK_MONOTONIC, TFD_NONBLOCK|TFD_CLOEXEC)
}

// NewTimerFDRealtime creates a new timerfd using CLOCK_REALTIME.
// Use this for wall-clock time that adjusts with system time changes.
func NewTimerFDRealtime() (*TimerFD, error) {
	return newTimerFD(CLOCK_REALTIME, TFD_NONBLOCK|TFD_CLOEXEC)
}

// NewTimerFDBoottime creates a new timerfd using CLOCK_BOOTTIME.
// This clock includes time spent in suspend.
func NewTimerFDBoottime() (*TimerFD, error) {
	return newTimerFD(CLOCK_BOOTTIME, TFD_NONBLOCK|TFD_CLOEXEC)
}

func newTimerFD(clockid, flags uintptr) (*TimerFD, error) {
	fd, errno := zcall.TimerfdCreate(clockid, flags)
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &TimerFD{fd: FD(fd)}, nil
}

// Fd returns the underlying file descriptor.
// Implements PollFd interface.
func (t *TimerFD) Fd() int {
	return t.fd.Fd()
}

// Close closes the timerfd.
// Implements PollCloser interface.
func (t *TimerFD) Close() error {
	return t.fd.Close()
}

// Arm sets the timer to expire after initial nanoseconds.
// If interval is non-zero, the timer repeats with that interval (in nanoseconds).
//
// Parameters:
//   - initial: time until first expiration in nanoseconds (0 disarms)
//   - interval: interval for periodic timer in nanoseconds (0 for one-shot)
func (t *TimerFD) Arm(initial, interval int64) error {
	raw := t.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	newValue := itimerspec{
		interval: timespec{
			sec:  interval / 1e9,
			nsec: interval % 1e9,
		},
		value: timespec{
			sec:  initial / 1e9,
			nsec: initial % 1e9,
		},
	}
	errno := zcall.TimerfdSettime(
		uintptr(raw),
		0, // relative time
		unsafe.Pointer(&newValue),
		nil, // don't need old value
	)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// ArmAt sets the timer to expire at an absolute time.
// If interval is non-zero, the timer repeats with that interval (in nanoseconds).
//
// Parameters:
//   - deadline: absolute time for first expiration (Unix nanoseconds)
//   - interval: interval for periodic timer in nanoseconds (0 for one-shot)
func (t *TimerFD) ArmAt(deadline, interval int64) error {
	raw := t.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	newValue := itimerspec{
		interval: timespec{
			sec:  interval / 1e9,
			nsec: interval % 1e9,
		},
		value: timespec{
			sec:  deadline / 1e9,
			nsec: deadline % 1e9,
		},
	}
	errno := zcall.TimerfdSettime(
		uintptr(raw),
		TFD_TIMER_ABSTIME,
		unsafe.Pointer(&newValue),
		nil,
	)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// ArmDuration is a convenience method that arms the timer using time.Duration.
func (t *TimerFD) ArmDuration(initial, interval time.Duration) error {
	return t.Arm(initial.Nanoseconds(), interval.Nanoseconds())
}

// Disarm stops the timer.
func (t *TimerFD) Disarm() error {
	return t.Arm(0, 0)
}

// Read reads the number of expirations since the last read.
// Returns iox.ErrWouldBlock if no expirations have occurred (non-blocking mode).
//
// The returned value is the number of times the timer has expired since
// the last successful read. For periodic timers, this may be > 1 if
// multiple intervals elapsed before reading.
func (t *TimerFD) Read() (uint64, error) {
	raw := t.fd.Raw()
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

// ReadInto reads expiration count into the provided buffer.
// buf must be at least 8 bytes.
func (t *TimerFD) ReadInto(buf []byte) (int, error) {
	if len(buf) < 8 {
		return 0, ErrInvalidParam
	}
	raw := t.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	n, errno := zcall.Read(uintptr(raw), buf[:8])
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// GetTime returns the current timer setting.
// Returns (remaining time until expiration, interval) in nanoseconds.
func (t *TimerFD) GetTime() (remaining, interval int64, err error) {
	raw := t.fd.Raw()
	if raw < 0 {
		return 0, 0, ErrClosed
	}
	var curr itimerspec
	errno := zcall.TimerfdGettime(uintptr(raw), unsafe.Pointer(&curr))
	if errno != 0 {
		return 0, 0, errFromErrno(errno)
	}
	remaining = curr.value.sec*1e9 + curr.value.nsec
	interval = curr.interval.sec*1e9 + curr.interval.nsec
	return remaining, interval, nil
}

// timespec matches struct timespec in Linux.
type timespec struct {
	sec  int64
	nsec int64
}

// itimerspec matches struct itimerspec in Linux.
type itimerspec struct {
	interval timespec
	value    timespec
}

// Clock IDs
const (
	CLOCK_REALTIME  = 0
	CLOCK_MONOTONIC = 1
	CLOCK_BOOTTIME  = 7
)

// timerfd flags
const (
	TFD_CLOEXEC       = 0x80000
	TFD_NONBLOCK      = 0x800
	TFD_TIMER_ABSTIME = 0x1
)

// Compile-time interface assertions
var (
	_ PollFd     = (*TimerFD)(nil)
	_ PollCloser = (*TimerFD)(nil)
	_ Timer      = (*TimerFD)(nil)
)
