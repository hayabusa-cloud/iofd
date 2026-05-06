# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/iofd)](https://goreportcard.com/report/github.com/hayabusa-cloud/iofd)
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

```go
// 从原始描述符创建 FD
fd := iofd.NewFD(rawFd)
// NewFD 接收关闭所有权。不要关闭复制出来的 FD 值；
// 等待使用方排空后再关闭。需要独立描述符所有者时请使用 fd.Dup()。

// 原子操作
fd.Raw()           // 获取原始 int32 值
fd.Valid()         // 检查是否有效（非负）
fd.Close()         // 排空后关闭同一 FD 单元

// I/O 操作
fd.Read(buf)       // 读取字节
fd.Write(buf)      // 写入字节

// 描述符标志
fd.SetNonblock(true)   // 设置 O_NONBLOCK
fd.SetCloexec(true)    // 设置 FD_CLOEXEC
fd.Dup()               // 带 CLOEXEC 复制
```

### 构造函数标志

| 构造函数 | 默认标志 |
|----------|----------|
| `NewEventFD`, `NewEventFDSemaphore` | `EFD_NONBLOCK | EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK | TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK | SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK`；close-on-exec 由内核设置 |
| `NewPidFDBlocking` | 阻塞 pidfd；close-on-exec 仍由内核设置 |
| `NewMemFD`, `NewMemFDSealed`, `NewMemFDHugeTLB` | `MFD_CLOEXEC` 加 memfd 专用标志；不存在创建时 nonblocking 标志 |

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

## 安全注意事项

- **原子操作**: `Raw`、`Valid` 和同一 `FD` 单元上的 `Close` 使用原子访问；调用方仍需在 `Close()` 前排空使用方
- **所有权**: `Close()` 只对同一个 `FD` 单元幂等；复制出来的打开 `FD` 值不是独立所有者
- **关闭顺序**: 只有在进行中的操作和借用的 raw 描述符使用方都排空后才调用 `Close()`
- **有效性检查**: 在操作可能已关闭的描述符前使用 `Valid()`
- **描述符复制**: 需要另一个可关闭描述符时使用 `Dup()` 或 `PidFD.GetFD()`
- **MappedRegion 生命周期**: `Bytes()` 切片仅在区域映射期间有效

## 许可证

MIT — 参见 [LICENSE](./LICENSE)。

©2025 Hayabusa Cloud Co., Ltd.
