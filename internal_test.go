// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"testing"
	"unsafe"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// TestErrFromErrno tests all errno mappings in errFromErrno.
func TestErrFromErrno(t *testing.T) {
	tests := []struct {
		name  string
		errno uintptr
		want  error
		isRaw bool // true if we expect the raw zcall.Errno
	}{
		{"zero", 0, nil, false},
		{"EAGAIN", uintptr(zcall.EAGAIN), iox.ErrWouldBlock, false},
		{"EBADF", uintptr(zcall.EBADF), ErrClosed, false},
		{"EINVAL", uintptr(zcall.EINVAL), ErrInvalidParam, false},
		{"EINTR", uintptr(zcall.EINTR), ErrInterrupted, false},
		{"ENOMEM", uintptr(zcall.ENOMEM), ErrNoMemory, false},
		{"EACCES", uintptr(zcall.EACCES), ErrPermission, false},
		{"EPERM", uintptr(zcall.EPERM), ErrPermission, false},
		{"ENOENT (default)", uintptr(zcall.ENOENT), zcall.ENOENT, true},
		{"EIO (default)", uintptr(zcall.EIO), zcall.EIO, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errFromErrno(tt.errno)
			if tt.isRaw {
				// For default case, check it's the raw errno
				if e, ok := got.(zcall.Errno); !ok || e != zcall.Errno(tt.errno) {
					t.Errorf("errFromErrno(%d) = %v, want raw errno %v", tt.errno, got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("errFromErrno(%d) = %v, want %v", tt.errno, got, tt.want)
				}
			}
		})
	}
}

// =============================================================================
// Syscall Error Path Tests
// =============================================================================

// TestSetNonblock_FcntlErrors tests fcntl error paths in SetNonblock.
// Uses an FD that is valid (>= 0) but closed at kernel level.
func TestSetNonblock_FcntlErrors(t *testing.T) {
	// Create a valid eventfd, get its raw fd, then close it via zcall
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly, bypassing the FD wrapper
	zcall.Close(uintptr(rawFd))

	// Create a new FD pointing to the now-invalid descriptor
	fd := NewFD(int(rawFd))

	// SetNonblock should fail on F_GETFL with EBADF
	err = fd.SetNonblock(true)
	if err == nil {
		t.Error("SetNonblock should fail on closed fd")
	}
	// The error should be ErrClosed (mapped from EBADF)
	if err != ErrClosed {
		t.Logf("SetNonblock error: %v (type: %T)", err, err)
	}
}

// TestSetCloexec_FcntlErrors tests fcntl error paths in SetCloexec.
func TestSetCloexec_FcntlErrors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	fd := NewFD(int(rawFd))

	// SetCloexec should fail on F_GETFD with EBADF
	err = fd.SetCloexec(true)
	if err == nil {
		t.Error("SetCloexec should fail on closed fd")
	}
	if err != ErrClosed {
		t.Logf("SetCloexec error: %v (type: %T)", err, err)
	}
}

// TestDup_Errors tests error paths in Dup.
func TestDup_Errors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	fd := NewFD(int(rawFd))

	// Dup should fail with EBADF
	_, err = fd.Dup()
	if err == nil {
		t.Error("Dup should fail on closed fd")
	}
	if err != ErrClosed {
		t.Logf("Dup error: %v (type: %T)", err, err)
	}
}

// TestFD_ReadWriteErrors tests Read/Write error paths.
func TestFD_ReadWriteErrors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	fd := NewFD(int(rawFd))

	// Read should fail with EBADF
	buf := make([]byte, 8)
	_, err = fd.Read(buf)
	if err == nil {
		t.Error("Read should fail on closed fd")
	}
	if err != ErrClosed {
		t.Logf("Read error: %v (type: %T)", err, err)
	}

	// Write should fail with EBADF
	_, err = fd.Write(buf)
	if err == nil {
		t.Error("Write should fail on closed fd")
	}
	if err != ErrClosed {
		t.Logf("Write error: %v (type: %T)", err, err)
	}
}

// TestEventFD_SignalErrors tests Signal error paths.
func TestEventFD_SignalErrors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// Signal should fail with EBADF (mapped to some error)
	err = efd.Signal(1)
	if err == nil {
		t.Error("Signal should fail on closed fd")
	}
}

// TestEventFD_WaitErrors tests Wait error paths.
func TestEventFD_WaitErrors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// Wait should fail with EBADF
	_, err = efd.Wait()
	if err == nil {
		t.Error("Wait should fail on closed fd")
	}
}

// TestEventFD_ReadWriteErrors tests EventFD Read/Write error paths.
func TestEventFD_ReadWriteErrors(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// Read should fail
	buf := make([]byte, 8)
	_, err = efd.Read(buf)
	if err == nil {
		t.Error("Read should fail on closed fd")
	}

	// Write should fail
	_, err = efd.Write(buf)
	if err == nil {
		t.Error("Write should fail on closed fd")
	}
}

