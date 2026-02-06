// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.hybscloud.com/iofd"
	"code.hybscloud.com/iox"
)

// =============================================================================
// EventFD Tests
// =============================================================================

func TestEventFD_Create(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	if efd.Fd() < 0 {
		t.Errorf("EventFD.Fd() returned invalid fd: %d", efd.Fd())
	}
}

func TestEventFD_CreateWithInitval(t *testing.T) {
	efd, err := iofd.NewEventFD(42)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	val, err := efd.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected initial value 42, got %d", val)
	}
}

func TestEventFD_SignalAndWait(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal with value 5
	err = efd.Signal(5)
	if err != nil {
		t.Fatalf("Signal failed: %v", err)
	}

	// Signal again with value 3 (should accumulate to 8)
	err = efd.Signal(3)
	if err != nil {
		t.Fatalf("Signal failed: %v", err)
	}

	// Wait should return accumulated value
	val, err := efd.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if val != 8 {
		t.Errorf("Expected accumulated value 8, got %d", val)
	}

	// Second wait should return ErrWouldBlock (counter reset to 0)
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

func TestEventFD_WouldBlock(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Wait on empty eventfd should return ErrWouldBlock
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock on empty eventfd, got %v", err)
	}
}

func TestEventFD_Semaphore(t *testing.T) {
	efd, err := iofd.NewEventFDSemaphore(3)
	if err != nil {
		t.Fatalf("NewEventFDSemaphore failed: %v", err)
	}
	defer efd.Close()

	// In semaphore mode, each read decrements by 1
	for i := 0; i < 3; i++ {
		val, err := efd.Wait()
		if err != nil {
			t.Fatalf("Wait %d failed: %v", i, err)
		}
		if val != 1 {
			t.Errorf("Semaphore Wait %d: expected 1, got %d", i, val)
		}
	}

	// Fourth wait should block
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock after semaphore exhausted, got %v", err)
	}
}

func TestEventFD_ReadWrite(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Write raw bytes
	buf := make([]byte, 8)
	buf[0] = 7 // little-endian uint64 = 7
	n, err := efd.Write(buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Write returned %d, expected 8", n)
	}

	// Read raw bytes
	rbuf := make([]byte, 8)
	n, err = efd.Read(rbuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Read returned %d, expected 8", n)
	}
	if rbuf[0] != 7 {
		t.Errorf("Read value mismatch: expected 7, got %d", rbuf[0])
	}
}

func TestEventFD_Close(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	err = efd.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Operations on closed fd should fail
	err = efd.Signal(1)
	if err == nil {
		t.Error("Signal on closed eventfd should fail")
	}
}

func TestEventFD_SignalZero(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal with 0 should be a no-op
	err = efd.Signal(0)
	if err != nil {
		t.Errorf("Signal(0) should succeed, got %v", err)
	}

	// Counter should still be 0
	_, err = efd.Wait()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock after Signal(0), got %v", err)
	}
}

// =============================================================================
// TimerFD Tests
// =============================================================================

func TestTimerFD_Create(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	if tfd.Fd() < 0 {
		t.Errorf("TimerFD.Fd() returned invalid fd: %d", tfd.Fd())
	}
}

func TestTimerFD_CreateRealtime(t *testing.T) {
	tfd, err := iofd.NewTimerFDRealtime()
	if err != nil {
		t.Fatalf("NewTimerFDRealtime failed: %v", err)
	}
	defer tfd.Close()

	if tfd.Fd() < 0 {
		t.Errorf("TimerFD.Fd() returned invalid fd: %d", tfd.Fd())
	}
}

func TestTimerFD_CreateBoottime(t *testing.T) {
	tfd, err := iofd.NewTimerFDBoottime()
	if err != nil {
		t.Fatalf("NewTimerFDBoottime failed: %v", err)
	}
	defer tfd.Close()

	if tfd.Fd() < 0 {
		t.Errorf("TimerFD.Fd() returned invalid fd: %d", tfd.Fd())
	}
}

func TestTimerFD_ArmAndRead(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer for 10ms one-shot
	err = tfd.Arm(10*int64(time.Millisecond), 0)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Wait for timer to expire
	time.Sleep(15 * time.Millisecond)

	// Read should return 1 expiration
	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expiration, got %d", count)
	}
}

func TestTimerFD_PeriodicTimer(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm periodic timer: first expiration at 5ms, then every 5ms
	interval := 5 * int64(time.Millisecond)
	err = tfd.Arm(interval, interval)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Wait for multiple expirations
	time.Sleep(22 * time.Millisecond)

	// Should have at least 3-4 expirations
	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count < 3 {
		t.Errorf("Expected at least 3 expirations, got %d", count)
	}
}

func TestTimerFD_Disarm(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer for 100ms
	err = tfd.Arm(100*int64(time.Millisecond), 0)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Disarm before expiration
	err = tfd.Disarm()
	if err != nil {
		t.Fatalf("Disarm failed: %v", err)
	}

	// Wait past original expiration time
	time.Sleep(150 * time.Millisecond)

	// Read should return ErrWouldBlock (no expirations)
	_, err = tfd.Expirations()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock after disarm, got %v", err)
	}
}

func TestTimerFD_WouldBlock(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Read on unarmed timer should return ErrWouldBlock
	_, err = tfd.Expirations()
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock on unarmed timer, got %v", err)
	}
}

func TestTimerFD_Close(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}

	err = tfd.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Operations on closed fd should fail
	err = tfd.Arm(int64(time.Second), 0)
	if err == nil {
		t.Error("Arm on closed timerfd should fail")
	}
}

// =============================================================================
// FD Type Tests
// =============================================================================

func TestFD_NewFD(t *testing.T) {
	// Create an eventfd to get a valid fd
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// NewFD should wrap the fd value
	fd := iofd.NewFD(efd.Fd())
	if fd.Fd() != efd.Fd() {
		t.Errorf("NewFD: expected fd %d, got %d", efd.Fd(), fd.Fd())
	}
}

func TestFD_Valid(t *testing.T) {
	// Create an eventfd to get a valid fd
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	fd := iofd.NewFD(efd.Fd())
	if !fd.Valid() {
		t.Error("Valid() should return true for valid fd")
	}

	// Close the underlying fd
	efd.Close()

	// InvalidFD should not be valid
	invalidFD := iofd.InvalidFD
	if invalidFD.Valid() {
		t.Error("InvalidFD.Valid() should return false")
	}
}

func TestFD_ReadWrite(t *testing.T) {
	// Use a pipe for testing Read/Write
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	// Write to eventfd (must be 8 bytes)
	buf := make([]byte, 8)
	buf[0] = 5 // little-endian uint64 = 5
	n, err := fd.Write(buf)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Write returned %d, expected 8", n)
	}

	// Read from eventfd
	rbuf := make([]byte, 8)
	n, err = fd.Read(rbuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Read returned %d, expected 8", n)
	}
	if rbuf[0] != 5 {
		t.Errorf("Read value mismatch: expected 5, got %d", rbuf[0])
	}
}

