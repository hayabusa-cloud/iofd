// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux && amd64

package iofd

// Test-only syscall numbers not exported by zcall on Linux amd64.
const (
	sysGetpid        = 39  // SYS_GETPID
	sysGettid        = 186 // SYS_GETTID
	sysRtSigprocmask = 14  // SYS_RT_SIGPROCMASK
	sysTgkill        = 234 // SYS_TGKILL
)