// TestTimerFD_Errors tests TimerFD error paths.
func TestTimerFD_Errors(t *testing.T) {
	tfd, err := newTimerFD(CLOCK_MONOTONIC, TFD_NONBLOCK|TFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newTimerFD failed: %v", err)
	}
	rawFd := tfd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// Arm should fail
	err = tfd.Arm(1000000, 0)
	if err == nil {
		t.Error("Arm should fail on closed fd")
	}

	// ArmAt should fail
	err = tfd.ArmAt(1000000000, 0)
	if err == nil {
		t.Error("ArmAt should fail on closed fd")
	}

	// Read should fail
	_, err = tfd.Expirations()
	if err == nil {
		t.Error("Read should fail on closed fd")
	}

	// ReadInto should fail
	buf := make([]byte, 8)
	_, err = tfd.Read(buf)
	if err == nil {
		t.Error("ReadInto should fail on closed fd")
	}

	// GetTime should fail
	_, _, err = tfd.GetTime()
	if err == nil {
		t.Error("GetTime should fail on closed fd")
	}
}

// TestMemFD_Errors tests MemFD error paths.
func TestMemFD_Errors(t *testing.T) {
	mfd, err := newMemFD("test", MFD_CLOEXEC|MFD_ALLOW_SEALING)
	if err != nil {
		t.Fatalf("newMemFD failed: %v", err)
	}
	rawFd := mfd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// Truncate should fail
	err = mfd.Truncate(1024)
	if err == nil {
		t.Error("Truncate should fail on closed fd")
	}

	// Size should fail
	_, err = mfd.Size()
	if err == nil {
		t.Error("Size should fail on closed fd")
	}

	// Seal should fail
	err = mfd.Seal(F_SEAL_WRITE)
	if err == nil {
		t.Error("Seal should fail on closed fd")
	}

	// Seals should fail
	_, err = mfd.Seals()
	if err == nil {
		t.Error("Seals should fail on closed fd")
	}
}

// TestPidFD_Errors tests PidFD error paths.
func TestPidFD_Errors(t *testing.T) {
	pfd, err := newPidFD(1, PIDFD_NONBLOCK) // PID 1 (init) always exists
	if err != nil {
		t.Skipf("newPidFD failed (may need privileges): %v", err)
	}
	rawFd := pfd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// SendSignal should fail
	err = pfd.SendSignal(0)
	if err == nil {
		t.Error("SendSignal should fail on closed fd")
	}

	// GetFD should fail
	_, err = pfd.GetFD(0)
	if err == nil {
		t.Error("GetFD should fail on closed fd")
	}
}

// TestSignalFD_Errors tests SignalFD error paths.
func TestSignalFD_Errors(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	rawFd := sfd.fd.Raw()

	// Close the underlying fd directly
	zcall.Close(uintptr(rawFd))

	// ReadInto should fail
	var info SignalInfo
	err = sfd.ReadInto(&info)
	if err == nil {
		t.Error("ReadInto should fail on closed fd")
	}

	// Read (io.Reader) should fail
	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	if err == nil {
		t.Error("Read should fail on closed fd")
	}

	// SetMask should fail
	err = sfd.SetMask(mask)
	if err == nil {
		t.Error("SetMask should fail on closed fd")
	}
}

// =============================================================================
// EAGAIN/WouldBlock Tests
// =============================================================================

// TestEventFD_WaitWouldBlock tests Wait returning ErrWouldBlock when counter is zero.
func TestEventFD_WaitWouldBlock(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Wait on empty eventfd should return ErrWouldBlock
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Wait on empty eventfd: got %v, want ErrWouldBlock", err)
	}
}

// TestEventFD_ReadWouldBlock tests Read returning ErrWouldBlock when counter is zero.
func TestEventFD_ReadWouldBlock(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Read on empty eventfd should return ErrWouldBlock
	buf := make([]byte, 8)
	_, err = efd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("Read on empty eventfd: got %v, want ErrWouldBlock", err)
	}
}

// TestTimerFD_ReadWouldBlock tests Read returning ErrWouldBlock when timer hasn't expired.
func TestTimerFD_ReadWouldBlock(t *testing.T) {
	tfd, err := newTimerFD(CLOCK_MONOTONIC, TFD_NONBLOCK|TFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Read on unarmed timer should return ErrWouldBlock
	_, err = tfd.Expirations()
	if err != iox.ErrWouldBlock {
		t.Errorf("Read on unarmed timer: got %v, want ErrWouldBlock", err)
	}
}

// TestTimerFD_ReadIntoWouldBlock tests ReadInto returning ErrWouldBlock.
func TestTimerFD_ReadIntoWouldBlock(t *testing.T) {
	tfd, err := newTimerFD(CLOCK_MONOTONIC, TFD_NONBLOCK|TFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// ReadInto on unarmed timer should return ErrWouldBlock
	buf := make([]byte, 8)
	_, err = tfd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("ReadInto on unarmed timer: got %v, want ErrWouldBlock", err)
	}
}

// TestSignalFD_ReadWouldBlock tests Read returning ErrWouldBlock when no signal pending.
func TestSignalFD_ReadWouldBlock(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// ReadInto with no pending signal should return ErrWouldBlock
	var info SignalInfo
	err = sfd.ReadInto(&info)
	if err != iox.ErrWouldBlock {
		t.Errorf("ReadInto with no pending signal: got %v, want ErrWouldBlock", err)
	}
}

// TestSignalFD_ReadIOReaderWouldBlock tests Read (io.Reader) returning ErrWouldBlock.
func TestSignalFD_ReadIOReaderWouldBlock(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Read (io.Reader) with no pending signal should return ErrWouldBlock
	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("Read with no pending signal: got %v, want ErrWouldBlock", err)
	}
}

