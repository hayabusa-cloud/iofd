// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

import (
	"unsafe"

	"code.hybscloud.com/zcall"
)

// MemFD represents a Linux memfd file descriptor.
// It provides an anonymous memory-backed file that can be used for:
//   - Inter-process communication via file descriptor passing
//   - Memory mapping without filesystem overhead
//   - Sealing to prevent modifications
//
// MemFD is created with MFD_CLOEXEC by default.
//
// Invariants:
//   - The file exists only in memory; it has no filesystem presence.
//   - Size starts at 0; use Truncate to set the desired size before use.
//   - Content is zeroed on allocation.
type MemFD struct {
	fd   FD
	name string
}

// NewMemFD creates a new memfd with the given name.
// The name is used for debugging (visible in /proc/[pid]/fd/).
// The memfd is created with MFD_CLOEXEC flag.
//
// The file is initially empty; call Truncate to set its size.
func NewMemFD(name string) (*MemFD, error) {
	return newMemFD(name, MFD_CLOEXEC)
}

// NewMemFDSealed creates a new memfd that allows sealing operations.
// Use this when you need to apply seals to prevent modifications.
func NewMemFDSealed(name string) (*MemFD, error) {
	return newMemFD(name, MFD_CLOEXEC|MFD_ALLOW_SEALING)
}

// NewMemFDHugeTLB creates a new memfd backed by huge pages.
// This can improve performance for large memory mappings.
// The size must be a multiple of the huge page size.
func NewMemFDHugeTLB(name string) (*MemFD, error) {
	return newMemFD(name, MFD_CLOEXEC|MFD_HUGETLB)
}

func newMemFD(name string, flags uintptr) (*MemFD, error) {
	// The name must be null-terminated for the syscall
	nameBytes := make([]byte, len(name)+1)
	copy(nameBytes, name)
	nameBytes[len(name)] = 0

	fd, errno := zcall.MemfdCreate(
		unsafe.Pointer(&nameBytes[0]),
		flags,
	)
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &MemFD{fd: FD(fd), name: name}, nil
}

// Fd returns the underlying file descriptor.
// Implements PollFd interface.
//
//go:nosplit
func (m *MemFD) Fd() int {
	return m.fd.Fd()
}

// Close closes the memfd.
// The memory is freed when all references (including mmaps) are released.
// Implements PollCloser interface.
func (m *MemFD) Close() error {
	return m.fd.Close()
}

// Raw returns the raw file descriptor for use in tight loops.
// The caller must ensure the MemFD remains valid while using the raw fd.
//
//go:nosplit
func (m *MemFD) Raw() int32 {
	return m.fd.Raw()
}

// Name returns the name given at creation.
func (m *MemFD) Name() string {
	return m.name
}

// Read reads from the memfd at the current file offset.
func (m *MemFD) Read(p []byte) (int, error) {
	return m.fd.Read(p)
}

// Write writes to the memfd at the current file offset.
func (m *MemFD) Write(p []byte) (int, error) {
	return m.fd.Write(p)
}

