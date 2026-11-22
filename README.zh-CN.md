# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Benchmarks](https://github.com/hayabusa-cloud/iofd/actions/workflows/benchmarks.yml/badge.svg)](https://github.com/hayabusa-cloud/iofd/actions/workflows/benchmarks.yml)
[![Codecov](https://codecov.io/gh/hayabusa-cloud/iofd/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/iofd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go 语言的 Unix 系统通用文件描述符抽象。

语言: [English](./README.md) | **简体中文** | [Español](./README.es.md) | [日本語](./README.ja.md) | [Français](./README.fr.md)

## 概述

`iofd` 为 Go 生态系统提供最小化的文件描述符抽象和专用的 Linux 句柄。它作为高性能 I/O 系统的标准句柄抽象。

### 主要特性

- **零开销**: 所有内核交互通过 `zcall` 汇编，绕过 Go 的系统调用钩子
- **零分配热路径**: 固定大小的 `EventFD`、`TimerFD` 和 `SignalFD` 成功路径让系统调用参数保持在栈上
- **专用句柄**: Linux 特有的 `EventFD`、`TimerFD`、`PidFD`、`MemFD`、`SignalFD`
- **跨平台核心**: 基础 `FD` 操作支持 Linux、Darwin 和 FreeBSD
- **显式所有权**: `FD` 的关闭幂等性只适用于同一个描述符单元；使用方排空后再关闭，独立关闭所有者请使用 `Dup`

## 安装

```bash
go get code.hybscloud.com/iofd
```

## 快速开始

### EventFD 信号

```go
efd, _ := iofd.NewEventFD(0)
defer efd.Close()

efd.Signal(1)
val, _ := efd.Wait() // val == 1
```

### TimerFD

```go
tfd, _ := iofd.NewTimerFD()
defer tfd.Close()

// 100ms 一次性定时器
tfd.ArmDuration(100*time.Millisecond, 0)
// ... 等待 poll/epoll/io_uring ...
count, _ := tfd.Expirations() // count == 1
```

### 错误处理

```go
_, err := efd.Wait()
if errors.Is(err, iox.ErrWouldBlock) {
    // 非阻塞且暂无数据，稍后重试
} else if errors.Is(err, iofd.ErrClosed) {
    // FD 已关闭
} else if err != nil {
    // 其他错误
}
```

## API

### 核心类型

| 类型 | 描述 |
|------|------|
| `FD` | 具有同单元原子生命周期操作的文件描述符单元 |
| `EventFD` | 用于线程间信号传递的 Linux eventfd |
| `TimerFD` | 用于高精度定时器的 Linux timerfd |
| `PidFD` | 用于无竞争进程管理的 Linux pidfd |
| `MemFD` | 用于匿名内存文件的 Linux memfd |
| `MappedRegion` | 用于零拷贝访问的内存映射区域 |
| `SignalFD` | 用于同步信号处理的 Linux signalfd |

### 接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `PollFd` | `Fd() int` | 可轮询的文件描述符 |
| `PollCloser` | `Fd()`, `Close()` | 可关闭的可轮询描述符 |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | 完整 I/O 句柄 |
| `Signaler` | `Signal()`, `Wait()` | 信号机制 |
| `Timer` | `Arm()`, `Disarm()` | 定时器句柄 |

### FD 操作

这个函数接管描述符 `rawFd`。它指向一个以 `O_RDWR` 打开的临时文件，内容为 `ab`，当前偏移量为 0。

```go
fd := iofd.NewFD(rawFd)
defer fd.Close()

dup, err := fd.Dup() // 两个描述符可分别关闭，共享文件偏移量。
if err != nil {
	return 0, err
}
defer dup.Close()

var buf [1]byte
n, err := fd.Read(buf[:]) // 读出 'a'，共享偏移量变为 1。
if err != nil {
	return 0, err
}
return dup.Write(buf[:n]) // 覆盖 'b'，文件内容变为 "aa"。
```

如果 `rawFd` 是借用的，请让原描述符保持打开，直到 `Dup` 返回。用完后只关闭副本。

| 操作 | 效果 |
|-----------|------|
| `fd.Raw()` | 返回借用的描述符编号（`int32`）。使用该编号期间须保持 `fd` 打开。 |
| `fd.Valid()` | 检查保存的描述符编号是否非负。检查结束后，另一个 goroutine 仍可立即关闭 `fd`。 |
| `fd.SetNonblock(true)` | 设置 `O_NONBLOCK`，对 `fd` 和它的副本都生效。 |
| `fd.SetCloexec(true)` | 只为 `fd` 设置 `FD_CLOEXEC`。`Dup` 会自动为新描述符设置该标志。 |

### 构造函数标志

| 构造函数 | 创建时的标志 |
|----------|----------|
| `NewEventFD` | `EFD_NONBLOCK`, `EFD_CLOEXEC` |
| `NewEventFDSemaphore` | `EFD_SEMAPHORE`, `EFD_NONBLOCK`, `EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK`, `TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK`, `SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK` |
| `NewPidFDBlocking` | `0` |
| `NewMemFD` | `MFD_CLOEXEC` |
| `NewMemFDSealed` | `MFD_CLOEXEC`, `MFD_ALLOW_SEALING` |
| `NewMemFDHugeTLB` | `MFD_CLOEXEC`, `MFD_HUGETLB` |

即使标志参数为 `0`，`pidfd_open` 也会设置 `FD_CLOEXEC`。`memfd_create` 没有用于非阻塞 I/O 的创建标志。

### MemFD 内存映射

```go
// 创建 memfd 并设置大小
mfd, _ := iofd.NewMemFD("buffer")
mfd.Truncate(4096)

// 映射以实现零拷贝访问
region, _ := mfd.Mmap(4096, iofd.PROT_READ|iofd.PROT_WRITE)
data := region.Bytes()  // 共享内存支持的 []byte
copy(data, []byte("hello"))

// 清理
region.Unmap()
mfd.Close()
```

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                      应用层                              │
├─────────────────────────────────────────────────────────┤
│  EventFD │ TimerFD │ MemFD │ PidFD │ SignalFD │   FD   │
├─────────────────────────────────────────────────────────┤
│                        iofd                              │
├─────────────────────────────────────────────────────────┤
│                       zcall                              │
│                   (零开销系统调用)                        │
├─────────────────────────────────────────────────────────┤
│                     Linux 内核                           │
└─────────────────────────────────────────────────────────┘
```

## 平台支持

| 平台 | FD 核心 | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------|---------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**注意**: 专用句柄（`EventFD`、`TimerFD` 等）是 Linux 特有的内核原语。在 Darwin 和 FreeBSD 上，仅核心 `FD` 类型可用。

## 基准测试

在 Linux 上，`make bench` 使用 `-cpu=1` 为每个基准测试采集 10 个 200 毫秒的样本，并报告耗时和内存分配。使用 `make bench BENCH=EventFD` 可只运行 eventfd 测试。

比较两个版本时，请在同一台机器上使用相同的内核和 Go 版本，分别保存输出，再运行 `benchstat before.txt after.txt`。`BenchmarkEventFD_SignalWait` 测量一次发送信号和一次等待的总耗时。定时器到期读取基准测试用 `retries/op` 表示每次成功读取所需的重试次数。

## 安全注意事项

- **原子操作**: `Raw`、`Valid` 和同一 `FD` 单元上的 `Close` 使用原子访问；调用方仍需在 `Close()` 前排空使用方
- **所有权**: `Close()` 只对同一个 `FD` 单元幂等；复制出来的打开 `FD` 值不是独立所有者
- **关闭顺序**: 只有在进行中的操作和借用的 raw 描述符使用方都排空后才调用 `Close()`
- **描述符复制**: 需要另一个可关闭描述符时使用 `Dup()` 或 `PidFD.GetFD()`
- **MappedRegion 生命周期**: `Bytes()` 切片仅在区域映射期间有效

## 许可证

MIT — 参见 [LICENSE](./LICENSE)。

©2025 Hayabusa Cloud Co., Ltd.