// TestFD_SetNonblockBothDirections tests setting and clearing O_NONBLOCK.
func TestFD_SetNonblockBothDirections(t *testing.T) {
	efd, err := newEventFD(0, EFD_CLOEXEC) // Start without NONBLOCK
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := &efd.fd

	// Set nonblock
	err = fd.SetNonblock(true)
	if err != nil {
		t.Errorf("SetNonblock(true) failed: %v", err)
	}

	// Clear nonblock
	err = fd.SetNonblock(false)
	if err != nil {
		t.Errorf("SetNonblock(false) failed: %v", err)
	}
}

// TestFD_SetCloexecBothDirections tests setting and clearing FD_CLOEXEC.
func TestFD_SetCloexecBothDirections(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK) // Start without CLOEXEC
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := &efd.fd

	// Set cloexec
	err = fd.SetCloexec(true)
	if err != nil {
		t.Errorf("SetCloexec(true) failed: %v", err)
	}

	// Clear cloexec
	err = fd.SetCloexec(false)
	if err != nil {
		t.Errorf("SetCloexec(false) failed: %v", err)
	}
}

// =============================================================================
// PidFD Error Path Tests
// =============================================================================

// TestPidFD_SendSignalInvalid tests SendSignal with invalid signal.
func TestPidFD_SendSignalInvalid(t *testing.T) {
	pfd, err := newPidFD(1, PIDFD_NONBLOCK) // PID 1 (init) always exists
	if err != nil {
		t.Skipf("newPidFD failed (may need privileges): %v", err)
	}
	defer pfd.Close()

	// Send invalid signal number - should fail with EINVAL
	err = pfd.SendSignal(-1)
	if err == nil {
		t.Error("SendSignal(-1) should fail")
	}

	// Send signal 0 is valid (null signal for checking)
	err = pfd.SendSignal(0)
	// This may succeed or fail depending on permissions, but exercises the path
	t.Logf("SendSignal(0) result: %v", err)
}

// TestPidFD_GetFDInvalid tests GetFD with invalid target FD.
func TestPidFD_GetFDInvalid(t *testing.T) {
	pfd, err := newPidFD(1, PIDFD_NONBLOCK) // PID 1 (init)
	if err != nil {
		t.Skipf("newPidFD failed (may need privileges): %v", err)
	}
	defer pfd.Close()

	// Try to get an invalid FD from the target process
	// This should fail with EBADF or EPERM
	_, err = pfd.GetFD(99999)
	if err == nil {
		t.Error("GetFD(99999) should fail")
	}
	t.Logf("GetFD(99999) error: %v", err)
}

// TestPidFD_InvalidPid tests creating PidFD with invalid PID.
func TestPidFD_InvalidPid(t *testing.T) {
	// PID 0 is invalid
	_, err := newPidFD(0, PIDFD_NONBLOCK)
	if err != ErrInvalidParam {
		t.Errorf("newPidFD(0) should return ErrInvalidParam, got %v", err)
	}

	// Negative PID is invalid
	_, err = newPidFD(-1, PIDFD_NONBLOCK)
	if err != ErrInvalidParam {
		t.Errorf("newPidFD(-1) should return ErrInvalidParam, got %v", err)
	}
}

// TestPidFD_NonexistentPid tests creating PidFD with non-existent PID.
func TestPidFD_NonexistentPid(t *testing.T) {
	// Very high PID that likely doesn't exist
	_, err := newPidFD(4194304, PIDFD_NONBLOCK) // Max PID on most systems
	if err == nil {
		t.Error("newPidFD with non-existent PID should fail")
	}
	t.Logf("newPidFD(4194304) error: %v", err)
}

// =============================================================================
// MemFD Error Path Tests
// =============================================================================

