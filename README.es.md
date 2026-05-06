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
- **Hot Paths sin Asignaciones**: Las rutas exitosas de tamaño fijo de `EventFD`, `TimerFD` y `SignalFD` mantienen los argumentos de syscall en la pila
- **Handles Especializados**: `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD` específicos de Linux
- **Núcleo Multiplataforma**: Las operaciones base de `FD` funcionan en Linux, Darwin y FreeBSD
- **Propiedad Explícita**: La idempotencia de cierre de `FD` aplica a una sola celda de descriptor; cierre después de drenar los usuarios y use `Dup` para un propietario de cierre independiente

## Instalación

```bash
go get code.hybscloud.com/iofd
```

## Inicio Rápido

### Señalización EventFD

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

// Temporizador one-shot a 100ms
tfd.ArmDuration(100*time.Millisecond, 0)
// ... esperar con poll/epoll/io_uring ...
count, _ := tfd.Expirations() // count == 1
```

### Manejo de Errores

```go
_, err := efd.Wait()
if errors.Is(err, iox.ErrWouldBlock) {
    // No bloqueante, sin datos disponibles: reintentar luego
} else if errors.Is(err, iofd.ErrClosed) {
    // FD cerrado
} else if err != nil {
    // Otro error
}
```

## API

### Tipos Principales

| Tipo | Descripción |
|------|-------------|
| `FD` | Celda de descriptor de archivo con operaciones atómicas de ciclo de vida en la misma celda |
| `EventFD` | eventfd de Linux para señalización entre hilos |
| `TimerFD` | timerfd de Linux para temporizadores de alta resolución |
| `PidFD` | pidfd de Linux para gestión de procesos sin condiciones de carrera |
| `MemFD` | memfd de Linux para archivos anónimos respaldados por memoria |
| `MappedRegion` | Región de memoria mapeada para acceso sin copia |
| `SignalFD` | signalfd de Linux para manejo síncrono de señales |

### Interfaces

| Interfaz | Métodos | Descripción |
|----------|---------|-------------|
| `PollFd` | `Fd() int` | Descriptor de archivo consultable |
| `PollCloser` | `Fd()`, `Close()` | Descriptor consultable cerrable |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Handle de E/S completo |
| `Signaler` | `Signal()`, `Wait()` | Mecanismo de señalización |
| `Timer` | `Arm()`, `Disarm()` | Handle de temporizador |

### Operaciones de FD

```go
// Crear FD desde descriptor raw
fd := iofd.NewFD(rawFd)
// NewFD toma la propiedad de cierre. No cierre valores FD copiados;
// cierre solo después de drenar los usuarios. Use fd.Dup() para un
// propietario de descriptor independiente.

// Operaciones atómicas
fd.Raw()           // Obtener valor int32 raw
fd.Valid()         // Verificar si es válido (no negativo)
fd.Close()         // Cierre de la misma celda tras drenar

// Operaciones de E/S
fd.Read(buf)       // Leer bytes
fd.Write(buf)      // Escribir bytes

// Flags del descriptor
fd.SetNonblock(true)   // Establecer O_NONBLOCK
fd.SetCloexec(true)    // Establecer FD_CLOEXEC
fd.Dup()               // Duplicar con CLOEXEC
```

### Flags de Constructores

| Constructor | Flags predeterminados |
|-------------|-----------------------|
| `NewEventFD`, `NewEventFDSemaphore` | `EFD_NONBLOCK | EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK | TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK | SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK`; close-on-exec lo establece el kernel |
| `NewPidFDBlocking` | pidfd bloqueante; close-on-exec también lo establece el kernel |
| `NewMemFD`, `NewMemFDSealed`, `NewMemFDHugeTLB` | `MFD_CLOEXEC` más flags específicos de memfd; no existe flag nonblocking en la creación |

### Mapeo de Memoria MemFD

```go
// Crear memfd y establecer tamaño
mfd, _ := iofd.NewMemFD("buffer")
mfd.Truncate(4096)

// Mapear para acceso sin copia
region, _ := mfd.Mmap(4096, iofd.PROT_READ|iofd.PROT_WRITE)
data := region.Bytes()  // []byte respaldado por memoria compartida
copy(data, []byte("hello"))

// Limpieza
region.Unmap()
mfd.Close()
```

## Arquitectura

```
┌─────────────────────────────────────────────────────────┐
│                   Capa de Aplicación                     │
├─────────────────────────────────────────────────────────┤
│  EventFD │ TimerFD │ MemFD │ PidFD │ SignalFD │   FD   │
├─────────────────────────────────────────────────────────┤
│                        iofd                              │
├─────────────────────────────────────────────────────────┤
│                       zcall                              │
│              (syscalls de cero sobrecarga)               │
├─────────────────────────────────────────────────────────┤
│                    Kernel Linux                          │
└─────────────────────────────────────────────────────────┘
```

## Soporte de Plataformas

| Plataforma | FD Núcleo | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------|-----------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Nota**: Los handles especializados (`EventFD`, `TimerFD`, etc.) son primitivas del kernel específicas de Linux. En Darwin y FreeBSD, solo el tipo `FD` núcleo está disponible.

## Consideraciones de Seguridad

- **Operaciones Atómicas**: `Raw`, `Valid` y `Close` en la misma celda `FD` usan acceso atómico; el llamador aún debe drenar usuarios antes de `Close()`
- **Propiedad**: `Close()` es idempotente para la misma celda `FD`; los valores `FD` abiertos y copiados no son propietarios independientes
- **Orden de Cierre**: Llame `Close()` solo después de drenar las operaciones en curso y los usuarios de descriptores raw prestados
- **Verificación de Validez**: Use `Valid()` antes de operaciones en descriptores potencialmente cerrados
- **Duplicación**: Use `Dup()` o `PidFD.GetFD()` cuando necesite otro descriptor cerrable
- **Vida Útil de MappedRegion**: El slice `Bytes()` solo es válido mientras la región esté mapeada

## Licencia

MIT — ver [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