func TestFD_ReadWriteEmpty(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	// Write empty slice should be no-op
	n, err := fd.Write(nil)
	if err != nil {
		t.Errorf("Write(nil) failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Write(nil) returned %d, expected 0", n)
	}

	// Read empty slice should be no-op
	n, err = fd.Read(nil)
	if err != nil {
		t.Errorf("Read(nil) failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Read(nil) returned %d, expected 0", n)
	}
}

func TestFD_SetNonblock(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	// EventFD is already non-blocking, try toggling
	err = fd.SetNonblock(false)
	if err != nil {
		t.Errorf("SetNonblock(false) failed: %v", err)
	}

	err = fd.SetNonblock(true)
	if err != nil {
		t.Errorf("SetNonblock(true) failed: %v", err)
	}
}

func TestFD_SetCloexec(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	// EventFD is already cloexec, try toggling
	err = fd.SetCloexec(false)
	if err != nil {
		t.Errorf("SetCloexec(false) failed: %v", err)
	}

	err = fd.SetCloexec(true)
	if err != nil {
		t.Errorf("SetCloexec(true) failed: %v", err)
	}
}

func TestFD_Dup(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	// Duplicate the fd
	newFD, err := fd.Dup()
	if err != nil {
		t.Fatalf("Dup failed: %v", err)
	}
	defer newFD.Close()

	if !newFD.Valid() {
		t.Error("Duplicated fd should be valid")
	}
	if newFD.Fd() == fd.Fd() {
		t.Error("Duplicated fd should have different value")
	}

	// Both fds should work - write to original, read from dup
	err = efd.Signal(42)
	if err != nil {
		t.Fatalf("Signal failed: %v", err)
	}

	// Read from duplicated fd
	buf := make([]byte, 8)
	n, err := newFD.Read(buf)
	if err != nil {
		t.Fatalf("Read from dup failed: %v", err)
	}
	if n != 8 {
		t.Errorf("Read returned %d, expected 8", n)
	}
}

func TestFD_ClosedOperations(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	fd := iofd.NewFD(efd.Fd())
	efd.Close()

	// Create a new closed FD
	closedFD := iofd.InvalidFD

	// Operations on closed fd should fail
	_, err = closedFD.Read(make([]byte, 8))
	if err == nil {
		t.Error("Read on closed fd should fail")
	}

	_, err = closedFD.Write(make([]byte, 8))
	if err == nil {
		t.Error("Write on closed fd should fail")
	}

	err = closedFD.SetNonblock(true)
	if err == nil {
		t.Error("SetNonblock on closed fd should fail")
	}

	err = closedFD.SetCloexec(true)
	if err == nil {
		t.Error("SetCloexec on closed fd should fail")
	}

	_, err = closedFD.Dup()
	if err == nil {
		t.Error("Dup on closed fd should fail")
	}

	// Close on already closed should be no-op (idempotent)
	err = fd.Close()
	// This may or may not error depending on implementation
	_ = err
}

// =============================================================================
// Additional TimerFD Tests
// =============================================================================

func TestTimerFD_ArmDuration(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm with duration
	err = tfd.ArmDuration(10*time.Millisecond, 0)
	if err != nil {
		t.Fatalf("ArmDuration failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expiration, got %d", count)
	}
}

func TestTimerFD_ArmDurationPeriodic(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm periodic timer with duration
	err = tfd.ArmDuration(5*time.Millisecond, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("ArmDuration failed: %v", err)
	}

	// Wait for multiple expirations
	time.Sleep(22 * time.Millisecond)

	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count < 3 {
		t.Errorf("Expected at least 3 expirations, got %d", count)
	}
}

func TestTimerFD_GetTime(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer for 100ms
	err = tfd.Arm(100*int64(time.Millisecond), 50*int64(time.Millisecond))
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// GetTime should return remaining time
	remaining, interval, err := tfd.GetTime()
	if err != nil {
		t.Fatalf("GetTime failed: %v", err)
	}

	// Remaining should be positive and less than initial
	if remaining <= 0 || remaining > 100*int64(time.Millisecond) {
		t.Errorf("Unexpected remaining time: %d", remaining)
	}

	// Interval should match what we set
	if interval != 50*int64(time.Millisecond) {
		t.Errorf("Expected interval %d, got %d", 50*int64(time.Millisecond), interval)
	}
}

func TestTimerFD_ReadInto(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer for 10ms
	err = tfd.Arm(10*int64(time.Millisecond), 0)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// ReadInto with buffer
	buf := make([]byte, 8)
	n, err := tfd.Read(buf)
	if err != nil {
		t.Fatalf("ReadInto failed: %v", err)
	}
	if n != 8 {
		t.Errorf("ReadInto returned %d, expected 8", n)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkEventFD_SignalWait(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = efd.Signal(1)
		_, _ = efd.Wait()
	}
}

func BenchmarkEventFD_Signal(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = efd.Signal(1)
	}
	// Drain
	_, _ = efd.Wait()
}

func BenchmarkTimerFD_ArmDisarm(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = tfd.Arm(int64(time.Second), 0)
		_ = tfd.Disarm()
	}
}

func BenchmarkEventFD_Create(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		efd, _ := iofd.NewEventFD(0)
		_ = efd.Close()
	}
}

func BenchmarkTimerFD_Create(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tfd, _ := iofd.NewTimerFD()
		_ = tfd.Close()
	}
}

// =============================================================================
// SignalFD Tests
// =============================================================================

func TestSigSet_AddDelHas(t *testing.T) {
	var set iofd.SigSet

	if !set.Empty() {
		t.Error("New SigSet should be empty")
	}

	set.Add(iofd.SIGINT)
	if !set.Has(iofd.SIGINT) {
		t.Error("SigSet should have SIGINT after Add")
	}
	if set.Empty() {
		t.Error("SigSet should not be empty after Add")
	}

	set.Add(iofd.SIGTERM)
	if !set.Has(iofd.SIGTERM) {
		t.Error("SigSet should have SIGTERM after Add")
	}

	set.Del(iofd.SIGINT)
	if set.Has(iofd.SIGINT) {
		t.Error("SigSet should not have SIGINT after Del")
	}
	if !set.Has(iofd.SIGTERM) {
		t.Error("SigSet should still have SIGTERM")
	}
}

func TestSigSet_Bounds(t *testing.T) {
	var set iofd.SigSet

	// Invalid signals should be ignored
	set.Add(0)
	set.Add(65)
	set.Add(-1)

	if !set.Empty() {
		t.Error("SigSet should still be empty after invalid Add")
	}

	if set.Has(0) || set.Has(65) || set.Has(-1) {
		t.Error("Has should return false for invalid signals")
	}
}