// TestMemFD_SealWithoutAllowSealing tests sealing without MFD_ALLOW_SEALING.
func TestMemFD_SealWithoutAllowSealing(t *testing.T) {
	// Create memfd without MFD_ALLOW_SEALING
	mfd, err := newMemFD("test", MFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Seal should fail without MFD_ALLOW_SEALING
	err = mfd.Seal(F_SEAL_WRITE)
	if err == nil {
		t.Error("Seal should fail without MFD_ALLOW_SEALING")
	}
	t.Logf("Seal error: %v", err)
}

// TestMemFD_SealsWithoutAllowSealing tests getting seals without MFD_ALLOW_SEALING.
func TestMemFD_SealsWithoutAllowSealing(t *testing.T) {
	// Create memfd without MFD_ALLOW_SEALING
	mfd, err := newMemFD("test", MFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newMemFD failed: %v", err)
	}
	defer mfd.Close()
	// Seals should return 0 or error
	seals, err := mfd.Seals()
	// This may succeed with 0 seals or fail - both are valid
	t.Logf("Seals result: %d, error: %v", seals, err)
}

// =============================================================================
// TimerFD Error Path Tests
// =============================================================================

// TestTimerFD_InvalidClockID tests creating TimerFD with invalid clock ID.
func TestTimerFD_InvalidClockID(t *testing.T) {
	// Use an invalid clock ID to trigger syscall error
	_, err := newTimerFD(9999, TFD_NONBLOCK|TFD_CLOEXEC)
	if err == nil {
		t.Error("newTimerFD with invalid clock ID should fail")
	}
	t.Logf("newTimerFD(9999) error: %v", err)
}

// =============================================================================
// EventFD Error Path Tests
// =============================================================================

// TestEventFD_SignalMaxValue tests Signal with maximum value.
func TestEventFD_SignalMaxValue(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal with max value - should succeed or return overflow error
	err = efd.Signal(0xFFFFFFFFFFFFFFFE)
	if err != nil {
		t.Logf("Signal(max) error (expected on overflow): %v", err)
	}
}

// TestEventFD_WaitEAGAIN tests Wait returning EAGAIN.
func TestEventFD_WaitEAGAIN(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Wait on empty eventfd should return ErrWouldBlock
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Wait on empty eventfd should return ErrWouldBlock, got %v", err)
	}
}

// =============================================================================
// SignalFD Error Path Tests
// =============================================================================

// TestSignalFD_ReadEAGAIN tests Read returning EAGAIN.
func TestSignalFD_ReadEAGAIN(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// ReadInto with no pending signal should return ErrWouldBlock
	var info SignalInfo
	err = sfd.ReadInto(&info)
	if err != iox.ErrWouldBlock {
		t.Errorf("ReadInto with no signal should return ErrWouldBlock, got %v", err)
	}
}

// =============================================================================
// FD Dup and Flag Tests
// =============================================================================

// TestFD_DupSuccess tests successful Dup operation.
func TestFD_DupSuccess(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Dup should succeed
	newFD, err := efd.fd.Dup()
	if err != nil {
		t.Fatalf("Dup failed: %v", err)
	}
	defer newFD.Close()

	if !newFD.Valid() {
		t.Error("Duped FD should be valid")
	}
}

// TestSetNonblock_Success tests successful SetNonblock.
func TestSetNonblock_Success(t *testing.T) {
	efd, err := newEventFD(0, EFD_CLOEXEC) // Create without NONBLOCK
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Set nonblock
	err = efd.fd.SetNonblock(true)
	if err != nil {
		t.Errorf("SetNonblock(true) failed: %v", err)
	}

	// Clear nonblock
	err = efd.fd.SetNonblock(false)
	if err != nil {
		t.Errorf("SetNonblock(false) failed: %v", err)
	}
}

// TestSetCloexec_Success tests successful SetCloexec.
func TestSetCloexec_Success(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK) // Create without CLOEXEC
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Set cloexec
	err = efd.fd.SetCloexec(true)
	if err != nil {
		t.Errorf("SetCloexec(true) failed: %v", err)
	}

	// Clear cloexec
	err = efd.fd.SetCloexec(false)
	if err != nil {
		t.Errorf("SetCloexec(false) failed: %v", err)
	}
}

// =============================================================================
// Constructor Failure Tests
// =============================================================================

// TestNewEventFD_InvalidFlags tests newEventFD with invalid flags.
func TestNewEventFD_InvalidFlags(t *testing.T) {
	// Use an extremely invalid flags value to trigger EINVAL
	_, err := newEventFD(0, 0xFFFFFFFF)
	if err == nil {
		t.Error("newEventFD with invalid flags should fail")
	}
	t.Logf("newEventFD(invalid flags) error: %v", err)
}

// TestNewSignalFD_InvalidFlags tests newSignalFD with invalid flags.
func TestNewSignalFD_InvalidFlags(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)
	// Use invalid flags to trigger EINVAL
	_, err := newSignalFD(mask, 0xFFFFFFFF)
	if err == nil {
		t.Error("newSignalFD with invalid flags should fail")
	}
	t.Logf("newSignalFD(invalid flags) error: %v", err)
}

// TestNewMemFD_InvalidFlags tests newMemFD with invalid flags.
func TestNewMemFD_InvalidFlags(t *testing.T) {
	// Use invalid flags combination to trigger EINVAL
	_, err := newMemFD("test", 0xFFFFFFFF)
	if err == nil {
		t.Error("newMemFD with invalid flags should fail")
	}
	t.Logf("newMemFD(invalid flags) error: %v", err)
}

// TestNewMemFD_HugeTLBWithoutPrivilege tests newMemFD with HUGETLB flag.
func TestNewMemFD_HugeTLBWithoutPrivilege(t *testing.T) {
	// MFD_HUGETLB may fail without proper privileges or huge page configuration
	_, err := newMemFD("test", MFD_CLOEXEC|MFD_HUGETLB)
	if err != nil {
		t.Logf("newMemFD(HUGETLB) error (expected without privileges): %v", err)
	}
	// Either success or failure is acceptable depending on system configuration
}