// Pread reads from the memfd at the specified offset without changing the file position.
func (m *MemFD) Pread(p []byte, offset int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	raw := m.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	iov := zcall.Iovec{Base: &p[0], Len: uint64(len(p))}
	n, errno := zcall.Preadv(uintptr(raw), unsafe.Pointer(&iov), 1, offset)
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// Pwrite writes to the memfd at the specified offset without changing the file position.
func (m *MemFD) Pwrite(p []byte, offset int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	raw := m.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	iov := zcall.Iovec{Base: &p[0], Len: uint64(len(p))}
	n, errno := zcall.Pwritev(uintptr(raw), unsafe.Pointer(&iov), 1, offset)
	if errno != 0 {
		return int(n), errFromErrno(errno)
	}
	return int(n), nil
}

// Truncate sets the size of the memfd.
// If the new size is larger, the extended area is zero-filled.
// If smaller, data beyond the new size is discarded.
func (m *MemFD) Truncate(size int64) error {
	raw := m.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	_, errno := zcall.Syscall4(zcall.SYS_FTRUNCATE, uintptr(raw), uintptr(size), 0, 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// Size returns the current size of the memfd.
func (m *MemFD) Size() (int64, error) {
	raw := m.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	var stat statBuf
	_, errno := zcall.Syscall4(zcall.SYS_FSTAT, uintptr(raw), uintptr(unsafe.Pointer(&stat)), 0, 0)
	if errno != 0 {
		return 0, errFromErrno(errno)
	}
	return stat.size, nil
}

// statBuf is a minimal struct stat for extracting file size.
// Layout matches Linux struct stat on 64-bit architectures (amd64, arm64, riscv64, loong64).
type statBuf struct {
	_    [48]byte // fields before st_size
	size int64    // st_size at offset 48
	_    [88]byte // remaining fields
}

// Compile-time size and offset checks for statBuf.
// Linux struct stat is 144 bytes on 64-bit architectures.
// st_size is at offset 48 on amd64, arm64, riscv64, loong64.
var _ [144]byte = [unsafe.Sizeof(statBuf{})]byte{}
var _ [48]byte = [unsafe.Offsetof(statBuf{}.size)]byte{}

// Seal applies seals to prevent certain operations.
// This is only available if the memfd was created with MFD_ALLOW_SEALING.
//
// Once a seal is applied, it cannot be removed.
func (m *MemFD) Seal(seals uint) error {
	raw := m.fd.Raw()
	if raw < 0 {
		return ErrClosed
	}
	_, errno := zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_ADD_SEALS, uintptr(seals), 0)
	if errno != 0 {
		return errFromErrno(errno)
	}
	return nil
}

// Seals returns the current seal flags.
func (m *MemFD) Seals() (uint, error) {
	raw := m.fd.Raw()
	if raw < 0 {
		return 0, ErrClosed
	}
	seals, errno := zcall.Syscall4(SYS_FCNTL, uintptr(raw), F_GET_SEALS, 0, 0)
	if errno != 0 {
		return 0, errFromErrno(errno)
	}
	return uint(seals), nil
}

// MappedRegion represents a memory-mapped region of a MemFD.
// The region must be unmapped with Unmap when no longer needed.
type MappedRegion struct {
	ptr    unsafe.Pointer
	length uintptr
}

// Ptr returns the base pointer of the mapped region.
//
//go:nosplit
func (r *MappedRegion) Ptr() unsafe.Pointer {
	return r.ptr
}

// Len returns the length of the mapped region in bytes.
//
//go:nosplit
func (r *MappedRegion) Len() int {
	return int(r.length)
}

// Bytes returns the mapped region as a byte slice.
// The returned slice is only valid while the region is mapped.
func (r *MappedRegion) Bytes() []byte {
	return unsafe.Slice((*byte)(r.ptr), r.length)
}

// Unmap unmaps the memory region.
// After this call, the region must not be accessed.
func (r *MappedRegion) Unmap() error {
	if r.ptr == nil {
		return nil
	}
	errno := zcall.Munmap(r.ptr, r.length)
	if errno != 0 {
		return errFromErrno(errno)
	}
	r.ptr = nil
	r.length = 0
	return nil
}

// Mmap maps the memfd into memory for zero-copy access.
// The mapping starts at offset 0 and spans the specified length.
//
// prot specifies the memory protection: PROT_READ, PROT_WRITE, or both.
// The memfd must be sized appropriately before mapping (see Truncate).
//
// The returned MappedRegion must be unmapped with Unmap when done.
// The MemFD can be closed after mapping; the region remains valid.
func (m *MemFD) Mmap(length int, prot int) (*MappedRegion, error) {
	return m.MmapAt(length, prot, 0)
}

// MmapAt maps the memfd into memory starting at the given offset.
// offset must be page-aligned (typically 4096 bytes).
func (m *MemFD) MmapAt(length int, prot int, offset int64) (*MappedRegion, error) {
	raw := m.fd.Raw()
	if raw < 0 {
		return nil, ErrClosed
	}
	if length <= 0 {
		return nil, ErrInvalidParam
	}
	ptr, errno := zcall.Mmap(nil, uintptr(length), uintptr(prot),
		MAP_SHARED, uintptr(raw), uintptr(offset))
	if errno != 0 {
		return nil, errFromErrno(errno)
	}
	return &MappedRegion{ptr: ptr, length: uintptr(length)}, nil
}

// Valid reports whether the memfd is still valid.
//
//go:nosplit
func (m *MemFD) Valid() bool {
	return m.fd.Valid()
}

// memfd flags
const (
	MFD_CLOEXEC       = 0x1
	MFD_ALLOW_SEALING = 0x2
	MFD_HUGETLB       = 0x4
	MFD_NOEXEC_SEAL   = 0x8
	MFD_EXEC          = 0x10
)

// Seal types for memfd
const (
	F_SEAL_SEAL         = 0x1  // Prevent further seals
	F_SEAL_SHRINK       = 0x2  // Prevent shrinking
	F_SEAL_GROW         = 0x4  // Prevent growing
	F_SEAL_WRITE        = 0x8  // Prevent writes
	F_SEAL_FUTURE_WRITE = 0x10 // Prevent future writes (allows current mappings)
)

// fcntl commands for sealing
const (
	F_ADD_SEALS = 1033
	F_GET_SEALS = 1034
)

// Memory protection flags for Mmap.
const (
	PROT_NONE  = 0x0
	PROT_READ  = 0x1
	PROT_WRITE = 0x2
	PROT_EXEC  = 0x4
)

// Memory mapping flags.
const (
	MAP_SHARED  = 0x1
	MAP_PRIVATE = 0x2
)

// Compile-time interface assertions
var (
	_ PollFd     = (*MemFD)(nil)
	_ PollCloser = (*MemFD)(nil)
	_ Handle     = (*MemFD)(nil)
)
