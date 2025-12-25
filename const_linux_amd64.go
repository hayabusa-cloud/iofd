// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux && amd64

package iofd

// Syscall numbers for Linux amd64.
const (
	SYS_DUP       = 32
	SYS_DUP2      = 33
	SYS_DUP3      = 292
	SYS_FCNTL     = 72
	SYS_FTRUNCATE = 77
	SYS_FSTAT     = 5
)
