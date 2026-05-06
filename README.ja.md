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
- **ゼロアロケーションホットパス**: 固定サイズの`EventFD`、`TimerFD`、`SignalFD`成功パスではsyscall引数をスタック上に保持します
- **特殊ハンドル**: Linux固有の`EventFD`、`TimerFD`、`PidFD`、`MemFD`、`SignalFD`
- **クロスプラットフォームコア**: 基本`FD`操作はLinux、Darwin、FreeBSDで動作
- **明示的所有権**: `FD`のクローズ冪等性は同じディスクリプタセルにだけ適用されます。使用中の処理を排出してから閉じ、独立したクローズ所有者には`Dup`を使います

## インストール

```bash
go get code.hybscloud.com/iofd
```

## クイックスタート

### EventFDシグナリング

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

// 100msのワンショットタイマー
tfd.ArmDuration(100*time.Millisecond, 0)
// ... poll/epoll/io_uringで待機 ...
count, _ := tfd.Expirations() // count == 1
```

### エラー処理

```go
_, err := efd.Wait()
if errors.Is(err, iox.ErrWouldBlock) {
    // ノンブロッキングでデータなし。あとで再試行
} else if errors.Is(err, iofd.ErrClosed) {
    // FDは閉じられています
} else if err != nil {
    // その他のエラー
}
```

## API

### コア型

| 型 | 説明 |
|----|------|
| `FD` | 同一セルのアトミックライフサイクル操作を持つファイルディスクリプタセル |
| `EventFD` | スレッド間シグナリング用Linux eventfd |
| `TimerFD` | 高精度タイマー用Linux timerfd |
| `PidFD` | 競合のないプロセス管理用Linux pidfd |
| `MemFD` | 匿名メモリバックファイル用Linux memfd |
| `MappedRegion` | ゼロコピーアクセス用メモリマップ領域 |
| `SignalFD` | 同期シグナル処理用Linux signalfd |

### インターフェース

| インターフェース | メソッド | 説明 |
|------------------|----------|------|
| `PollFd` | `Fd() int` | ポーリング可能なファイルディスクリプタ |
| `PollCloser` | `Fd()`, `Close()` | クローズ可能なポーリングディスクリプタ |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | 完全I/Oハンドル |
| `Signaler` | `Signal()`, `Wait()` | シグナリング機構 |
| `Timer` | `Arm()`, `Disarm()` | タイマーハンドル |

### FD操作

```go
// 生ディスクリプタからFDを作成
fd := iofd.NewFD(rawFd)
// NewFDはクローズ所有権を受け取ります。コピーしたFD値を閉じないでください。
// 使用中の処理を排出してから閉じてください。独立したディスクリプタ
// 所有者が必要な場合はfd.Dup()を使います。

// アトミック操作
fd.Raw()           // 生int32値を取得
fd.Valid()         // 有効かチェック（非負）
fd.Close()         // 排出後に同じ FD セルを閉じる

// I/O操作
fd.Read(buf)       // バイト読み取り
fd.Write(buf)      // バイト書き込み

// ディスクリプタフラグ
fd.SetNonblock(true)   // O_NONBLOCKを設定
fd.SetCloexec(true)    // FD_CLOEXECを設定
fd.Dup()               // CLOEXECで複製
```

### コンストラクタフラグ

| コンストラクタ | デフォルトフラグ |
|----------------|------------------|
| `NewEventFD`, `NewEventFDSemaphore` | `EFD_NONBLOCK | EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK | TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK | SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK`。close-on-execはカーネルが設定 |
| `NewPidFDBlocking` | ブロッキングpidfd。close-on-execは引き続きカーネルが設定 |
| `NewMemFD`, `NewMemFDSealed`, `NewMemFDHugeTLB` | `MFD_CLOEXEC`とmemfd固有フラグ。作成時nonblockingフラグは存在しません |

### MemFDメモリマッピング

```go
// memfdを作成しサイズを設定
mfd, _ := iofd.NewMemFD("buffer")
mfd.Truncate(4096)

// ゼロコピーアクセス用にマップ
region, _ := mfd.Mmap(4096, iofd.PROT_READ|iofd.PROT_WRITE)
data := region.Bytes()  // 共有メモリでバックされた[]byte
copy(data, []byte("hello"))

// クリーンアップ
region.Unmap()
mfd.Close()
```

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                    アプリケーション層                     │
├─────────────────────────────────────────────────────────┤
│  EventFD │ TimerFD │ MemFD │ PidFD │ SignalFD │   FD   │
├─────────────────────────────────────────────────────────┤
│                        iofd                              │
├─────────────────────────────────────────────────────────┤
│                       zcall                              │
│               (ゼロオーバーヘッドsyscall)                 │
├─────────────────────────────────────────────────────────┤
│                    Linuxカーネル                         │
└─────────────────────────────────────────────────────────┘
```

## プラットフォームサポート

| プラットフォーム | FDコア | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------------|--------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**注意**: 特殊ハンドル（`EventFD`、`TimerFD`など）はLinux固有のカーネルプリミティブです。DarwinとFreeBSDでは、コア`FD`型のみ利用可能です。

## 安全性に関する考慮事項

- **アトミック操作**: `Raw`、`Valid`、同じ`FD`セルの`Close`はアトミックアクセスを使います。呼び出し側は`Close()`前に使用者を排出する必要があります
- **所有権**: `Close()`は同じ`FD`セルに対してのみ冪等です。コピーされた開いた`FD`値は独立所有者ではありません
- **クローズ順序**: 進行中の操作と借用されたrawディスクリプタ使用者を排出してから`Close()`を呼び出してください
- **有効性チェック**: 閉じられた可能性のあるディスクリプタに対する操作前に`Valid()`を使用
- **複製**: 別のクローズ可能ディスクリプタが必要な場合は`Dup()`または`PidFD.GetFD()`を使います
- **MappedRegionのライフタイム**: `Bytes()`スライスは領域がマップされている間のみ有効

## ライセンス

MIT — [LICENSE](./LICENSE)を参照。

©2025 Hayabusa Cloud Co., Ltd.
