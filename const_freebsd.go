// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build freebsd

package iofd

// Syscall numbers for FreeBSD.
// Reference: /usr/include/sys/syscall.h (FreeBSD 14.x)
const (
	SYS_DUP       = 41
	SYS_DUP2      = 90
	SYS_DUP3      = 0 // Not available; use fcntl F_DUPFD_CLOEXEC
	SYS_FCNTL     = 92
	SYS_FTRUNCATE = 480 // freebsd6_ftruncate
	SYS_FSTAT     = 551 // freebsd12_fstat
)

// File descriptor flags for fcntl F_GETFD/F_SETFD.
const (
	FD_CLOEXEC = 1
)

// File status flags for fcntl F_GETFL/F_SETFL.
const (
	O_NONBLOCK = 0x4
	O_CLOEXEC  = 0x100000
)

// fcntl commands.
const (
	F_DUPFD         = 0
	F_GETFD         = 1
	F_SETFD         = 2
	F_GETFL         = 3
	F_SETFL         = 4
	F_DUPFD_CLOEXEC = 17
)