func TestSignalFD_Create(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	if sfd.Fd() < 0 {
		t.Errorf("SignalFD.Fd() returned invalid fd: %d", sfd.Fd())
	}

	if sfd.Mask() != mask {
		t.Errorf("Mask mismatch: expected %v, got %v", mask, sfd.Mask())
	}
}

func TestSignalFD_WouldBlock(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// ReadInto on signalfd with no pending signals should return ErrWouldBlock
	var info iofd.SignalInfo
	err = sfd.ReadInto(&info)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock on empty signalfd, got %v", err)
	}
}

func TestSignalFD_SetMask(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Update mask
	var newMask iofd.SigSet
	newMask.Add(iofd.SIGUSR2)
	newMask.Add(iofd.SIGTERM)

	err = sfd.SetMask(newMask)
	if err != nil {
		t.Fatalf("SetMask failed: %v", err)
	}

	if sfd.Mask() != newMask {
		t.Errorf("Mask not updated: expected %v, got %v", newMask, sfd.Mask())
	}
}

func TestSignalFD_Close(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}

	err = sfd.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Operations on closed fd should fail
	var info iofd.SignalInfo
	err = sfd.ReadInto(&info)
	if err == nil {
		t.Error("ReadInto on closed signalfd should fail")
	}
}

// =============================================================================
// PidFD Tests
// =============================================================================

func TestPidFD_Create(t *testing.T) {
	// Create a pidfd for our own process
	pid := 1 // init process, always exists
	pfd, err := iofd.NewPidFD(pid)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	if pfd.Fd() < 0 {
		t.Errorf("PidFD.Fd() returned invalid fd: %d", pfd.Fd())
	}

	if pfd.PID() != pid {
		t.Errorf("PID mismatch: expected %d, got %d", pid, pfd.PID())
	}

	if !pfd.Valid() {
		t.Error("PidFD should be valid")
	}
}

func TestPidFD_InvalidPID(t *testing.T) {
	_, err := iofd.NewPidFD(0)
	if err == nil {
		t.Error("NewPidFD(0) should fail")
	}

	_, err = iofd.NewPidFD(-1)
	if err == nil {
		t.Error("NewPidFD(-1) should fail")
	}
}

func TestPidFD_NonexistentPID(t *testing.T) {
	// Use a very high PID that almost certainly doesn't exist
	_, err := iofd.NewPidFD(999999999)
	if err == nil {
		t.Error("NewPidFD for nonexistent PID should fail")
	}
}

func TestPidFD_Close(t *testing.T) {
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}

	err = pfd.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if pfd.Valid() {
		t.Error("PidFD should not be valid after close")
	}

	// Operations on closed fd should fail
	err = pfd.SendSignal(0)
	if err == nil {
		t.Error("SendSignal on closed pidfd should fail")
	}
}

func TestPidFD_SendSignal(t *testing.T) {
	// Send signal 0 (null signal) to init process - used for checking if process exists
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	// Signal 0 should succeed (just checks if process exists)
	err = pfd.SendSignal(0)
	if err != nil {
		// This may fail due to permissions, which is acceptable
		t.Logf("SendSignal(0) returned: %v (may be permission denied)", err)
	}
}

func TestPidFD_Valid(t *testing.T) {
	// PidFD for init process should be valid
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	if !pfd.Valid() {
		t.Error("PidFD should be valid for running process")
	}

	if pfd.Fd() < 0 {
		t.Error("PidFD.Fd() should return valid fd")
	}
}

// =============================================================================
// MemFD Tests
// =============================================================================

func TestMemFD_Create(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-memfd")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	if mfd.Fd() < 0 {
		t.Errorf("MemFD.Fd() returned invalid fd: %d", mfd.Fd())
	}

	if mfd.Name() != "test-memfd" {
		t.Errorf("Name mismatch: expected 'test-memfd', got '%s'", mfd.Name())
	}

	if !mfd.Valid() {
		t.Error("MemFD should be valid")
	}
}

func TestMemFD_ReadWrite(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-rw")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Set size first
	err = mfd.Truncate(1024)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Write data
	data := []byte("Hello, memfd!")
	n, err := mfd.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, expected %d", n, len(data))
	}

	// Seek back to beginning (using pread would be better, but we don't have lseek)
	// For now, just verify the size
	size, err := mfd.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 1024 {
		t.Errorf("Size mismatch: expected 1024, got %d", size)
	}
}

func TestMemFD_PreadPwrite(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-pread-pwrite")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Set up file with some content
	if err := mfd.Truncate(100); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Pwrite at offset 10
	data := []byte("hello")
	n, err := mfd.Pwrite(data, 10)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Pwrite wrote %d bytes, expected %d", n, len(data))
	}

	// Pwrite at offset 50
	data2 := []byte("world")
	n, err = mfd.Pwrite(data2, 50)
	if err != nil {
		t.Fatalf("Pwrite failed: %v", err)
	}
	if n != len(data2) {
		t.Errorf("Pwrite wrote %d bytes, expected %d", n, len(data2))
	}

	// Pread from offset 10
	buf := make([]byte, 5)
	n, err = mfd.Pread(buf, 10)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Pread read %d bytes, expected 5", n)
	}
	if string(buf) != "hello" {
		t.Errorf("Pread got %q, expected %q", string(buf), "hello")
	}

	// Pread from offset 50
	n, err = mfd.Pread(buf, 50)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if n != 5 {
		t.Errorf("Pread read %d bytes, expected 5", n)
	}
	if string(buf) != "world" {
		t.Errorf("Pread got %q, expected %q", string(buf), "world")
	}

	// Test empty buffer (no-op)
	n, err = mfd.Pread(nil, 0)
	if err != nil {
		t.Errorf("Pread(nil) failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Pread(nil) returned %d, expected 0", n)
	}

	n, err = mfd.Pwrite(nil, 0)
	if err != nil {
		t.Errorf("Pwrite(nil) failed: %v", err)
	}
	if n != 0 {
		t.Errorf("Pwrite(nil) returned %d, expected 0", n)
	}
}

func TestMemFD_PreadPwriteOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-pread-closed")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	mfd.Close()

	buf := make([]byte, 10)
	_, err = mfd.Pread(buf, 0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}

	_, err = mfd.Pwrite(buf, 0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_PwriteSealed(t *testing.T) {
	// Create a memfd with sealing enabled
	mfd, err := iofd.NewMemFDSealed("test-pwrite-sealed")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	defer mfd.Close()

	// Set size and seal writes
	if err := mfd.Truncate(100); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}
	if err := mfd.Seal(iofd.F_SEAL_WRITE); err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	// Pwrite should fail with permission error
	buf := []byte("test")
	_, err = mfd.Pwrite(buf, 0)
	if err == nil {
		t.Error("Expected error writing to sealed memfd, got nil")
	}
}

