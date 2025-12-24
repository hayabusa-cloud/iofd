// ©Hayabusa Cloud Co., Ltd. 2025. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package iofd

import "errors"

// Error definitions for iofd operations.
// These errors provide semantic meaning for common file descriptor failures.
var (
	// ErrClosed indicates the file descriptor has been closed.
	ErrClosed = errors.New("fd: file descriptor closed")

	// ErrInvalidParam indicates an invalid parameter was passed.
	ErrInvalidParam = errors.New("fd: invalid parameter")

	// ErrInterrupted indicates the operation was interrupted by a signal.
	ErrInterrupted = errors.New("fd: interrupted")

	// ErrNoMemory indicates insufficient memory for the operation.
	ErrNoMemory = errors.New("fd: no memory")

	// ErrPermission indicates permission denied.
	ErrPermission = errors.New("fd: permission denied")

	// ErrOverflow indicates a counter overflow (for eventfd).
	ErrOverflow = errors.New("fd: counter overflow")
)
