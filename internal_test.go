// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"testing"

	"code.hybscloud.com/iox"
	"code.hybscloud.com/zcall"
)

// TestEventfdValuePtr tests the internal eventfdValuePtr function.
func TestEventfdValuePtr(t *testing.T) {
	val := uint64(12345)
	p := eventfdValuePtr(&val)

	if p == nil {
		t.Fatal("eventfdValuePtr returned nil")
	}

	// Verify the pointer points to the correct value
	got := *(*uint64)(p)
	if got != 12345 {
		t.Errorf("eventfdValuePtr returned wrong value: got %d, want 12345", got)
	}
}

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
	_, err = tfd.Read()
	if err == nil {
		t.Error("Read should fail on closed fd")
	}

	// ReadInto should fail
	buf := make([]byte, 8)
	_, err = tfd.ReadInto(buf)
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

	// Read should fail
	_, err = sfd.Read()
	if err == nil {
		t.Error("Read should fail on closed fd")
	}

	// ReadInto should fail
	buf := make([]byte, 128)
	_, err = sfd.ReadInto(buf)
	if err == nil {
		t.Error("ReadInto should fail on closed fd")
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
	_, err = tfd.Read()
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
	_, err = tfd.ReadInto(buf)
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

	// Read with no pending signal should return ErrWouldBlock
	_, err = sfd.Read()
	if err != iox.ErrWouldBlock {
		t.Errorf("Read with no pending signal: got %v, want ErrWouldBlock", err)
	}
}

// TestSignalFD_ReadIntoWouldBlock tests ReadInto returning ErrWouldBlock.
func TestSignalFD_ReadIntoWouldBlock(t *testing.T) {
	var mask SigSet
	mask.Add(SIGUSR1)

	sfd, err := newSignalFD(mask, SFD_NONBLOCK|SFD_CLOEXEC)
	if err != nil {
		t.Fatalf("newSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// ReadInto with no pending signal should return ErrWouldBlock
	buf := make([]byte, 128)
	_, err = sfd.ReadInto(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("ReadInto with no pending signal: got %v, want ErrWouldBlock", err)
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

	// Read with no pending signal should return ErrWouldBlock
	_, err = sfd.Read()
	if err != iox.ErrWouldBlock {
		t.Errorf("Read with no signal should return ErrWouldBlock, got %v", err)
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

	// ReadInto with exactly 128 bytes (minimum required)
	buf := make([]byte, 128)
	_, err = sfd.ReadInto(buf)
	// Should return EAGAIN since no signal is pending
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("ReadInto(128) error: %v", err)
	}

	// ReadInto with more than 128 bytes
	largeBuf := make([]byte, 256)
	_, err = sfd.ReadInto(largeBuf)
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("ReadInto(256) error: %v", err)
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
	_, err = tfd.ReadInto(buf)
	// Should return EAGAIN since timer is not armed
	if err != iox.ErrWouldBlock && err != nil {
		t.Logf("ReadInto(8) error: %v", err)
	}

	// ReadInto with more than 8 bytes
	largeBuf := make([]byte, 16)
	_, err = tfd.ReadInto(largeBuf)
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
