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
- **专用句柄**: Linux 特有的 `EventFD`、`TimerFD`、`PidFD`、`MemFD`、`SignalFD`
- **跨平台核心**: 基础 `FD` 操作支持 Linux、Darwin 和 FreeBSD

## 安装

```bash
go get code.hybscloud.com/iofd
```

## 快速开始

```go
efd, _ := iofd.NewEventFD(0)
efd.Signal(1)
val, _ := efd.Wait() // val == 1
efd.Close()
```

## API

### 核心类型

| 类型 | 描述 |
|------|------|
| `FD` | 具有原子操作的通用文件描述符 |
| `EventFD` | 用于线程间信号传递的 Linux eventfd |
| `TimerFD` | 用于高精度定时器的 Linux timerfd |
| `PidFD` | 用于无竞争进程管理的 Linux pidfd |
| `MemFD` | 用于匿名内存文件的 Linux memfd |
| `SignalFD` | 用于同步信号处理的 Linux signalfd |

### 接口

| 接口 | 方法 | 描述 |
|------|------|------|
| `PollFd` | `Fd() int` | 可轮询的文件描述符 |
| `PollCloser` | `Fd()`, `Close()` | 可关闭的可轮询描述符 |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | 完整 I/O 句柄 |
| `Signaler` | `Signal()`, `Wait()` | 信号机制 |
| `Timer` | `Arm()`, `Disarm()`, `Read()` | 定时器句柄 |

### FD 操作

```go
// 从原始描述符创建 FD
fd := iofd.NewFD(rawFd)

// 原子操作
fd.Raw()           // 获取原始 int32 值
fd.Valid()         // 检查是否有效（非负）
fd.Close()         // 幂等关闭

// I/O 操作
fd.Read(buf)       // 读取字节
fd.Write(buf)      // 写入字节

// 描述符标志
fd.SetNonblock(true)   // 设置 O_NONBLOCK
fd.SetCloexec(true)    // 设置 FD_CLOEXEC
fd.Dup()               // 带 CLOEXEC 复制
```

## 平台支持

| 平台 | FD 核心 | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------|---------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**注意**: 专用句柄（`EventFD`、`TimerFD` 等）是 Linux 特有的内核原语。在 Darwin 和 FreeBSD 上，仅核心 `FD` 类型可用。

## 许可证

MIT — 参见 [LICENSE](./LICENSE)。

©2025 Hayabusa Cloud Co., Ltd.
