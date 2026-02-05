package memongo

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/100mslive/memongo/v2/memongolog"
	"github.com/100mslive/memongo/v2/mongobin"
)

// Options is the configuration options for a launched MongoDB binary
type Options struct {
	// ShouldUseReplica indicates whether a replica should be used. If this is not specified,
	// no replica will be used and mongo server will be run as standalone.
	ShouldUseReplica bool

	// ReplicaSetName is the name of the replica set. Defaults to "rs0".
	// Only used when ShouldUseReplica is true.
	ReplicaSetName string

	// Port to run MongoDB on. If this is not specified, a random (OS-assigned)
	// port will be used
	Port int

	// Path to the cache for downloaded mongod binaries. Defaults to the
	// system cache location.
	CachePath string

	// If DownloadURL and MongodBin are not given, this version of MongoDB will
	// be downloaded
	MongoVersion string

	// If given, mongod will be downloaded from this URL instead of the
	// auto-detected URL based on the current platform and MongoVersion
	DownloadURL string

	// If given, this binary will be run instead of downloading a mongod binary
	MongodBin string

	// Logger for printing messages. Defaults to printing to stdout.
	Logger *log.Logger

	// A LogLevel to log at. Defaults to LogLevelInfo.
	LogLevel memongolog.LogLevel

	// How long to wait for mongod to start up and report a port number. Does
	// not include download time, only startup time. Defaults to 10 seconds.
	StartupTimeout time.Duration

	// If set, pass the --auth flag to mongod. This will allow tests to setup
	// authentication.
	Auth bool

	// WiredTigerCacheSizeGB sets the maximum size of the WiredTiger cache in GB.
	// This is useful to limit memory usage in test environments.
	// Only applies when using WiredTiger storage engine (MongoDB 7.0+ or replica sets).
	// If not set, MongoDB uses its default (typically 50% of RAM minus 1GB).
	WiredTigerCacheSizeGB float64
}

func (opts *Options) fillDefaults() error {
	// Set default replica set name
	if opts.ReplicaSetName == "" {
		opts.ReplicaSetName = "rs0"
	}

	if opts.MongodBin == "" {
		opts.MongodBin = os.Getenv("MEMONGO_MONGOD_BIN")
	}
	if opts.MongodBin == "" {
		// The user didn't give us a local path to a binary. That means we need
		// a download URL and a cache path.

		// Determine the cache path
		if opts.CachePath == "" {
			opts.CachePath = os.Getenv("MEMONGO_CACHE_PATH")
		}
		if opts.CachePath == "" && os.Getenv("XDG_CACHE_HOME") != "" {
			opts.CachePath = path.Join(os.Getenv("XDG_CACHE_HOME"), "memongo")
		}
		if opts.CachePath == "" {
			if runtime.GOOS == "darwin" {
				opts.CachePath = path.Join(os.Getenv("HOME"), "Library", "Caches", "memongo")
			} else {
				opts.CachePath = path.Join(os.Getenv("HOME"), ".cache", "memongo")
			}
		}

		// Determine the download URL
		if opts.DownloadURL == "" {
			opts.DownloadURL = os.Getenv("MEMONGO_DOWNLOAD_URL")
		}
		if opts.DownloadURL == "" {
			if opts.MongoVersion == "" {
				return fmt.Errorf("one of MongoVersion, DownloadURL, or MongodBin must be given")
			}

			// Auto-detect Apple Silicon and use x86_64 binary via Rosetta 2
			if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
				opts.DownloadURL = getAppleSiliconDownloadURL(opts.MongoVersion)
			} else {
				spec, err := mongobin.MakeDownloadSpec(opts.MongoVersion)
				if err != nil {
					return err
				}
				opts.DownloadURL = spec.GetDownloadURL()
			}
		}
	}

	// Determine the port number
	if opts.Port == 0 {
		mongoVersionEnv := os.Getenv("MEMONGO_MONGOD_PORT")
		if mongoVersionEnv != "" {
			port, err := strconv.Atoi(mongoVersionEnv)

			if err != nil {
				return fmt.Errorf("error parsing MEMONGO_MONGOD_PORT: %s", err)
			}

			opts.Port = port
		}
	}

	if opts.Port == 0 {
		port, err := getFreePort()
		if err != nil {
			return fmt.Errorf("error finding a free port: %s", err)
		}

		opts.Port = port

		if opts.StartupTimeout == 0 {
			opts.StartupTimeout = 10 * time.Second
		}
	}

	return nil
}

func (opts *Options) getLogger() *memongolog.Logger {
	return memongolog.New(opts.Logger, opts.LogLevel)
}

func (opts *Options) getOrDownloadBinPath() (string, error) {
	if opts.MongodBin != "" {
		return opts.MongodBin, nil
	}

	// Download or fetch from cache
	binPath, err := mongobin.GetOrDownloadMongod(opts.DownloadURL, opts.CachePath, opts.getLogger())
	if err != nil {
		return "", err
	}

	return binPath, nil
}

func getFreePort() (int, error) {
	// Based on: https://github.com/phayes/freeport/blob/master/freeport.go
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// getAppleSiliconDownloadURL returns the x86_64 macOS download URL for Apple Silicon Macs.
// Apple Silicon can run x86_64 binaries via Rosetta 2.
func getAppleSiliconDownloadURL(version string) string {
	// For MongoDB 6.0+, native arm64 builds are available but may have issues,
	// so we use x86_64 via Rosetta 2 for maximum compatibility.
	// Format: https://fastdl.mongodb.org/osx/mongodb-macos-x86_64-VERSION.tgz
	return fmt.Sprintf("https://fastdl.mongodb.org/osx/mongodb-macos-x86_64-%s.tgz", version)
}
