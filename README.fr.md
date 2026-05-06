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
- **Hot Paths sans Allocation**: Les chemins de succès à taille fixe de `EventFD`, `TimerFD` et `SignalFD` gardent les arguments syscall sur la pile
- **Handles Spécialisés**: `EventFD`, `TimerFD`, `PidFD`, `MemFD`, `SignalFD` spécifiques à Linux
- **Noyau Multiplateforme**: Les opérations de base `FD` fonctionnent sur Linux, Darwin et FreeBSD
- **Propriété Explicite**: L'idempotence de fermeture de `FD` s'applique à une seule cellule de descripteur; fermez après avoir drainé les utilisateurs et utilisez `Dup` pour un propriétaire de fermeture indépendant

## Installation

```bash
go get code.hybscloud.com/iofd
```

## Démarrage Rapide

### Signalisation EventFD

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

// Minuterie one-shot à 100ms
tfd.ArmDuration(100*time.Millisecond, 0)
// ... attendre avec poll/epoll/io_uring ...
count, _ := tfd.Expirations() // count == 1
```

### Gestion des Erreurs

```go
_, err := efd.Wait()
if errors.Is(err, iox.ErrWouldBlock) {
    // Non bloquant, aucune donnée disponible: réessayer plus tard
} else if errors.Is(err, iofd.ErrClosed) {
    // FD fermé
} else if err != nil {
    // Autre erreur
}
```

## API

### Types Principaux

| Type | Description |
|------|-------------|
| `FD` | Cellule de descripteur de fichier avec opérations atomiques de cycle de vie sur la même cellule |
| `EventFD` | eventfd Linux pour la signalisation inter-threads |
| `TimerFD` | timerfd Linux pour les minuteries haute résolution |
| `PidFD` | pidfd Linux pour la gestion de processus sans condition de course |
| `MemFD` | memfd Linux pour les fichiers anonymes en mémoire |
| `MappedRegion` | Région mémoire mappée pour accès zéro-copie |
| `SignalFD` | signalfd Linux pour le traitement synchrone des signaux |

### Interfaces

| Interface | Méthodes | Description |
|-----------|----------|-------------|
| `PollFd` | `Fd() int` | Descripteur de fichier interrogeable |
| `PollCloser` | `Fd()`, `Close()` | Descripteur interrogeable fermable |
| `Handle` | `Fd()`, `Close()`, `Read()`, `Write()` | Handle d'E/S complet |
| `Signaler` | `Signal()`, `Wait()` | Mécanisme de signalisation |
| `Timer` | `Arm()`, `Disarm()` | Handle de minuterie |

### Opérations FD

```go
// Créer FD depuis un descripteur brut
fd := iofd.NewFD(rawFd)
// NewFD prend la propriété de fermeture. Ne fermez pas les valeurs FD copiées;
// fermez seulement après avoir drainé les utilisateurs. Utilisez fd.Dup()
// pour un propriétaire de descripteur indépendant.

// Opérations atomiques
fd.Raw()           // Obtenir la valeur int32 brute
fd.Valid()         // Vérifier si valide (non négatif)
fd.Close()         // Fermeture même-cellule après drainage

// Opérations d'E/S
fd.Read(buf)       // Lire des octets
fd.Write(buf)      // Écrire des octets

// Drapeaux du descripteur
fd.SetNonblock(true)   // Définir O_NONBLOCK
fd.SetCloexec(true)    // Définir FD_CLOEXEC
fd.Dup()               // Dupliquer avec CLOEXEC
```

### Drapeaux des Constructeurs

| Constructeur | Drapeaux par défaut |
|--------------|---------------------|
| `NewEventFD`, `NewEventFDSemaphore` | `EFD_NONBLOCK | EFD_CLOEXEC` |
| `NewTimerFD`, `NewTimerFDRealtime`, `NewTimerFDBoottime` | `TFD_NONBLOCK | TFD_CLOEXEC` |
| `NewSignalFD` | `SFD_NONBLOCK | SFD_CLOEXEC` |
| `NewPidFD` | `PIDFD_NONBLOCK`; close-on-exec est défini par le kernel |
| `NewPidFDBlocking` | pidfd bloquant; close-on-exec reste défini par le kernel |
| `NewMemFD`, `NewMemFDSealed`, `NewMemFDHugeTLB` | `MFD_CLOEXEC` plus drapeaux propres à memfd; aucun drapeau nonblocking n'existe à la création |

### Mappage Mémoire MemFD

```go
// Créer memfd et définir la taille
mfd, _ := iofd.NewMemFD("buffer")
mfd.Truncate(4096)

// Mapper pour accès zéro-copie
region, _ := mfd.Mmap(4096, iofd.PROT_READ|iofd.PROT_WRITE)
data := region.Bytes()  // []byte soutenu par mémoire partagée
copy(data, []byte("hello"))

// Nettoyage
region.Unmap()
mfd.Close()
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Couche Application                     │
├─────────────────────────────────────────────────────────┤
│  EventFD │ TimerFD │ MemFD │ PidFD │ SignalFD │   FD   │
├─────────────────────────────────────────────────────────┤
│                        iofd                              │
├─────────────────────────────────────────────────────────┤
│                       zcall                              │
│              (syscalls zéro surcharge)                   │
├─────────────────────────────────────────────────────────┤
│                    Noyau Linux                           │
└─────────────────────────────────────────────────────────┘
```

## Support des Plateformes

| Plateforme | FD Noyau | EventFD | TimerFD | PidFD | MemFD | SignalFD |
|------------|----------|---------|---------|-------|-------|----------|
| Linux/amd64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Linux/arm64 | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Darwin/arm64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| FreeBSD/amd64 | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |

**Note**: Les handles spécialisés (`EventFD`, `TimerFD`, etc.) sont des primitives kernel spécifiques à Linux. Sur Darwin et FreeBSD, seul le type `FD` noyau est disponible.

## Considérations de Sécurité

- **Opérations Atomiques**: `Raw`, `Valid` et `Close` sur la même cellule `FD` utilisent un accès atomique; l'appelant doit toujours drainer les utilisateurs avant `Close()`
- **Propriété**: `Close()` est idempotent pour la même cellule `FD`; les valeurs `FD` ouvertes copiées ne sont pas des propriétaires indépendants
- **Ordre de Fermeture**: Appelez `Close()` seulement après avoir drainé les opérations en cours et les utilisateurs de descripteurs raw empruntés
- **Vérification de Validité**: Utilisez `Valid()` avant les opérations sur des descripteurs potentiellement fermés
- **Duplication**: Utilisez `Dup()` ou `PidFD.GetFD()` lorsqu'un autre descripteur fermable est requis
- **Durée de Vie de MappedRegion**: Le slice `Bytes()` n'est valide que pendant le mappage de la région

## Licence

MIT — voir [LICENSE](./LICENSE).

©2025 Hayabusa Cloud Co., Ltd.
