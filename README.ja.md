# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/iofd)](https://goreportcard.com/report/github.com/hayabusa-cloud/iofd)
[![Codecov](https://codecov.io/gh/hayabusa-cloud/iofd/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/iofd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go言語向けUnixシステム用汎用ファイルディスクリプタ抽象化。

言語: [English](./README.md) | [简体中文](./README.zh-CN.md) | [Español](./README.es.md) | **日本語** | [Français](./README.fr.md)

## 概要

`iofd`はGoエコシステム向けに最小限のファイルディスクリプタ抽象化と特殊なLinuxハンドルを提供します。高性能I/Oシステムの標準ハンドル抽象化として機能します。

### 主な特徴

- **ゼロオーバーヘッド**: `zcall`アセンブリによる全カーネル操作、Goのsyscallフックをバイパス
- **特殊ハンドル**: Linux固有の`EventFD`、`TimerFD`、`PidFD`、`MemFD`、`SignalFD`
- **クロスプラットフォームコア**: 基本`FD`操作はLinux、Darwin、FreeBSDで動作

## インストール

```bash
go get code.hybscloud.com/iofd
```

## クイックスタート

```go
efd, _ := iofd.NewEventFD(0)
efd.Signal(1)
val, _ := efd.Wait() // val == 1
efd.Close()
```

## API

### コア型

| 型 | 説明 |
|----|------|
| `FD` | アトミック操作を持つ汎用ファイルディスクリプタ |
| `EventFD` | スレッド間シグナリング用Linux eventfd |
| `TimerFD` | 高精度タイマー用Linux timerfd |
| `PidFD` | 競合のないプロセス管理用Linux pidfd |
| `MemFD` | 匿名メモリバックファイル用Linux memfd |
| `SignalFD` | 同期シグナル処理用Linux signalfd |

### インターフェース

| インターフェース | メソッド | 説明 |
|------------------|----------|------|
| `PollFd` | `Fd() int` | ポーリング可能なファイルディスクリプタ |
| `PollCloser` | `Fd()`, `Close()` | クローズ可能なポーリングディスクリプタ |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | 完全I/Oハンドル |
| `Signaler` | `Signal()`, `Wait()` | シグナリング機構 |
| `Timer` | `Arm()`, `Disarm()`, `Read()` | タイマーハンドル |

### FD操作

```go
// 生ディスクリプタからFDを作成
fd := iofd.NewFD(rawFd)

// アトミック操作
fd.Raw()           // 生int32値を取得
fd.Valid()         // 有効かチェック（非負）
fd.Close()         // 冪等クローズ

// I/O操作
fd.Read(buf)       // バイト読み取り
fd.Write(buf)      // バイト書き込み

// ディスクリプタフラグ
fd.SetNonblock(true)   // O_NONBLOCKを設定
fd.SetCloexec(true)    // FD_CLOEXECを設定
fd.Dup()               // CLOEXECで複製
```

## プラットフォームサポート

| プラットフォーム | FDコア | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------------|--------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**注意**: 特殊ハンドル（`EventFD`、`TimerFD`など）はLinux固有のカーネルプリミティブです。DarwinとFreeBSDでは、コア`FD`型のみ利用可能です。

## ライセンス

MIT — [LICENSE](./LICENSE)を参照。

©2025 Hayabusa Cloud Co., Ltd.
