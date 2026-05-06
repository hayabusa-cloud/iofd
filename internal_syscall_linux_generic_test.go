// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux && (arm64 || riscv64 || loong64)

package iofd

// Test-only syscall numbers for supported generic Linux syscall-table arches.
const (
	sysGetpid        = 172 // SYS_GETPID
	sysGettid        = 178 // SYS_GETTID
	sysRtSigprocmask = 135 // SYS_RT_SIGPROCMASK
	sysTgkill        = 131 // SYS_TGKILL
)