func TestMemFD_PreadNegativeOffset(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-pread-negative")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	if err := mfd.Truncate(100); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Negative offset should fail
	buf := make([]byte, 10)
	_, err = mfd.Pread(buf, -1)
	if err == nil {
		t.Error("Expected error with negative offset, got nil")
	}
}

func TestMemFD_Truncate(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-truncate")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
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

func TestMemFD_Sealing(t *testing.T) {
	mfd, err := iofd.NewMemFDSealed("test-seal")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	defer mfd.Close()

	// Set size first
	err = mfd.Truncate(1024)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Get initial seals (should be none)
	seals, err := mfd.Seals()
	if err != nil {
		t.Fatalf("Seals failed: %v", err)
	}
	if seals != 0 {
		t.Errorf("Initial seals should be 0, got %d", seals)
	}

	// Add shrink seal
	err = mfd.Seal(iofd.F_SEAL_SHRINK)
	if err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	seals, err = mfd.Seals()
	if err != nil {
		t.Fatalf("Seals failed: %v", err)
	}
	if seals&iofd.F_SEAL_SHRINK == 0 {
		t.Error("F_SEAL_SHRINK should be set")
	}

	// Shrinking should now fail
	err = mfd.Truncate(512)
	if err == nil {
		t.Error("Truncate to smaller size should fail after F_SEAL_SHRINK")
	}

	// Growing should still work
	err = mfd.Truncate(2048)
	if err != nil {
		t.Errorf("Truncate to larger size should work: %v", err)
	}
}

func TestMemFD_Close(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-close")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}

	err = mfd.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if mfd.Valid() {
		t.Error("MemFD should not be valid after close")
	}

	// Operations on closed fd should fail
	_, err = mfd.Size()
	if err == nil {
		t.Error("Size on closed memfd should fail")
	}
}

// =============================================================================
// Additional Benchmarks
// =============================================================================

func BenchmarkSignalFD_Create(b *testing.B) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sfd, _ := iofd.NewSignalFD(mask)
		_ = sfd.Close()
	}
}

func BenchmarkPidFD_Create(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		pfd, _ := iofd.NewPidFD(1)
		_ = pfd.Close()
	}
}

func BenchmarkMemFD_Create(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mfd, _ := iofd.NewMemFD("bench")
		_ = mfd.Close()
	}
}

func BenchmarkMemFD_Truncate(b *testing.B) {
	mfd, err := iofd.NewMemFD("bench-truncate")
	if err != nil {
		b.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = mfd.Truncate(4096)
	}
}

// =============================================================================
// Additional FD Tests for Coverage
// =============================================================================

func TestFD_CloseIdempotent(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	fd := iofd.NewFD(efd.Fd())

	// First close should succeed
	err = fd.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	// Second close should be no-op (idempotent)
	err = fd.Close()
	if err != nil {
		t.Errorf("Second close should be no-op: %v", err)
	}

	// Third close should also be no-op
	err = fd.Close()
	if err != nil {
		t.Errorf("Third close should be no-op: %v", err)
	}
}

func TestFD_InvalidOperations(t *testing.T) {
	// Test operations on InvalidFD
	invalidFD := iofd.InvalidFD

	if invalidFD.Valid() {
		t.Error("InvalidFD should not be valid")
	}

	if invalidFD.Fd() >= 0 {
		t.Error("InvalidFD.Fd() should return negative")
	}

	if invalidFD.Raw() >= 0 {
		t.Error("InvalidFD.Raw() should return negative")
	}
}

func TestEventFD_LargeSignal(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal with large value
	largeVal := uint64(0xFFFFFFFF)
	err = efd.Signal(largeVal)
	if err != nil {
		t.Fatalf("Signal with large value failed: %v", err)
	}

	val, err := efd.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if val != largeVal {
		t.Errorf("Expected %d, got %d", largeVal, val)
	}
}

func TestEventFD_MultipleSignals(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Signal multiple times
	for i := uint64(1); i <= 10; i++ {
		err = efd.Signal(i)
		if err != nil {
			t.Fatalf("Signal %d failed: %v", i, err)
		}
	}

	// Wait should return sum: 1+2+3+...+10 = 55
	val, err := efd.Wait()
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if val != 55 {
		t.Errorf("Expected sum 55, got %d", val)
	}
}

func TestTimerFD_GetTimeUnarmed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// GetTime on unarmed timer should return zeros
	remaining, interval, err := tfd.GetTime()
	if err != nil {
		t.Fatalf("GetTime failed: %v", err)
	}
	if remaining != 0 {
		t.Errorf("Unarmed timer should have 0 remaining, got %d", remaining)
	}
	if interval != 0 {
		t.Errorf("Unarmed timer should have 0 interval, got %d", interval)
	}
}

func TestTimerFD_RearmTimer(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm for 1 second
	err = tfd.Arm(int64(time.Second), 0)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Rearm for 10ms
	err = tfd.Arm(10*int64(time.Millisecond), 0)
	if err != nil {
		t.Fatalf("Rearm failed: %v", err)
	}

	// Wait and read
	time.Sleep(15 * time.Millisecond)
	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expiration, got %d", count)
	}
}

func TestTimerFD_ReadIntoSmallBuffer(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// ReadInto with too small buffer should fail
	buf := make([]byte, 4)
	_, err = tfd.Read(buf)
	if err == nil {
		t.Error("ReadInto with small buffer should fail")
	}
}

func TestSignalFD_ReadSmallBuffer(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Read (io.Reader) with too small buffer should fail
	buf := make([]byte, 64)
	_, err = sfd.Read(buf)
	if err == nil {
		t.Error("Read with small buffer should fail")
	}
}

func TestSignalFD_MultipleSignals(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)
	mask.Add(iofd.SIGUSR2)
	mask.Add(iofd.SIGTERM)
	mask.Add(iofd.SIGINT)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	if !mask.Has(iofd.SIGUSR1) || !mask.Has(iofd.SIGUSR2) {
		t.Error("Mask should contain both SIGUSR1 and SIGUSR2")
	}
}

func TestMemFD_EmptyName(t *testing.T) {
	mfd, err := iofd.NewMemFD("")
	if err != nil {
		t.Fatalf("NewMemFD with empty name failed: %v", err)
	}
	defer mfd.Close()

	if mfd.Name() != "" {
		t.Errorf("Expected empty name, got '%s'", mfd.Name())
	}
}

func TestMemFD_LongName(t *testing.T) {
	// memfd name can be up to 249 bytes
	longName := string(make([]byte, 200))
	for i := range longName {
		longName = longName[:i] + "a" + longName[i+1:]
	}
	longName = "test-long-name-" + longName[:100]

	mfd, err := iofd.NewMemFD(longName)
	if err != nil {
		t.Fatalf("NewMemFD with long name failed: %v", err)
	}
	defer mfd.Close()
}

