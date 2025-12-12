// Package httpfs implements a net/http FileSystem interface compatible
// wrapper around absfs.Filer that supports both read and write operations.
// It bridges the gap between the absfs filesystem abstraction and Go's
// standard http.FileServer.
package httpfs

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/absfs/absfs"
)

type Httpfs struct {
	fs absfs.Filer
}

func New(fs absfs.Filer) *Httpfs {
	return &Httpfs{fs}
}

func (filer *Httpfs) Open(name string) (http.File, error) {
	f, err := filer.OpenFile(name, os.O_RDONLY, 0400)

	return http.File(f), err
}

// OpenFile opens a file using the given flags and the given mode.
func (filer *Httpfs) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	return filer.fs.OpenFile(name, flag, perm)
}

// Mkdir creates a directory in the filesystem, return an error if any
// happens.
func (filer *Httpfs) Mkdir(name string, perm os.FileMode) error {
	return filer.fs.Mkdir(name, perm)
}

// MkdirAll creates all missing directories in `name` without returning an error
// for directories that already exist
func (filer *Httpfs) MkdirAll(name string, perm os.FileMode) error {
	// Virtual filesystems use forward slashes
	p := "/"
	for _, name := range strings.Split(name, "/") {
		if name == "" {
			continue
		}
		p = path.Join(p, name)
		err := filer.Mkdir(p, perm)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (filer *Httpfs) Remove(name string) error {
	return filer.fs.Remove(name)
}

// RemoveAller is an optional interface that filesystems can implement
// to provide optimized recursive directory removal.
type RemoveAller interface {
	RemoveAll(path string) error
}

// RemoveAll removes a directory after removing all children of that directory.
// If the underlying filesystem implements RemoveAller, it delegates to that.
// Returns nil for non-existent paths (matching os.RemoveAll behavior).
func (filer *Httpfs) RemoveAll(pathname string) error {
	// Check if the underlying filesystem implements RemoveAll
	if ra, ok := filer.fs.(RemoveAller); ok {
		err := ra.RemoveAll(pathname)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	info, err := filer.Stat(pathname)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// If it's not a directory, just remove it
	if !info.IsDir() {
		return filer.Remove(pathname)
	}

	// Open directory read-only to list entries
	f, err := filer.OpenFile(pathname, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	infos, err := f.Readdir(0)
	f.Close()
	if err != nil {
		return err
	}

	for _, info := range infos {
		name := info.Name()
		// Skip . and .. to avoid infinite recursion
		if name == "." || name == ".." {
			continue
		}
		if err := filer.RemoveAll(path.Join(pathname, name)); err != nil {
			return err
		}
	}

	err = filer.Remove(pathname)
	// Some filesystems (e.g., memfs) return "directory not empty" even when
	// only . and .. remain. Check if the directory was actually removed.
	if err != nil {
		if _, statErr := filer.Stat(pathname); os.IsNotExist(statErr) {
			return nil
		}
	}
	return err
}

// Stat returns the FileInfo structure describing file. If there is an error, it will be of type *PathError.
func (filer *Httpfs) Stat(name string) (os.FileInfo, error) {
	return filer.fs.Stat(name)
}

// Chmod changes the mode of the named file to mode.
func (filer *Httpfs) Chmod(name string, mode os.FileMode) error {
	return filer.fs.Chmod(name, mode)
}

// Chtimes changes the access and modification times of the named file.
func (filer *Httpfs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return filer.fs.Chtimes(name, atime, mtime)
}

// Chown changes the owner and group ids of the named file.
func (filer *Httpfs) Chown(name string, uid, gid int) error {
	return filer.fs.Chown(name, uid, gid)
}

// ReadDir reads the named directory and returns a list of directory entries
// sorted by filename.
func (filer *Httpfs) ReadDir(name string) ([]fs.DirEntry, error) {
	return filer.fs.ReadDir(name)
}

// ReadFile reads the named file and returns its contents.
// A successful call returns err == nil, not err == EOF.
func (filer *Httpfs) ReadFile(name string) ([]byte, error) {
	return filer.fs.ReadFile(name)
}

// Sub returns an fs.FS corresponding to the subtree rooted at dir.
func (filer *Httpfs) Sub(dir string) (fs.FS, error) {
	return absfs.FilerToFS(filer.fs, dir)
}
