# iofd — Copilot Instructions

## Philosophy

**Last line of defense.** Report only truly severe bugs. No style, no suggestions.

## Correct Patterns (do NOT flag)

| Pattern | Reason |
|---------|--------|
| `atomic.SwapInt32((*int32)(fd), -1)` | Intentional atomic FD close |
| `atomic.LoadInt32((*int32)(fd))` | Intentional atomic FD access |
| `//go:nosplit` on accessors | Required for hot paths |
| `unsafe.Pointer(&nameBytes[0])` | Required for syscall strings |
| `unsafe.Pointer(&stat)` | Required for syscall structs |
| `unsafe.Slice((*byte)(ptr), n)` | Standard mmap slice creation |
| `var _ [144]byte = [unsafe.Sizeof(...)]byte{}` | Compile-time size assertion |
| `var _ [48]byte = [unsafe.Offsetof(...)]byte{}` | Compile-time offset assertion |

## Error Handling

- `errFromErrno()` converts `zcall.Errno` to semantic errors
- `iox.ErrWouldBlock` is control flow, not failure
- `ErrClosed` returned when `fd.Raw() < 0`

## Only Report

- Data races (missing atomic on shared state)
- Use-after-close (accessing FD after Close)
- Resource leaks (missing Close on error paths)
- Buffer overflows
- Null pointer dereference