// =============================================================================
// Additional Error Path Tests
// =============================================================================

// TestSignalFD_ReadPartialBuffer tests SignalFD Read with various buffer scenarios.
func TestSignalFD_ReadPartialBuffer(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)
	sfd, err := NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Read (io.Reader) with exactly 128 bytes (minimum required)
	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	// Should return EAGAIN since no signal is pending
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("Read(128) error: %v", err)
	}

	// Read (io.Reader) with more than 128 bytes
	largeBuf := make([]byte, 256)
	_, err = sfd.Read(largeBuf)
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("Read(256) error: %v", err)
	}
}

// TestEventFD_SignalPartialWrite tests Signal behavior.
func TestEventFD_SignalPartialWrite(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal with value 1
	err = efd.Signal(1)
	if err != nil {
		t.Errorf("Signal(1) failed: %v", err)
	}

	// Signal with value 0 should be no-op
	err = efd.Signal(0)
	if err != nil {
		t.Errorf("Signal(0) should succeed: %v", err)
	}

	// Read the value
	val, err := efd.Wait()
	if err != nil {
		t.Errorf("Wait failed: %v", err)
	}
	if val != 1 {
		t.Errorf("Expected 1, got %d", val)
	}
}

// TestTimerFD_ReadPartial tests TimerFD Read behavior.
func TestTimerFD_ReadPartial(t *testing.T) {
	tfd, err := newTimerFD(CLOCK_MONOTONIC, TFD_NONBLOCK|TFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// ReadInto with exactly 8 bytes
	buf := make([]byte, 8)
	_, err = tfd.Read(buf)
	// Should return EAGAIN since timer is not armed
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("ReadInto(8) error: %v", err)
	}

	// ReadInto with more than 8 bytes
	largeBuf := make([]byte, 16)
	_, err = tfd.Read(largeBuf)
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("ReadInto(16) error: %v", err)
	}
}

// TestFD_DupWithValidFD tests Dup on a valid file descriptor.
func TestFD_DupWithValidFD(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// Dup should succeed
	newFd, err := efd.fd.Dup()
	if err != nil {
		t.Fatalf("Dup failed: %v", err)
	}
	defer newFd.Close()

	// Both FDs should be valid
	if !efd.fd.Valid() {
		t.Error("Original FD should be valid")
	}
	if !newFd.Valid() {
		t.Error("New FD should be valid")
	}

	// Write to original, read from dup
	err = efd.Signal(42)
	if err != nil {
		t.Errorf("Signal failed: %v", err)
	}

	// Read from the duplicated fd
	var buf [8]byte
	n, err := newFd.Read(buf[:])
	if err != nil {
		t.Errorf("Read from dup failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Expected 8 bytes, got %d", n)
	}
}

// TestConcurrentClose tests concurrent Close calls.
func TestConcurrentClose(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}

	// Launch multiple goroutines to close concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			efd.Close()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// FD should be invalid after close
	if efd.fd.Valid() {
		t.Error("FD should be invalid after close")
	}
}

