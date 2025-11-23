// Package httpfs implements a net/http FileSystem interface compatible
// wrapper around absfs.Filer that supports both read and write operations.
// It bridges the gap between the absfs filesystem abstraction and Go's
// standard http.FileServer.
package httpfs

import (
	"net/http"
	"os"
	"path/filepath"
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
	p := string(filepath.Separator)
	for _, name := range strings.Split(name, p) {
		if name == "" {
			continue
		}
		p = filepath.Join(p, name)
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

// RemoveAll removes a directory after removing all children of that directory.
func (filer *Httpfs) RemoveAll(path string) (err error) {
	info, err := filer.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// if it's not a directory remove it and return
	if !info.IsDir() {
		return filer.Remove(path)
	}

	f, err := filer.OpenFile(path, os.O_RDWR, 0700)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// get and loop through each directory entry calling remove all recursively
	infos, err := f.Readdir(0)
	if err != nil {
		return err
	}
	f.Close()

	for _, info := range infos {
		err = filer.RemoveAll(filepath.Join(path, info.Name()))
		if err != nil {
			return err
		}
	}

	return filer.Remove(path)
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