func TestMemFD_SealAll(t *testing.T) {
	mfd, err := iofd.NewMemFDSealed("test-seal-all")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	defer mfd.Close()

	// Set size
	err = mfd.Truncate(4096)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Apply all seals at once
	allSeals := uint(iofd.F_SEAL_SHRINK | iofd.F_SEAL_GROW | iofd.F_SEAL_WRITE | iofd.F_SEAL_SEAL)
	err = mfd.Seal(allSeals)
	if err != nil {
		t.Fatalf("Seal all failed: %v", err)
	}

	// Verify seals
	seals, err := mfd.Seals()
	if err != nil {
		t.Fatalf("Seals failed: %v", err)
	}

	if seals&iofd.F_SEAL_SHRINK == 0 {
		t.Error("F_SEAL_SHRINK not set")
	}
	if seals&iofd.F_SEAL_GROW == 0 {
		t.Error("F_SEAL_GROW not set")
	}
	if seals&iofd.F_SEAL_WRITE == 0 {
		t.Error("F_SEAL_WRITE not set")
	}
	if seals&iofd.F_SEAL_SEAL == 0 {
		t.Error("F_SEAL_SEAL not set")
	}

	// Further sealing should fail due to F_SEAL_SEAL
	err = mfd.Seal(iofd.F_SEAL_FUTURE_WRITE)
	if err == nil {
		t.Error("Sealing after F_SEAL_SEAL should fail")
	}
}

func TestMemFD_WriteReadCycle(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-cycle")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Set size
	err = mfd.Truncate(1024)
	if err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Write pattern
	pattern := []byte("ABCDEFGHIJKLMNOP")
	n, err := mfd.Write(pattern)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(pattern) {
		t.Errorf("Write returned %d, expected %d", n, len(pattern))
	}

	// Verify size unchanged
	size, err := mfd.Size()
	if err != nil {
		t.Fatalf("Size failed: %v", err)
	}
	if size != 1024 {
		t.Errorf("Size changed unexpectedly: %d", size)
	}
}

func TestPidFD_GetFD(t *testing.T) {
	// Create a pidfd for our own process
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	// GetFD requires special privileges, so this may fail with EPERM
	_, err = pfd.GetFD(0) // Try to get stdin from init
	if err == nil {
		t.Log("GetFD succeeded (running with elevated privileges)")
	} else {
		t.Logf("GetFD failed as expected without privileges: %v", err)
	}
}

// =============================================================================
// Additional Benchmarks for Coverage
// =============================================================================

func BenchmarkFD_Raw(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fd.Raw()
	}
}

func BenchmarkFD_Valid(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fd.Valid()
	}
}

func BenchmarkEventFD_ReadWrite(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	wbuf := make([]byte, 8)
	wbuf[0] = 1
	rbuf := make([]byte, 8)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = efd.Write(wbuf)
		_, _ = efd.Read(rbuf)
	}
}

func BenchmarkTimerFD_GetTime(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	_ = tfd.Arm(int64(time.Hour), int64(time.Hour))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = tfd.GetTime()
	}
}

func BenchmarkSigSet_Add(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var set iofd.SigSet
		set.Add(iofd.SIGUSR1)
		set.Add(iofd.SIGUSR2)
		set.Add(iofd.SIGTERM)
	}
}

func BenchmarkSigSet_Has(b *testing.B) {
	var set iofd.SigSet
	set.Add(iofd.SIGUSR1)
	set.Add(iofd.SIGUSR2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = set.Has(iofd.SIGUSR1)
		_ = set.Has(iofd.SIGTERM)
	}
}

func BenchmarkMemFD_Size(b *testing.B) {
	mfd, err := iofd.NewMemFD("bench-size")
	if err != nil {
		b.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	_ = mfd.Truncate(4096)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = mfd.Size()
	}
}

func BenchmarkMemFD_ReadWrite(b *testing.B) {
	mfd, err := iofd.NewMemFD("bench-rw")
	if err != nil {
		b.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	_ = mfd.Truncate(4096)
	buf := make([]byte, 64)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = mfd.Write(buf)
	}
}

func BenchmarkSignalFD_SetMask(b *testing.B) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		b.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	var newMask iofd.SigSet
	newMask.Add(iofd.SIGUSR2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = sfd.SetMask(newMask)
	}
}

func BenchmarkPidFD_SendSignal(b *testing.B) {
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		b.Fatalf("NewPidFD failed: %v", err)
	}
	defer pfd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Signal 0 just checks process existence
		_ = pfd.SendSignal(0)
	}
}

// =============================================================================
// Additional Tests for Coverage
// =============================================================================

func TestNewPidFDBlocking(t *testing.T) {
	// Create a blocking pidfd for init process
	pfd, err := iofd.NewPidFDBlocking(1)
	if err != nil {
		t.Fatalf("NewPidFDBlocking failed: %v", err)
	}
	defer pfd.Close()

	if pfd.Fd() < 0 {
		t.Errorf("PidFD.Fd() returned invalid fd: %d", pfd.Fd())
	}
	if pfd.PID() != 1 {
		t.Errorf("Expected PID 1, got %d", pfd.PID())
	}
}

func TestNewMemFDHugeTLB(t *testing.T) {
	// HugeTLB requires special system configuration, so this may fail
	mfd, err := iofd.NewMemFDHugeTLB("test-hugetlb")
	if err != nil {
		// Expected to fail without hugepages configured
		t.Logf("NewMemFDHugeTLB failed (expected without hugepages): %v", err)
		return
	}
	defer mfd.Close()

	if mfd.Fd() < 0 {
		t.Errorf("MemFD.Fd() returned invalid fd: %d", mfd.Fd())
	}
}

func TestMemFD_Read(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-read")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Truncate to set size
	if err := mfd.Truncate(64); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Write some data
	data := []byte("hello world")
	n, err := mfd.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, expected %d", n, len(data))
	}

	// Read back - note: file position is at end after write
	// We need to use a fresh memfd or seek, but MemFD doesn't expose seek
	// So we test that Read works on an empty position
	buf := make([]byte, 64)
	n, err = mfd.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	// Read at current position (after the data we wrote)
	t.Logf("Read %d bytes", n)
}

func TestTimerFD_ArmAt(t *testing.T) {
	// Use CLOCK_REALTIME for absolute time test since we have wall clock time
	tfd, err := iofd.NewTimerFDRealtime()
	if err != nil {
		t.Fatalf("NewTimerFDRealtime failed: %v", err)
	}
	defer tfd.Close()

	// Set timer to fire at an absolute wall clock time 50ms from now
	deadline := time.Now().Add(50 * time.Millisecond).UnixNano()
	err = tfd.ArmAt(deadline, 0)
	if err != nil {
		t.Fatalf("ArmAt failed: %v", err)
	}

	// Wait for timer to fire
	time.Sleep(100 * time.Millisecond)

	// Read should return the expiration count
	count, err := tfd.Expirations()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if count < 1 {
		t.Errorf("Expected at least 1 expiration, got %d", count)
	}
}