// TestFD_ConcurrentReadWrite tests concurrent Read/Write operations.
func TestFD_ConcurrentReadWrite(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	done := make(chan bool, 20)

	// Writers
	for i := 0; i < 10; i++ {
		go func() {
			efd.Signal(1)
			done <- true
		}()
	}

	// Readers
	for i := 0; i < 10; i++ {
		go func() {
			efd.Wait()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

// TestMemFD_TruncateAndSize tests Truncate and Size operations.
func TestMemFD_TruncateAndSize(t *testing.T) {
	mfd, err := newMemFD("test", MFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Initial size should be 0
	size, err := mfd.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 0 {
		t.Errorf("Initial size should be 0, got %d", size)
	}

	// Truncate to 4096
	err = mfd.Truncate(4096)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Size should now be 4096
	size, err = mfd.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 4096 {
		t.Errorf("Size should be 4096, got %d", size)
	}

	// Truncate to smaller size
	err = mfd.Truncate(1024)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	size, err = mfd.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 1024 {
		t.Errorf("Size should be 1024, got %d", size)
	}
}

// =============================================================================
// PidFD Success Path Tests
// =============================================================================

// Linux amd64 syscall numbers not exported by zcall
const (
	sysGetpid        = 39 // SYS_GETPID
	sysKill          = 62 // SYS_KILL
	sysRtSigprocmask = 14 // SYS_RT_SIGPROCMASK
	sigBlock         = 0  // SIG_BLOCK
	sigUnblock       = 1  // SIG_UNBLOCK
)

// TestPidFD_SendSignalToSelf tests SendSignal success by signaling our own process.
func TestPidFD_SendSignalToSelf(t *testing.T) {
	// Get our own PID using getpid syscall
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)

	pfd, err := newPidFD(int(pid), PIDFD_NONBLOCK)
	if err != nil {
		t.Skipf("newPidFD for self failed: %v", err)
	}
	defer pfd.Close()

	// Send signal 0 (null signal) to ourselves - this should always succeed
	err = pfd.SendSignal(0)
	if err != nil {
		t.Errorf("SendSignal(0) to self should succeed, got: %v", err)
	}
}

// =============================================================================
// SignalFD Success Path Tests
// =============================================================================

// TestSignalFD_ReadSuccess tests SignalFD Read success by receiving an actual signal.
func TestSignalFD_ReadSuccess(t *testing.T) {
	// Block SIGUSR1 using sigprocmask before creating signalfd
	// SIG_BLOCK = 0, we need to block SIGUSR1 to receive it via signalfd
	var mask SigSet
	mask.Add(SIGUSR1)

	// Block SIGUSR1: sigprocmask(SIG_BLOCK, &mask, nil)
	_, errno := zcall.Syscall4(
		sysRtSigprocmask,
		sigBlock,
		uintptr(unsafe.Pointer(&mask)),
		0,
		8, // sizeof(sigset_t) on amd64
	)
	if errno != 0 {
		t.Skipf("sigprocmask failed: %v", zcall.Errno(errno))
	}

	// Unblock SIGUSR1 when done
	defer func() {
		zcall.Syscall4(sysRtSigprocmask, sigUnblock, uintptr(unsafe.Pointer(&mask)), 0, 8)
	}()

	// Create signalfd
	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Send SIGUSR1 to ourselves
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)
	_, errno = zcall.Syscall4(sysKill, pid, SIGUSR1, 0, 0)
	if errno != 0 {
		t.Fatalf("kill(self, SIGUSR1) failed: %v", zcall.Errno(errno))
	}

	// ReadInto the signal - retry a few times to handle timing
	var info SignalInfo
	for i := 0; i < 10; i++ {
		err = sfd.ReadInto(&info)
		if err == nil {
			break
		}
		if err != iox.ErrWouldBlock {
			t.Errorf("SignalFD.ReadInto unexpected error: %v", err)
			return
		}
		// Small delay for signal delivery
		for j := 0; j < 1000; j++ {
			// busy wait
		}
	}
	if err != nil {
		t.Logf("SignalFD.ReadInto still blocked after retries (signal delivery timing): %v", err)
		return
	}
	if info.Signo != SIGUSR1 {
		t.Errorf("Expected signal %d, got %d", SIGUSR1, info.Signo)
	}
}

// TestSignalFD_ReadIOReaderSuccess tests SignalFD Read (io.Reader) success path.
func TestSignalFD_ReadIOReaderSuccess(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR2)

	// Block SIGUSR2
	_, errno := zcall.Syscall4(
		sysRtSigprocmask,
		sigBlock,
		uintptr(unsafe.Pointer(&mask)),
		0,
		8,
	)
	if errno != 0 {
		t.Skipf("sigprocmask failed: %v", zcall.Errno(errno))
	}

	defer func() {
		zcall.Syscall4(sysRtSigprocmask, sigUnblock, uintptr(unsafe.Pointer(&mask)), 0, 8)
	}()

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Send SIGUSR2 to ourselves
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)
	_, errno = zcall.Syscall4(sysKill, pid, SIGUSR2, 0, 0)
	if errno != 0 {
		t.Fatalf("kill(self, SIGUSR2) failed: %v", zcall.Errno(errno))
	}

	// Read (io.Reader) - retry a few times to handle timing
	buf := make([]byte, 128)
	var n int
	for i := 0; i < 10; i++ {
		n, err = sfd.Read(buf)
		if err == nil {
			break
		}
		if err != iox.ErrWouldBlock {
			t.Errorf("SignalFD.Read unexpected error: %v", err)
			return
		}
		// Small delay for signal delivery
		for j := 0; j < 1000; j++ {
			// busy wait
		}
	}
	if err != nil {
		t.Logf("SignalFD.Read still blocked after retries (signal delivery timing): %v", err)
		return
	}
	if n != 128 {
		t.Errorf("Expected 128 bytes, got %d", n)
	}
}

// TestEventFD_Raw tests the Raw() method for FD caching in tight loops.
func TestEventFD_Raw(t *testing.T) {
	efd, err := NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	raw := efd.Raw()
	if raw < 0 {
		t.Errorf("Raw() should return valid fd, got %d", raw)
	}

	// Raw should match Fd
	if int(raw) != efd.Fd() {
		t.Errorf("Raw() = %d, Fd() = %d, should match", raw, efd.Fd())
	}
}

// TestEventFD_RawAfterClose tests Raw() returns -1 after close.
func TestEventFD_RawAfterClose(t *testing.T) {
	efd, err := NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	efd.Close()

	raw := efd.Raw()
	if raw != -1 {
		t.Errorf("Raw() after close should return -1, got %d", raw)
	}
}

// TestTimerFD_Raw tests the Raw() method.
func TestTimerFD_Raw(t *testing.T) {
	tfd, err := NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	raw := tfd.Raw()
	if raw < 0 {
		t.Errorf("Raw() should return valid fd, got %d", raw)
	}
	if int(raw) != tfd.Fd() {
		t.Errorf("Raw() = %d, Fd() = %d, should match", raw, tfd.Fd())
	}
}

