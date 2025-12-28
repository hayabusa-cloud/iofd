// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build darwin

package iofd

// Syscall numbers for Darwin.
// Reference: /usr/include/sys/syscall.h (XNU kernel)
const (
	SYS_FCNTL = 92
)

// File descriptor flags for fcntl F_GETFD/F_SETFD.
const (
	FD_CLOEXEC = 1
)

// File status flags for fcntl F_GETFL/F_SETFL.
const (
	O_NONBLOCK = 0x4
	O_CLOEXEC  = 0x1000000
)

// fcntl commands.
const (
	F_DUPFD         = 0
	F_GETFD         = 1
	F_SETFD         = 2
	F_GETFL         = 3
	F_SETFL         = 4
	F_DUPFD_CLOEXEC = 67
)
