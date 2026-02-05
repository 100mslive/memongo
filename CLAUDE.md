# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Memongo is a Go package that spins up a real MongoDB server backed by in-memory storage for use in testing. It downloads and caches official MongoDB binaries, manages the mongod process lifecycle, and provides a simple API for tests.

**Module path:** `github.com/100mslive/memongo/v2`

## Build and Test Commands

```bash
# Run all tests with race detection and coverage
./scripts/runUnitTests.sh

# Or directly
go test ./... -cover -race

# Run a single test
go test -run TestMemongo -v

# Lint (requires golangci-lint)
golangci-lint run
```

## Architecture

### Core Components

- **memongo.go** - Main `Server` struct and `Start`/`StartWithOptions` entry points. Manages mongod process startup, port assignment, and cleanup.

- **config.go** - `Options` struct for configuring MongoDB version, replica sets, authentication, ports, cache paths, logging, and memory limits.

- **mongobin/** - Handles MongoDB binary downloads:
  - `downloadSpec.go` - Generates version/platform/arch specifications
  - `downloadURL.go` - Constructs download URLs for different platforms
  - `getOrDownload.go` - Caching logic, downloads binaries only when not cached

- **monitor/** - Process watcher that spawns a shell subprocess to monitor the parent process and kill mongod if the parent exits abnormally (prevents zombie processes).

- **memongolog/** - Custom logger with four levels: Debug, Info, Warn, Silent.

### Server Methods

- `Port()` - Returns the port the server is listening on
- `URI()` - Returns mongodb:// URI to connect to
- `URIWithRandomDB()` - Returns URI with random database name for test isolation
- `Stop()` - Kills the mongo server and cleans up
- `Ping(ctx)` - Healthcheck that returns nil if server is responsive
- `IsReplicaSet()` - Returns true if started as replica set
- `ReplicaSetName()` - Returns replica set name (empty if not a replica set)
- `DBPath()` - Returns path to database directory (for diagnostics)

### Configuration Options

```go
type Options struct {
    MongoVersion          string        // Required: e.g., "8.0.0"
    ShouldUseReplica      bool          // Enable replica set mode
    ReplicaSetName        string        // Custom replica set name (default: "rs0")
    Auth                  bool          // Enable authentication
    Port                  int           // Custom port (0 = auto)
    CachePath             string        // Binary cache location
    DownloadURL           string        // Custom MongoDB download URL
    MongodBin             string        // Path to pre-downloaded mongod
    LogLevel              LogLevel      // Debug, Info, Warn, Silent
    StartupTimeout        time.Duration // Default: 10s
    WiredTigerCacheSizeGB float64       // Memory limit for WiredTiger (e.g., 0.25 for 256MB)
}
```

### Key Behaviors

**Storage Engine Selection:**
- MongoDB <7.0: Uses `ephemeralForTest` (fast in-memory storage)
- MongoDB 7.0+: Uses `wiredTiger` (ephemeralForTest was removed)
- Replica sets: Always use `wiredTiger` (required)

**Configuration Precedence:**
1. Explicit `Options` struct parameters
2. Environment variables (`MEMONGO_CACHE_PATH`, `MEMONGO_DOWNLOAD_URL`, `MEMONGO_MONGOD_BIN`, `MEMONGO_MONGOD_PORT`)
3. System defaults

**Platform Support:**
- macOS (darwin) x86_64 and arm64
- Linux: Ubuntu, Debian, RHEL, SUSE, Amazon Linux
- **Apple Silicon:** Auto-detected and uses x86_64 binary via Rosetta 2 (no manual configuration needed)

## Linting Rules

- `math/rand` is blacklisted - use `crypto/rand` instead
- Cyclomatic complexity limit: 20
- Test files are exempt from some linters (gocyclo, errcheck, gosec, maligned)