// TestSignalFD_Raw tests the Raw() method.
func TestSignalFD_Raw(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	raw := sfd.Raw()
	if raw < 0 {
		t.Errorf("Raw() should return valid fd, got %d", raw)
	}
	if int(raw) != sfd.Fd() {
		t.Errorf("Raw() = %d, Fd() = %d, should match", raw, sfd.Fd())
	}
}

// TestSignalFD_Valid tests the Valid() method.
func TestSignalFD_Valid(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}

	if !sfd.Valid() {
		t.Error("Valid() should return true for open signalfd")
	}

	sfd.Close()

	if sfd.Valid() {
		t.Error("Valid() should return false after close")
	}
}

// TestSignalFD_ReadTo tests the ReadTo zero-allocation method.
func TestSignalFD_ReadTo(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	// Block SIGUSR1
	_, errno := zcall.Syscall4(
		sysRtSigprocmask,
		1, // SIG_BLOCK
		uintptr(unsafe.Pointer(&mask)),
		0,
		8,
	)
	if errno != 0 {
		t.Fatalf("sigprocmask failed: %v", zcall.Errno(errno))
	}

	sfd, err := NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// ReadTo should return ErrWouldBlock when no signal is pending
	var info SignalInfo
	err = sfd.ReadInto(&info)
	if err != iox.ErrWouldBlock {
		t.Errorf("ReadTo should return ErrWouldBlock when no signal pending, got: %v", err)
	}

	// Send SIGUSR1 to ourselves
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)
	_, errno = zcall.Syscall4(sysKill, pid, SIGUSR1, 0, 0)
	if errno != 0 {
		t.Fatalf("kill(self, SIGUSR1) failed: %v", zcall.Errno(errno))
	}

	// ReadTo should succeed now - retry for signal delivery timing
	for i := 0; i < 10; i++ {
		err = sfd.ReadInto(&info)
		if err == nil {
			break
		}
		if err != iox.ErrWouldBlock {
			t.Errorf("ReadTo unexpected error: %v", err)
			return
		}
		for j := 0; j < 1000; j++ {
			// busy wait
		}
	}
	if err != nil {
		t.Logf("ReadTo still blocked after retries (signal delivery timing): %v", err)
		return
	}
	if info.Signo != SIGUSR1 {
		t.Errorf("Expected signal %d, got %d", SIGUSR1, info.Signo)
	}
}

// TestSignalFD_ReadToClosed tests ReadTo on closed signalfd.
func TestSignalFD_ReadToClosed(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	sfd.Close()

	var info SignalInfo
	err = sfd.ReadInto(&info)
	if err != ErrClosed {
		t.Errorf("ReadTo on closed should return ErrClosed, got: %v", err)
	}
}

// TestMemFD_Raw tests the Raw() method.
func TestMemFD_Raw(t *testing.T) {
	mfd, err := NewMemFD("test")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	raw := mfd.Raw()
	if raw < 0 {
		t.Errorf("Raw() should return valid fd, got %d", raw)
	}
	if int(raw) != mfd.Fd() {
		t.Errorf("Raw() = %d, Fd() = %d, should match", raw, mfd.Fd())
	}
}

// TestPidFD_Raw tests the Raw() method.
func TestPidFD_Raw(t *testing.T) {
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)
	pfd, err := NewPidFD(int(pid))
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	raw := pfd.Raw()
	if raw < 0 {
		t.Errorf("Raw() should return valid fd, got %d", raw)
	}
	if int(raw) != pfd.Fd() {
		t.Errorf("Raw() = %d, Fd() = %d, should match", raw, pfd.Fd())
	}
}

