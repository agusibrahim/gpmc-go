# GPMC-Go - Google Photos Mobile Client

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/agusibrahim/gpmc-go)](LICENSE)

A high-performance Google Photos client written in Go, based on reverse-engineered mobile API. This is a Go implementation of the original [google_photos_mobile_client](https://github.com/xob0t/google_photos_mobile_client) project.

## Features

| Feature | Description |
|---------|-------------|
| **Smart Deduplication** | SHA-1 hash-based detection prevents duplicate uploads |
| **Concurrent Uploads** | Multi-threaded upload system with configurable worker pools |
| **Album Management** | Create albums, add media, AUTO mode for directory-based naming |
| **Local SQLite Cache** | Incremental sync of remote library metadata for fast lookups |
| **Advanced Filtering** | Regex, recursive scanning, file exclusion, path matching |
| **Progress Tracking** | Real-time progress updates via channels and progress bars |
| **Media Operations** | Trash, delete, restore, favorite, archive, caption management |
| **Download URLs** | Retrieve direct download links for any media item |
| **Config File Support** | YAML configuration at `~/.gpmc/config.yaml` |

## Installation

### Pre-built Binaries
Download the latest pre-built binaries for your platform (Windows, Linux, macOS) from the [GitHub Releases](https://github.com/agusibrahim/gpmc-go/releases) page.

### Prerequisites (if building from source)
- Go 1.25 or later
- Google Photos auth_data (captured from mobile app requests)

### From Source

```bash
git clone https://github.com/agusibrahim/gpmc-go.git
cd gpmc-go
go build -o gpmc ./cmd/gpmc
```

## Configuration

### Initialize Config

```bash
./gpmc init-config
```

This creates `~/.gpmc/config.yaml`:

```yaml
# Google Photos auth data (required)
# Capture from mobile app or set GP_AUTH_DATA environment variable
auth_data: ""

# Proxy settings (optional)
proxy: ""

# Language code (default: en-US)
language: "en-US"

# Request timeout in seconds (default: 60)
timeout: 60

# File upload timeout in seconds (default: 1800)
upload_timeout: 1800

# Log level: DEBUG, INFO, WARNING, ERROR, CRITICAL (default: INFO)
log_level: "INFO"

# Concurrent upload threads (default: 1)
threads: 1
```

### Priority Order
1. CLI flags
2. Environment variables (`GP_AUTH_DATA`, `GP_PROXY`)
3. Config file
4. Default values

## Usage

### Upload Files

```bash
# Basic upload
./gpmc /path/to/photo.jpg

# Upload to album
./gpmc /path/to/photos --album "Summer 2024"

# Recursive directory scan with progress
./gpmc /path/to/photos --recursive --progress --threads 8

# Upload with filters (regex)
./gpmc /path/to/photos --filter ".*\.jpg$" --regex --recursive

# Upload and delete from host after success
./gpmc /path/to/photo.jpg --delete-from-host

# Force upload (skip duplicate check)
./gpmc /path/to/photo.jpg --force-upload

# Storage saver quality
./gpmc /path/to/photo.jpg --saver
```

### Album Management

```bash
# AUTO album mode - uses directory name
./gpmc /path/to/2024/vacation --album AUTO --recursive

# Custom album name
./gpmc /path/to/photos --album "My Album"
```

### Cache Management

```bash
# Update local library cache
./gpmc update-cache --progress

# This enables fast duplicate detection by syncing your
# remote library metadata to local SQLite database
```

### Web UI Server

```bash
# Start web server
./gpmc serve --port 8080

# Open http://localhost:8080
```

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--auth-data` | Google auth data | from config |
| `--album` | Album name (use "AUTO" for directory-based) | "" |
| `--proxy` | Proxy URL (protocol://user:pass@host:port) | "" |
| `--progress` | Show upload progress | false |
| `--recursive` | Scan directories recursively | false |
| `--threads` | Concurrent upload threads | 1 |
| `--force-upload` | Skip hash check, upload all files | false |
| `--delete-from-host` | Delete files after successful upload | false |
| `--use-quota` | Count against storage quota | false |
| `--saver` | Upload in storage saver quality | false |
| `--timeout` | Request timeout (seconds) | 60 |
| `--upload-timeout` | File upload timeout (seconds) | 1800 |
| `--log-level` | DEBUG|INFO|WARNING|ERROR|CRITICAL | INFO |
| `--filter` | Filter expression for file selection | "" |
| `--exclude` | Exclude files matching filter | false |
| `--regex` | Use regex for filtering | false |
| `--ignore-case` | Case-insensitive filtering | false |
| `--match-path` | Match against full path instead of filename | false |
| `--config` | Config file path | ~/.gpmc/config.yaml |
| `--resume` | Resume completed files from local manifest | false |
| `--manifest-path` | Local upload manifest JSON path | "" |
| `--output` | Output format (`text` or `json`) | text |
| `--verify` | Verify uploaded media via hash lookup after commit | false |

## Architecture

```
gpmc-go/
├── cmd/
│   └── gpmc/
│       └── main.go              # CLI entry point (cobra)
├── internal/
│   ├── proto/
│   │   ├── types.go             # Protobuf type definitions
│   │   ├── encoder.go           # Protobuf encoding
│   │   ├── decoder.go           # Protobuf decoding
│   │   └── defs.go              # Message type definitions
│   ├── api/
│   │   ├── api.go               # Auth, session, retry logic
│   │   ├── endpoints.go         # API methods
│   │   └── urls.go              # URL constants
│   ├── client/
│   │   ├── client.go            # Main client
│   │   ├── upload.go            # Upload orchestration
│   │   ├── upload_options.go    # Upload options & progress types
│   │   ├── album.go             # Album management
│   │   ├── cache.go             # Cache sync logic
│   │   ├── media_ops.go         # Media operations
│   │   └── filter.go            # File filtering
│   ├── db/
│   │   ├── db.go                # SQLite storage
│   │   └── migrations.go        # Schema migrations
│   ├── models/
│   │   └── media_item.go        # MediaItem struct
│   ├── hash/
│   │   └── hash.go              # SHA-1 calculation
│   ├── parser/
│   │   └── db_update_parser.go  # Parse protobuf to MediaItem
│   ├── util/
│   │   └── util.go              # Utilities
│   ├── errors/
│   │   └── errors.go            # Custom error types
│   ├── config/
│   │   └── config.go            # YAML config handling
│   └── web/
│       └── server.go            # Web UI server
├── tests/
│   ├── integration_test.go      # E2E tests
│   └── run_all_tests.sh         # Test runner
├── go.mod
├── go.sum
└── README.md
```

## Supported Media Types

**Images**: jpg, jpeg, png, gif, webp, heic, heif, bmp, tiff, arw, cr2, dng, nef, orf, pef, raf, rw2, srw

**Videos**: mp4, mov, avi, mkv, webm, flv, wmv

## Testing

```bash
# Run unit tests
go test ./...

# Run integration tests (requires GP_AUTH_DATA environment variable)
go test ./tests/ -v
```

## Database Schema

Local SQLite cache at `~/.gpmc/cache.db`:

```sql
CREATE TABLE remote_media (
    media_key TEXT PRIMARY KEY,
    dedup_key TEXT,
    filename TEXT,
    -- 42 more columns for metadata
);

CREATE TABLE state (
    id INTEGER PRIMARY KEY CHECK(id=1),
    sync_token TEXT,
    resume_token TEXT,
    init_complete INTEGER
);

CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY
);
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GP_AUTH_DATA` | Google auth data |
| `GP_PROXY` | Proxy URL |

## Dependencies

```
google.golang.org/protobuf    # Protobuf encoding/decoding
github.com/spf13/cobra        # CLI framework
modernc.org/sqlite            # Pure-Go SQLite (no CGo)
github.com/schollz/progressbar/v3  # Progress bars
gopkg.in/yaml.v3              # YAML config
```

## Disclaimer

This project is for educational purposes only. Use at your own risk. Respect Google's Terms of Service.
