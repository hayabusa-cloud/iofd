# iofd

[![Go Reference](https://pkg.go.dev/badge/code.hybscloud.com/iofd.svg)](https://pkg.go.dev/code.hybscloud.com/iofd)
[![Go Report Card](https://goreportcard.com/badge/github.com/hayabusa-cloud/iofd)](https://goreportcard.com/report/github.com/hayabusa-cloud/iofd)
[![Codecov](https://codecov.io/gh/hayabusa-cloud/iofd/graph/badge.svg)](https://codecov.io/gh/hayabusa-cloud/iofd)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Abstractions universelles de descripteurs de fichiers pour systèmes Unix en Go.

Langue: [English](./README.md) | [简体中文](./README.zh-CN.md) | [Español](./README.es.md) | [日本語](./README.ja.md) | **Français**

## Aperçu

`iofd` fournit des abstractions minimales de descripteurs de fichiers et des handles Linux spécialisés pour l'écosystème Go. Il sert d'abstraction canonique de handles pour les systèmes d'E/S haute performance.

### Caractéristiques Principales

- **Zéro Surcharge**: Toutes les interactions kernel via assembleur `zcall`, contournant les hooks syscall de Go
- **Handles Spécialisés**: `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD` spécifiques à Linux
- **Noyau Multiplateforme**: Les opérations de base `FD` fonctionnent sur Linux, Darwin et FreeBSD

## Installation

```bash
go get code.hybscloud.com/iofd
```

## Démarrage Rapide

```go
efd, _ := iofd.NewEventFD(0)
efd.Signal(1)
val, _ := efd.Wait() // val == 1
efd.Close()
```

## API

### Types Principaux

| Type | Description |
|------|-------------|
| `FD` | Descripteur de fichier universel avec opérations atomiques |
| `EventFD` | eventfd Linux pour la signalisation inter-threads |
| `TimerFD` | timerfd Linux pour les minuteries haute résolution |
| `PidFD` | pidfd Linux pour la gestion de processus sans condition de course |
| `MemFD` | memfd Linux pour les fichiers anonymes en mémoire |
| `SignalFD` | signalfd Linux pour le traitement synchrone des signaux |

### Interfaces

| Interface | Méthodes | Description |
|-----------|----------|-------------|
| `PollFd` | `Fd() int` | Descripteur de fichier interrogeable |
| `PollCloser` | `Fd()`, `Close()` | Descripteur interrogeable fermable |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Handle d'E/S complet |
| `Signaler` | `Signal()`, `Wait()` | Mécanisme de signalisation |
| `Timer` | `Arm()`, `Disarm()`, `Read()` | Handle de minuterie |

### Opérations FD

```go
// Créer FD depuis un descripteur brut
fd := iofd.NewFD(rawFd)

// Opérations atomiques
fd.Raw()           // Obtenir la valeur int32 brute
fd.Valid()         // Vérifier si valide (non négatif)
fd.Close()         // Fermeture idempotente

// Opérations d'E/S
fd.Read(buf)       // Lire des octets
fd.Write(buf)      // Écrire des octets

// Drapeaux du descripteur
fd.SetNonblock(true)   // Définir O_NONBLOCK
fd.SetCloexec(true)    // Définir FD_CLOEXEC
fd.Dup()               // Dupliquer avec CLOEXEC
```

## Support des Plateformes

| Plateforme | FD Noyau | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------|----------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Note**: Les handles spécialisés (`EventFD`, `TimerFD`, etc.) sont des primitives kernel spécifiques à Linux. Sur Darwin et FreeBSD, seul le type `FD` noyau est disponible.

## Licence

MIT — voir [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