func TestTimerFD_ArmAtClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	// ArmAt on closed fd should return error
	err = tfd.ArmAt(time.Now().UnixNano(), 0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestSignalFD_Del(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)
	mask.Add(iofd.SIGUSR2)

	// Test Del on valid signal
	mask.Del(iofd.SIGUSR1)
	if mask.Has(iofd.SIGUSR1) {
		t.Error("Del failed to remove SIGUSR1")
	}
	if !mask.Has(iofd.SIGUSR2) {
		t.Error("Del incorrectly removed SIGUSR2")
	}

	// Test Del on invalid signal (out of bounds)
	mask.Del(0)   // Below valid range
	mask.Del(100) // Above valid range (signals are 1-64)
}

func TestSignalFD_ReadInto(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	defer sfd.Close()

	// Read (io.Reader) with small buffer should fail
	smallBuf := make([]byte, 4)
	_, err = sfd.Read(smallBuf)
	if err != iofd.ErrInvalidParam {
		t.Errorf("Expected ErrInvalidParam for small buffer, got %v", err)
	}

	// Read (io.Reader) with sufficient buffer (no signal pending) should return WouldBlock
	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

func TestFD_DupFallback(t *testing.T) {
	// Create an eventfd to have a valid fd
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Wrap in FD and dup
	fd := iofd.NewFD(efd.Fd())
	dupFd, err := fd.Dup()
	if err != nil {
		t.Fatalf("Dup failed: %v", err)
	}
	defer dupFd.Close()

	if dupFd.Fd() < 0 {
		t.Errorf("Dup returned invalid fd: %d", dupFd.Fd())
	}
	if dupFd.Fd() == fd.Fd() {
		t.Errorf("Dup returned same fd as original")
	}
}

// =============================================================================
// Error Path Coverage Tests
// =============================================================================

func TestEventFD_SignalOnClosed(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	efd.Close()

	err = efd.Signal(1)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestEventFD_WaitOnClosed(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	efd.Close()

	_, err = efd.Wait()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestEventFD_ReadOnClosed(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	efd.Close()

	buf := make([]byte, 8)
	_, err = efd.Read(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestEventFD_WriteOnClosed(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	efd.Close()

	buf := make([]byte, 8)
	buf[0] = 1
	_, err = efd.Write(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_ArmOnClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	err = tfd.Arm(1000000, 0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_ReadOnClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	_, err = tfd.Expirations()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_ReadIntoOnClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	buf := make([]byte, 8)
	_, err = tfd.Read(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_GetTimeOnClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	_, _, err = tfd.GetTime()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_TruncateOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-closed")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	mfd.Close()

	err = mfd.Truncate(1024)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_SizeOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-closed")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	mfd.Close()

	_, err = mfd.Size()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_SealOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFDSealed("test-closed")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	mfd.Close()

	err = mfd.Seal(iofd.F_SEAL_WRITE)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_SealsOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFDSealed("test-closed")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	mfd.Close()

	_, err = mfd.Seals()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_Mmap(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-mmap")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Set size before mapping
	size := 4096
	if err := mfd.Truncate(int64(size)); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Map the memfd
	region, err := mfd.Mmap(size, iofd.PROT_READ|iofd.PROT_WRITE)
	if err != nil {
		t.Fatalf("Mmap failed: %v", err)
	}
	defer region.Unmap()

	// Test MappedRegion methods
	if region.Ptr() == nil {
		t.Fatal("Ptr returned nil")
	}
	if region.Len() != size {
		t.Fatalf("Len = %d, want %d", region.Len(), size)
	}

	// Write via Bytes slice
	data := region.Bytes()
	if len(data) != size {
		t.Fatalf("Bytes len = %d, want %d", len(data), size)
	}
	copy(data, []byte("hello mmap"))

	// Read back via memfd
	buf := make([]byte, 10)
	n, err := mfd.Pread(buf, 0)
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if string(buf[:n]) != "hello mmap" {
		t.Fatalf("Read = %q, want %q", buf[:n], "hello mmap")
	}
}

func TestMemFD_MmapAt(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-mmap-at")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Set size for two pages
	pageSize := 4096
	if err := mfd.Truncate(int64(pageSize * 2)); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Map second page only
	region, err := mfd.MmapAt(pageSize, iofd.PROT_READ|iofd.PROT_WRITE, int64(pageSize))
	if err != nil {
		t.Fatalf("MmapAt failed: %v", err)
	}
	defer region.Unmap()

	// Write to mapped region
	data := region.Bytes()
	copy(data, []byte("page2"))

	// Read from second page via memfd
	buf := make([]byte, 5)
	n, err := mfd.Pread(buf, int64(pageSize))
	if err != nil {
		t.Fatalf("Pread failed: %v", err)
	}
	if string(buf[:n]) != "page2" {
		t.Fatalf("Read = %q, want %q", buf[:n], "page2")
	}
}

func TestMemFD_MmapOnClosed(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-mmap-closed")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	mfd.Close()

	_, err = mfd.Mmap(4096, iofd.PROT_READ)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestMemFD_MmapInvalidLength(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-mmap-invalid")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	_, err = mfd.Mmap(0, iofd.PROT_READ)
	if err != iofd.ErrInvalidParam {
		t.Errorf("Expected ErrInvalidParam, got %v", err)
	}

	_, err = mfd.Mmap(-1, iofd.PROT_READ)
	if err != iofd.ErrInvalidParam {
		t.Errorf("Expected ErrInvalidParam, got %v", err)
	}
}

func TestMappedRegion_UnmapIdempotent(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-unmap")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	if err := mfd.Truncate(4096); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	region, err := mfd.Mmap(4096, iofd.PROT_READ)
	if err != nil {
		t.Fatalf("Mmap failed: %v", err)
	}

	// First unmap
	if err := region.Unmap(); err != nil {
		t.Fatalf("First Unmap failed: %v", err)
	}

	// Second unmap should be no-op
	if err := region.Unmap(); err != nil {
		t.Fatalf("Second Unmap failed: %v", err)
	}
}

func TestMemFD_MmapSyscallError(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-mmap-error")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}
	defer mfd.Close()

	// Don't truncate - memfd size is 0
	// Try to map with non-page-aligned offset to trigger EINVAL
	_, err = mfd.MmapAt(4096, iofd.PROT_READ, 1) // offset 1 is not page-aligned
	if err == nil {
		t.Fatal("Expected error for non-page-aligned offset")
	}
}

func TestPidFD_SendSignalOnClosed(t *testing.T) {
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	pfd.Close()

	err = pfd.SendSignal(0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestPidFD_GetFDOnClosed(t *testing.T) {
	pfd, err := iofd.NewPidFD(1)
	if err != nil {
		t.Fatalf("NewPidFD failed: %v", err)
	}
	pfd.Close()

	_, err = pfd.GetFD(0)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestSignalFD_ReadIntoOnClosed(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	sfd.Close()

	var info iofd.SignalInfo
	err = sfd.ReadInto(&info)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestSignalFD_ReadIOReaderOnClosed(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	sfd.Close()

	buf := make([]byte, 128)
	_, err = sfd.Read(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestSignalFD_SetMaskOnClosed(t *testing.T) {
	var mask iofd.SigSet
	mask.Add(iofd.SIGUSR1)

	sfd, err := iofd.NewSignalFD(mask)
	if err != nil {
		t.Fatalf("NewSignalFD failed: %v", err)
	}
	sfd.Close()

	err = sfd.SetMask(mask)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestFD_SetNonblockOnClosed(t *testing.T) {
	fd := iofd.NewFD(999999) // Invalid fd
	err := fd.Close()        // Close it
	if err != nil {
		// Expected - invalid fd
	}

	fd2 := iofd.NewFD(-1) // Already invalid
	err = fd2.SetNonblock(true)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestFD_SetCloexecOnClosed(t *testing.T) {
	fd := iofd.NewFD(-1) // Invalid fd
	err := fd.SetCloexec(true)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestFD_DupOnClosed(t *testing.T) {
	fd := iofd.NewFD(-1) // Invalid fd
	_, err := fd.Dup()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestFD_ReadOnClosed(t *testing.T) {
	fd := iofd.NewFD(-1) // Invalid fd
	buf := make([]byte, 8)
	_, err := fd.Read(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestFD_WriteOnClosed(t *testing.T) {
	fd := iofd.NewFD(-1) // Invalid fd
	buf := make([]byte, 8)
	_, err := fd.Write(buf)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

// =============================================================================
// Edge Case Tests for Coverage
// =============================================================================

func TestEventFD_ReadSmallBuffer(t *testing.T) {
	efd, err := iofd.NewEventFD(1)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Read with buffer < 8 bytes should return ErrInvalidParam
	buf := make([]byte, 4)
	_, err = efd.Read(buf)
	if err != iofd.ErrInvalidParam {
		t.Errorf("Expected ErrInvalidParam for small buffer, got %v", err)
	}
}

func TestEventFD_WriteSmallBuffer(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Write with buffer < 8 bytes should return ErrInvalidParam
	buf := make([]byte, 4)
	_, err = efd.Write(buf)
	if err != iofd.ErrInvalidParam {
		t.Errorf("Expected ErrInvalidParam for small buffer, got %v", err)
	}
}

func TestFD_ReadEmptyBuffer(t *testing.T) {
	efd, err := iofd.NewEventFD(1)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())
	// Read with empty buffer should return 0, nil
	n, err := fd.Read(nil)
	if n != 0 || err != nil {
		t.Errorf("Read(nil) should return (0, nil), got (%d, %v)", n, err)
	}

	n, err = fd.Read([]byte{})
	if n != 0 || err != nil {
		t.Errorf("Read([]) should return (0, nil), got (%d, %v)", n, err)
	}
}

func TestFD_WriteEmptyBuffer(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	fd := iofd.NewFD(efd.Fd())
	// Write with empty buffer should return 0, nil
	n, err := fd.Write(nil)
	if n != 0 || err != nil {
		t.Errorf("Write(nil) should return (0, nil), got (%d, %v)", n, err)
	}

	n, err = fd.Write([]byte{})
	if n != 0 || err != nil {
		t.Errorf("Write([]) should return (0, nil), got (%d, %v)", n, err)
	}
}

func TestSigSet_OutOfRange(t *testing.T) {
	var s iofd.SigSet

	// Add/Del/Has with out-of-range signals should be no-ops or return false
	s.Add(0)  // Below range
	s.Add(65) // Above range
	s.Add(-1) // Negative

	if !s.Empty() {
		t.Error("SigSet should be empty after adding out-of-range signals")
	}

	s.Del(0)  // Should be no-op
	s.Del(65) // Should be no-op

	if s.Has(0) {
		t.Error("Has(0) should return false")
	}
	if s.Has(65) {
		t.Error("Has(65) should return false")
	}
	if s.Has(-1) {
		t.Error("Has(-1) should return false")
	}
}

// =============================================================================
// EAGAIN Handling Tests for ReadInto methods
// =============================================================================

func TestEventFD_ReadWouldBlock(t *testing.T) {
	efd, err := iofd.NewEventFD(0) // counter = 0
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Read on empty counter should return ErrWouldBlock
	buf := make([]byte, 8)
	_, err = efd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

func TestTimerFD_ReadIntoWouldBlock(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// ReadInto on unarmed timer should return ErrWouldBlock
	buf := make([]byte, 8)
	_, err = tfd.Read(buf)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

// =============================================================================
// Valid() Method Coverage Tests
// =============================================================================

func TestEventFD_Valid(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}

	// Should be valid when open
	if !efd.Valid() {
		t.Error("EventFD should be valid when open")
	}

	efd.Close()

	// Should be invalid after close
	if efd.Valid() {
		t.Error("EventFD should be invalid after close")
	}
}

func TestTimerFD_Valid(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}

	// Should be valid when open
	if !tfd.Valid() {
		t.Error("TimerFD should be valid when open")
	}

	tfd.Close()

	// Should be invalid after close
	if tfd.Valid() {
		t.Error("TimerFD should be invalid after close")
	}
}

func TestMemFD_Valid(t *testing.T) {
	mfd, err := iofd.NewMemFD("test-valid")
	if err != nil {
		t.Fatalf("NewMemFD failed: %v", err)
	}

	// Should be valid when open
	if !mfd.Valid() {
		t.Error("MemFD should be valid when open")
	}

	mfd.Close()

	// Should be invalid after close
	if mfd.Valid() {
		t.Error("MemFD should be invalid after close")
	}
}

// =============================================================================
// New API Tests
// =============================================================================

func TestEventFD_WaitInto(t *testing.T) {
	efd, err := iofd.NewEventFD(42)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	var val uint64
	err = efd.WaitInto(&val)
	if err != nil {
		t.Fatalf("WaitInto failed: %v", err)
	}
	if val != 42 {
		t.Errorf("Expected 42, got %d", val)
	}
}

func TestEventFD_WaitIntoWouldBlock(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	var val uint64
	err = efd.WaitInto(&val)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

func TestEventFD_WaitIntoClosed(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	efd.Close()

	var val uint64
	err = efd.WaitInto(&val)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_ExpirationsInto(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer for 5ms one-shot
	err = tfd.Arm(5*int64(time.Millisecond), 0)
	if err != nil {
		t.Fatalf("Arm failed: %v", err)
	}

	// Wait for timer to expire
	time.Sleep(10 * time.Millisecond)

	var count uint64
	err = tfd.ExpirationsInto(&count)
	if err != nil {
		t.Fatalf("ExpirationsInto failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expiration, got %d", count)
	}
}

func TestTimerFD_ExpirationsIntoWouldBlock(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Timer not armed, should return ErrWouldBlock
	var count uint64
	err = tfd.ExpirationsInto(&count)
	if err != iox.ErrWouldBlock {
		t.Errorf("Expected ErrWouldBlock, got %v", err)
	}
}

func TestTimerFD_ExpirationsIntoClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	var count uint64
	err = tfd.ExpirationsInto(&count)
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

func TestTimerFD_GetTimeDuration(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm timer
	err = tfd.ArmDuration(100*time.Millisecond, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("ArmDuration failed: %v", err)
	}

	// Get time using duration API
	remaining, interval, err := tfd.GetTimeDuration()
	if err != nil {
		t.Fatalf("GetTimeDuration failed: %v", err)
	}

	// Remaining should be positive and less than initial
	if remaining <= 0 || remaining > 100*time.Millisecond {
		t.Errorf("Unexpected remaining time: %v", remaining)
	}

	// Interval should match what we set
	if interval != 50*time.Millisecond {
		t.Errorf("Expected interval %v, got %v", 50*time.Millisecond, interval)
	}
}

func TestTimerFD_GetTimeDurationClosed(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	tfd.Close()

	_, _, err = tfd.GetTimeDuration()
	if err != iofd.ErrClosed {
		t.Errorf("Expected ErrClosed, got %v", err)
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestTimerFD_ZeroDuration(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm with zero duration should disarm
	err = tfd.ArmDuration(0, 0)
	if err != nil {
		t.Fatalf("ArmDuration(0, 0) failed: %v", err)
	}

	// Timer should be disarmed
	remaining, interval, err := tfd.GetTimeDuration()
	if err != nil {
		t.Fatalf("GetTimeDuration failed: %v", err)
	}
	if remaining != 0 || interval != 0 {
		t.Errorf("Expected (0, 0), got (%v, %v)", remaining, interval)
	}
}

func TestMemFD_SealInteractions(t *testing.T) {
	mfd, err := iofd.NewMemFDSealed("test-seal-interact")
	if err != nil {
		t.Fatalf("NewMemFDSealed failed: %v", err)
	}
	defer mfd.Close()

	// Set size
	if err := mfd.Truncate(4096); err != nil {
		t.Fatalf("Truncate failed: %v", err)
	}

	// Seal against shrink
	if err := mfd.Seal(iofd.F_SEAL_SHRINK); err != nil {
		t.Fatalf("Seal failed: %v", err)
	}

	// Truncate to larger should work
	if err := mfd.Truncate(8192); err != nil {
		t.Errorf("Growing after F_SEAL_SHRINK should work: %v", err)
	}

	// Truncate to smaller should fail
	err = mfd.Truncate(4096)
	if err == nil {
		t.Error("Shrinking after F_SEAL_SHRINK should fail")
	}

	// Map should still work
	region, err := mfd.Mmap(4096, iofd.PROT_READ)
	if err != nil {
		t.Fatalf("Mmap failed: %v", err)
	}
	region.Unmap()
}

func TestEventFD_AlternatingSignalWait(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	// Alternating signal and wait
	for i := uint64(1); i <= 10; i++ {
		err = efd.Signal(i)
		if err != nil {
			t.Fatalf("Signal(%d) failed: %v", i, err)
		}

		val, err := efd.Wait()
		if err != nil {
			t.Fatalf("Wait failed: %v", err)
		}
		if val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}
}

// =============================================================================
// Concurrency Tests
// =============================================================================

func TestEventFD_ConcurrentSignalWait(t *testing.T) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		t.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	const numOps = 1000
	var wg sync.WaitGroup
	var totalSignaled, totalWaited int64

	// Signalers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps/10; j++ {
				if err := efd.Signal(1); err == nil {
					atomic.AddInt64(&totalSignaled, 1)
				}
			}
		}()
	}

	// Waiters
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps/10; j++ {
				if val, err := efd.Wait(); err == nil {
					atomic.AddInt64(&totalWaited, int64(val))
				}
			}
		}()
	}

	wg.Wait()

	// Drain any remaining
	for {
		val, err := efd.Wait()
		if err == iox.ErrWouldBlock {
			break
		}
		if err == nil {
			totalWaited += int64(val)
		}
	}

	if totalSignaled != totalWaited {
		t.Errorf("Signaled %d, waited %d", totalSignaled, totalWaited)
	}
}

func TestTimerFD_ConcurrentArmDisarm(t *testing.T) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		t.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	const numOps = 100
	var wg sync.WaitGroup

	// Concurrent armers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = tfd.ArmDuration(time.Hour, 0) // Long timer so it doesn't fire
			}
		}()
	}

	// Concurrent disarmers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				_ = tfd.Disarm()
			}
		}()
	}

	wg.Wait()

	// Timer should still be functional
	if !tfd.Valid() {
		t.Error("TimerFD should still be valid")
	}
}

// =============================================================================
// Additional Benchmarks
// =============================================================================

func BenchmarkEventFD_Wait(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = efd.Signal(1)
		_, _ = efd.Wait()
	}
}

func BenchmarkEventFD_WaitInto(b *testing.B) {
	efd, err := iofd.NewEventFD(0)
	if err != nil {
		b.Fatalf("NewEventFD failed: %v", err)
	}
	defer efd.Close()

	var val uint64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = efd.Signal(1)
		_ = efd.WaitInto(&val)
	}
}

func BenchmarkTimerFD_Arm(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = tfd.Arm(int64(time.Hour), 0)
	}
}

func BenchmarkTimerFD_Expirations(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm with very short timer
	_ = tfd.Arm(1, 1) // 1ns initial, 1ns interval

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = tfd.Expirations()
	}
}

func BenchmarkTimerFD_ExpirationsInto(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	// Arm with very short timer
	_ = tfd.Arm(1, 1) // 1ns initial, 1ns interval

	var count uint64

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = tfd.ExpirationsInto(&count)
	}
}

func BenchmarkTimerFD_GetTimeDuration(b *testing.B) {
	tfd, err := iofd.NewTimerFD()
	if err != nil {
		b.Fatalf("NewTimerFD failed: %v", err)
	}
	defer tfd.Close()

	_ = tfd.ArmDuration(time.Hour, time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = tfd.GetTimeDuration()
	}
}
