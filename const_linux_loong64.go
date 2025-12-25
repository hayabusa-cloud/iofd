// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux && loong64

package iofd

// Syscall numbers for Linux loong64 (uses generic syscall table).
const (
	SYS_DUP       = 23
	SYS_DUP2      = 0 // Not available; use fcntl F_DUPFD
	SYS_DUP3      = 24
	SYS_FCNTL     = 25
	SYS_FTRUNCATE = 46
	SYS_FSTAT     = 80
)
