// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

//go:build linux && loong64

package iofd

// Syscall numbers for Linux loong64 (uses generic syscall table).
const (
	SYS_FCNTL = 25
)
