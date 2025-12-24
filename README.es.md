# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/iofd)](https://goreportcard.com/report/github.com/hayabusa-cloud/iofd)
[![Codecov](https://codecov.io/gh/hayabusa-cloud/iofd/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/iofd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Abstracciones universales de descriptores de archivo para sistemas Unix en Go.

Idioma: [English](./README.md) | [简体中文](./README.zh-CN.md) | **Español** | [日本語](./README.ja.md) | [Français](./README.fr.md)

## Descripción General

`iofd` proporciona abstracciones mínimas de descriptores de archivo y handles especializados de Linux para el ecosistema Go. Sirve como la abstracción canónica de handles para sistemas de E/S de alto rendimiento.

### Características Principales

- **Cero Sobrecarga**: Todas las interacciones con el kernel via ensamblador `zcall`, evitando los hooks de syscall de Go
- **Handles Especializados**: `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD` específicos de Linux
- **Núcleo Multiplataforma**: Las operaciones base de `FD` funcionan en Linux, Darwin y FreeBSD

## Instalación

```bash
go get code.hybscloud.com/iofd
```

## Inicio Rápido

```go
efd, _ := iofd.NewEventFD(0)
efd.Signal(1)
val, _ := efd.Wait() // val == 1
efd.Close()
```

## API

### Tipos Principales

| Tipo | Descripción |
|------|-------------|
| `FD` | Descriptor de archivo universal con operaciones atómicas |
| `EventFD` | eventfd de Linux para señalización entre hilos |
| `TimerFD` | timerfd de Linux para temporizadores de alta resolución |
| `PidFD` | pidfd de Linux para gestión de procesos sin condiciones de carrera |
| `MemFD` | memfd de Linux para archivos anónimos respaldados por memoria |
| `SignalFD` | signalfd de Linux para manejo síncrono de señales |

### Interfaces

| Interfaz | Métodos | Descripción |
|----------|---------|-------------|
| `PollFd` | `Fd() int` | Descriptor de archivo consultable |
| `PollCloser` | `Fd()`, `Close()` | Descriptor consultable cerrable |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Handle de E/S completo |
| `Signaler` | `Signal()`, `Wait()` | Mecanismo de señalización |
| `Timer` | `Arm()`, `Disarm()`, `Read()` | Handle de temporizador |

### Operaciones de FD

```go
// Crear FD desde descriptor raw
fd := iofd.NewFD(rawFd)

// Operaciones atómicas
fd.Raw()           // Obtener valor int32 raw
fd.Valid()         // Verificar si es válido (no negativo)
fd.Close()         // Cierre idempotente

// Operaciones de E/S
fd.Read(buf)       // Leer bytes
fd.Write(buf)      // Escribir bytes

// Flags del descriptor
fd.SetNonblock(true)   // Establecer O_NONBLOCK
fd.SetCloexec(true)    // Establecer FD_CLOEXEC
fd.Dup()               // Duplicar con CLOEXEC
```

## Soporte de Plataformas

| Plataforma | FD Núcleo | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------|-----------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Nota**: Los handles especializados (`EventFD`, `TimerFD`, etc.) son primitivas del kernel específicas de Linux. En Darwin y FreeBSD, solo el tipo `FD` núcleo está disponible.

## Licencia

MIT — ver [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