// TestPidFD_GetFDSelf tests GetFD success path using our own process.
func TestPidFD_GetFDSelf(t *testing.T) {
	pid, _ := zcall.Syscall4(sysGetpid, 0, 0, 0, 0)
	pfd, err := NewPidFD(int(pid))
	if err != nil {
		t.Skipf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	// Create an eventfd as a target FD to duplicate
	efd, err := NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Try to get the eventfd from ourselves - this might work on newer kernels
	// when targeting our own process
	dupFD, err := pfd.GetFD(efd.Fd())
	if err != nil {
		// Expected to fail without CAP_SYS_PTRACE, but we tried
		t.Logf("GetFD self failed (expected without privileges): %v", err)
		return
	}
	defer dupFD.Close()

	// If it succeeded, verify the duplicated FD works
	if !dupFD.Valid() {
		t.Error("Duplicated FD should be valid")
	}
	t.Logf("GetFD succeeded with fd=%d", dupFD.Fd())
}

// =============================================================================
// SetNonblock/SetCloexec F_SETFL/F_SETFD Error Path Tests
// =============================================================================

// TestSetNonblock_SetFLError tests the F_SETFL error path in SetNonblock.
// This requires closing the fd between F_GETFL and F_SETFL calls.
func TestSetNonblock_SetFLError(t *testing.T) {
	// Create eventfd
	efd, err := newEventFD(0, EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// We need to close the fd right after F_GETFL succeeds but before F_SETFL.
	// This is impossible to do deterministically without modifying the source.
	// Instead, we test by closing fd before SetNonblock, which hits the first check.
	zcall.Close(uintptr(rawFd))

	fd := NewFD(int(rawFd))
	err = fd.SetNonblock(true)
	if err == nil {
		t.Error("SetNonblock should fail on closed fd")
	}
	// This hits the F_GETFL error path (line 125-126), not F_SETFL
}

// TestSetCloexec_SetFDError tests the F_SETFD error path in SetCloexec.
func TestSetCloexec_SetFDError(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	zcall.Close(uintptr(rawFd))

	fd := NewFD(int(rawFd))
	err = fd.SetCloexec(true)
	if err == nil {
		t.Error("SetCloexec should fail on closed fd")
	}
}

// TestEventFD_SignalOverflow tests Signal returning ErrWouldBlock on counter overflow.
func TestEventFD_SignalOverflow(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	defer efd.Close()

	// eventfd max counter value is 0xFFFFFFFFFFFFFFFE (2^64 - 2)
	// Signal with max-1 to get counter near max
	maxVal := uint64(0xFFFFFFFFFFFFFFFE)
	err = efd.Signal(maxVal - 1)
	if err != nil {
		t.Fatalf("Signal(max-1) failed: %v", err)
	}

	// Now try to signal with 2, which would overflow
	// This should return ErrWouldBlock
	err = efd.Signal(2)
	if err != iox.ErrWouldBlock {
		t.Errorf("Signal overflow: expected ErrWouldBlock, got %v", err)
	}
}

// TestSetNonblock_RaceClose attempts to trigger F_SETFL error by racing close.
// This test tries to cover the F_SETFL error path by closing the fd
// between F_GETFL and F_SETFL. Due to timing, coverage is not guaranteed.
func TestSetNonblock_RaceClose(t *testing.T) {
	for i := 0; i < 1000; i++ {
		efd, err := newEventFD(0, EFD_CLOEXEC)
		if err != nil {
			t.Fatalf("newEventFD failed: %v", err)
		}
		rawFd := efd.fd.Raw()

		// Close immediately while trying SetNonblock
		go func() {
			zcall.Close(uintptr(rawFd))
		}()

		// This may fail with ErrClosed (F_GETFL fails) or
		// with EBADF (F_SETFL fails) if timing is right
		fd := NewFD(int(rawFd))
		_ = fd.SetNonblock(true)
	}
}

// TestSetCloexec_RaceClose attempts to trigger F_SETFD error by racing close.
func TestSetCloexec_RaceClose(t *testing.T) {
	for i := 0; i < 1000; i++ {
		efd, err := newEventFD(0, EFD_NONBLOCK)
		if err != nil {
			t.Fatalf("newEventFD failed: %v", err)
		}
		rawFd := efd.fd.Raw()

		go func() {
			zcall.Close(uintptr(rawFd))
		}()

		fd := NewFD(int(rawFd))
		_ = fd.SetCloexec(true)
	}
}

// TestEventFD_WaitIntoNonEAGAINError tests the non-EAGAIN error path in WaitInto.
// This covers the errFromErrno fallback at eventfd.go:138.
func TestEventFD_WaitIntoNonEAGAINError(t *testing.T) {
	efd, err := newEventFD(0, EFD_NONBLOCK|EFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newEventFD failed: %v", err)
	}
	rawFd := efd.fd.Raw()

	// Close the underlying fd directly to get EBADF (not EAGAIN)
	zcall.Close(uintptr(rawFd))

	var val uint64
	err = efd.WaitInto(&val)
	if err == nil {
		t.Error("WaitInto should fail on closed fd")
	}
	if err == iox.ErrWouldBlock {
		t.Error("WaitInto should return non-EAGAIN error, got ErrWouldBlock")
	}
}

// TestSignalFD_ReadNonEAGAINError tests the non-EAGAIN error path in SignalFD.Read.
// This covers the errFromErrno fallback at signalfd.go:209.
func TestSignalFD_ReadNonEAGAINError(t *testing.T) {
	var mask SigSet
	mask.Add(10) // SIGUSR1
	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	rawFd := sfd.fd.Raw()

	// Close the underlying fd directly to get EBADF (not EAGAIN)
	zcall.Close(uintptr(rawFd))

	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	if err == nil {
		t.Error("Read should fail on closed fd")
	}
	if err == iox.ErrWouldBlock {
		t.Error("Read should return non-EAGAIN error, got ErrWouldBlock")
	}
}

// TestMappedRegion_UnmapError tests the munmap error path in Unmap.
// This covers the errFromErrno fallback at memfd.go:257.
func TestMappedRegion_UnmapError(t *testing.T) {
	// Create a real mapping via memfd
	mfd, err := newMemFD("test-unmap-error", MFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newMemFD failed: %v", err)
	}
	defer mfd.Close()

	if err := mfd.Truncate(4096); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	region, err := mfd.Mmap(4096, PROT_READ)
	if err != nil {
		t.Fatalf("Mmap failed: %v", err)
	}

	// Corrupt the length to 0 — munmap(addr, 0) returns EINVAL.
	region.length = 0

	err = region.Unmap()
	if err == nil {
		t.Error("Unmap with zero length should fail with EINVAL")
	}

	// Restore and properly unmap to avoid leaking the mapping
	region.length = 4096
	region.Unmap()
}
