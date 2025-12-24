// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux

package iofd

// Syscall numbers for Linux.
// These are architecture-specific; values here are for amd64.
// Other architectures may require separate const_linux_<arch>.go files.
const (
	SYS_DUP       = 32
	SYS_DUP2      = 33
	SYS_DUP3      = 292
	SYS_FCNTL     = 72
	SYS_FTRUNCATE = 77
	SYS_FSTAT     = 5
)

// File descriptor flags for fcntl F_GETFD/F_SETFD.
const (
	FD_CLOEXEC = 1
)

// File status flags for fcntl F_GETFL/F_SETFL.
const (
	O_NONBLOCK = 0x800
	O_CLOEXEC  = 0x80000
)

// fcntl commands.
const (
	F_DUPFD         = 0
	F_GETFD         = 1
	F_SETFD         = 2
	F_GETFL         = 3
	F_SETFL         = 4
	F_DUPFD_CLOEXEC = 1030
)
