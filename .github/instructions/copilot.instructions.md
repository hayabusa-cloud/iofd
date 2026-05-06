# iofd Review Instructions

## Scope

This file is a conservative review baseline for `code.hybscloud.com/iofd`.
Report only confirmed correctness, safety, resource-lifecycle, syscall-contract,
or public API contract bugs.

If a finding depends on speculation, style preference, broader refactoring, or
low-severity documentation wording, do not report it. Prefer no comment over a
noisy or uncertain comment.

## Report Format

Keep every report short and concrete:

- Severity: `P1` or `P2` only.
- Location: exact file and line.
- Failure: what breaks, and under which reachable condition.
- Fix: smallest concrete correction.
- Evidence: cite the exact code path, syscall contract, or public package
  contract.

Do not write overview-only comments, compliments, style notes, broad
suggestions, or speculative "consider" comments.

## Report Only

- Data races on shared descriptor state, signal masks, or mapped-region state.
- Use-after-close, double ownership, or close-before-drain that can execute
  under the package lifecycle contract.
- Missing `Close`, `Unmap`, or ownership release on a reachable error path.
- Buffer overrun, buffer underrun, nil pointer dereference, invalid unsafe
  pointer lifetime, or invalid slice lifetime.
- Wrong syscall number, syscall argument, creation flag, errno mapping, or build
  tag for a supported target.
- Public documentation that contradicts exported behavior or the runtime-tested
  support contract.

## Do Not Report

- Style, naming, formatting, translation wording, comment polish, or README
  phrasing unless it contradicts exported behavior.
- Missing extra defensive checks that only duplicate kernel or package
  preconditions.
- Suggestions to add dependencies, replace `zcall`, or route syscalls through
  Go's standard runtime path.
- Allocation concerns unless a reviewed hot path measurably allocates or an
  escape is visible in the implementation.
- Platform-support comments based only on compile-only CI targets.

## Accepted Low-Level Patterns

| Pattern | Reason |
|---------|--------|
| `atomic.SwapInt32((*int32)(fd), -1)` | Same-cell atomic close transition. |
| `atomic.LoadInt32((*int32)(fd))` | Same-cell atomic descriptor access. |
| `fd.Raw() < 0` before operations | Closed-descriptor sentinel check. |
| `iox.ErrWouldBlock` returns | Nonblocking control flow, not failure. |
| `//go:nosplit` on tiny accessors | Hot-path stack discipline. |
| `noCopy` marker fields | Vet-visible copied-owner guard. |
| `unsafe.Pointer(&nameBytes[0])` | Syscall string pointer with local lifetime. |
| `unsafe.Pointer(&stat)` | Syscall struct pointer with local lifetime. |
| `unsafe.Slice((*byte)(ptr), n)` | mmap-backed slice view while mapped. |
| `var _ [N]byte = [unsafe.Sizeof(...)]byte{}` | Compile-time size assertion. |
| `var _ [N]byte = [unsafe.Offsetof(...)]byte{}` | Compile-time offset assertion. |

## Package Contracts

- `FD.Close` is idempotent for one addressable `FD` cell.
- Copied open `FD` values are not independent close authorities.
- Raw descriptor numbers returned by `Fd` or `Raw` are borrowed. They must not be
  closed directly, and callers must drain their users before `Close`.
- Use `Dup` or `PidFD.GetFD` when an independent close-capable descriptor is
  required.
- All kernel calls go through `code.hybscloud.com/zcall`.
- Fixed-size success paths for `EventFD`, `TimerFD`, and `SignalFD` are expected
  to stay allocation-free.

## Platform Notes

- User-facing support tables list runtime-tested public targets.
- Linux `riscv64` and `loong64` may appear as compile-only CI targets without
  becoming public runtime support rows.
- Linux `getpid` and `gettid` are documented as always successful. Do not report
  missing errno checks for test-only uses of those calls unless a test target
  explicitly relies on syscall filtering. Report missing `tgkill` error handling
  if the fallible delivery syscall is unchecked.
