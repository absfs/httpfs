# httpfs

[![CI](https://github.com/absfs/httpfs/actions/workflows/ci.yml/badge.svg)](https://github.com/absfs/httpfs/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/absfs/httpfs)](https://goreportcard.com/report/github.com/absfs/httpfs)
[![GoDoc](https://godoc.org/github.com/absfs/httpfs?status.svg)](https://godoc.org/github.com/absfs/httpfs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The `httpfs` package implements a `net/http` FileSystem interface compatible object that includes both file reading and file writing operations. It provides a bridge between the [absfs](https://github.com/absfs/absfs) filesystem abstraction and Go's standard `http.FileServer`.

## Features

- Implements `http.FileSystem` interface for serving files over HTTP
- Full read/write filesystem operations (Open, OpenFile, Mkdir, Remove, etc.)
- Compatible with any `absfs.Filer` implementation
- Supports recursive directory operations (MkdirAll, RemoveAll)
- File metadata operations (Stat, Chmod, Chtimes, Chown)

## Installation

```bash
go get github.com/absfs/httpfs
```

## Usage

### Basic HTTP File Server

```go
package main

import (
    "log"
    "net/http"

    "github.com/absfs/httpfs"
    "github.com/absfs/memfs"
)

func main() {
    // Create an in-memory filesystem
    mfs, err := memfs.NewFS()
    if err != nil {
        log.Fatal(err)
    }

    // Wrap it with httpfs
    fs := httpfs.New(mfs)

    // Serve files over HTTP
    http.Handle("/", http.FileServer(fs))
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### File Operations

```go
package main

import (
    "log"
    "os"

    "github.com/absfs/httpfs"
    "github.com/absfs/memfs"
)

func main() {
    // Create filesystem
    mfs, _ := memfs.NewFS()
    fs := httpfs.New(mfs)

    // Create a directory
    if err := fs.MkdirAll("/path/to/dir", 0755); err != nil {
        log.Fatal(err)
    }

    // Create and write to a file
    f, err := fs.OpenFile("/path/to/file.txt", os.O_CREATE|os.O_RDWR, 0644)
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    f.Write([]byte("Hello, World!"))

    // Get file info
    info, err := fs.Stat("/path/to/file.txt")
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("File size: %d bytes", info.Size())
}
```

## API

### Core Methods

- `New(fs absfs.Filer) *Httpfs` - Creates a new HTTP filesystem wrapper
- `Open(name string) (http.File, error)` - Opens a file for reading
- `OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error)` - Opens a file with flags

### Directory Operations

- `Mkdir(name string, perm os.FileMode) error` - Creates a directory
- `MkdirAll(name string, perm os.FileMode) error` - Creates all directories in path
- `Remove(name string) error` - Removes a file or empty directory
- `RemoveAll(path string) error` - Recursively removes a directory and its contents

### Metadata Operations

- `Stat(name string) (os.FileInfo, error)` - Returns file information
- `Chmod(name string, mode os.FileMode) error` - Changes file mode
- `Chtimes(name string, atime time.Time, mtime time.Time) error` - Changes access/modification times
- `Chown(name string, uid, gid int) error` - Changes file owner and group

## Requirements

- Go 1.22 or higher

## Testing

```bash
go test -v ./...
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Related Projects

- [absfs](https://github.com/absfs/absfs) - Abstract filesystem interface for Go
- [memfs](https://github.com/absfs/memfs) - In-memory filesystem implementation
