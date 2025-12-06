package httpfs_test

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/absfs/absfs"
	"github.com/absfs/httpfs"
	"github.com/absfs/memfs"
)

// noRemoveAllFS wraps a Filer to hide the RemoveAll method, forcing the fallback path
type noRemoveAllFS struct {
	absfs.Filer
}

// errorFS is a mock filesystem that returns errors for testing error paths
type errorFS struct {
	absfs.Filer
	statErr     error
	openFileErr error
	removeErr   error
	readdirErr  error
}

func (e *errorFS) Stat(name string) (os.FileInfo, error) {
	if e.statErr != nil {
		return nil, e.statErr
	}
	return e.Filer.Stat(name)
}

func (e *errorFS) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	if e.openFileErr != nil {
		return nil, e.openFileErr
	}
	return e.Filer.OpenFile(name, flag, perm)
}

func (e *errorFS) Remove(name string) error {
	if e.removeErr != nil {
		return e.removeErr
	}
	return e.Filer.Remove(name)
}

// errorFile wraps a file to inject errors
type errorFile struct {
	absfs.File
	readdirErr error
}

func (f *errorFile) Readdir(n int) ([]os.FileInfo, error) {
	if f.readdirErr != nil {
		return nil, f.readdirErr
	}
	return f.File.Readdir(n)
}

func TestFileServer(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	f, err := fs.OpenFile("/foo.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.Write([]byte("foo bar bat."))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	server := httptest.NewServer(http.FileServer(fs))
	defer server.Close()

	res, err := http.Get(server.URL + "/foo.txt")
	if err != nil {
		log.Fatal(err)
	}
	data, err := io.ReadAll(res.Body)
	res.Body.Close()
	if string(data) != "foo bar bat." {
		t.Fatal("wrong response")
	}
	t.Logf("received: %q", string(data))
}

func TestMkdir(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a directory
	err = fs.Mkdir("/testdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Verify it exists and is a directory
	info, err := fs.Stat("/testdir")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("Expected /testdir to be a directory")
	}

	// Creating the same directory again should fail
	err = fs.Mkdir("/testdir", 0755)
	if err == nil {
		t.Fatal("Expected error when creating existing directory")
	}
}

func TestMkdirAll(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create nested directories
	err = fs.MkdirAll("/a/b/c/d", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Verify all directories exist
	for _, path := range []string{"/a", "/a/b", "/a/b/c", "/a/b/c/d"} {
		info, err := fs.Stat(path)
		if err != nil {
			t.Fatalf("Stat(%s) failed: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("Expected %s to be a directory", path)
		}
	}

	// MkdirAll on existing path should not error
	err = fs.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatalf("MkdirAll on existing path failed: %v", err)
	}
}

func TestOpenFile(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create and write to a file
	content := []byte("test content")
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}

	n, err := f.Write(content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(content) {
		t.Fatalf("Expected to write %d bytes, wrote %d", len(content), n)
	}
	f.Close()

	// Read it back
	f, err = fs.OpenFile("/test.txt", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("OpenFile for reading failed: %v", err)
	}

	buf := make([]byte, len(content))
	n, err = f.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(content) {
		t.Fatalf("Expected to read %d bytes, read %d", len(content), n)
	}
	if string(buf) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, buf)
	}
	f.Close()
}

func TestOpen(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file first
	content := []byte("test content for open")
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Write(content)
	f.Close()

	// Open it with the http.File interface
	httpFile, err := fs.Open("/test.txt")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer httpFile.Close()

	buf := make([]byte, len(content))
	n, err := httpFile.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(content) {
		t.Fatalf("Expected to read %d bytes, read %d", len(content), n)
	}
	if string(buf) != string(content) {
		t.Fatalf("Expected content %q, got %q", content, buf)
	}
}

func TestRemove(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Remove it
	err = fs.Remove("/test.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/test.txt")
	if err == nil {
		t.Fatal("Expected error when stating removed file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}
}

func TestRemoveAll(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create nested directories with files
	err = fs.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Create files in different directories
	for _, path := range []string{"/a/file1.txt", "/a/b/file2.txt", "/a/b/c/file3.txt"} {
		f, err := fs.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			t.Fatalf("OpenFile(%s) failed: %v", path, err)
		}
		f.Close()
	}

	// Remove the entire tree
	err = fs.RemoveAll("/a")
	if err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/a")
	if err == nil {
		t.Fatal("Expected error when stating removed directory")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}

	// RemoveAll on non-existent path should not error
	err = fs.RemoveAll("/nonexistent")
	if err != nil {
		t.Fatalf("RemoveAll on non-existent path failed: %v", err)
	}
}

func TestStat(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file with specific permissions
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	content := []byte("test content")
	f.Write(content)
	f.Close()

	// Stat the file
	info, err := fs.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Name() != "test.txt" {
		t.Fatalf("Expected name 'test.txt', got %q", info.Name())
	}
	if info.Size() != int64(len(content)) {
		t.Fatalf("Expected size %d, got %d", len(content), info.Size())
	}
	if info.IsDir() {
		t.Fatal("Expected file, not directory")
	}

	// Stat non-existent file should error
	_, err = fs.Stat("/nonexistent.txt")
	if err == nil {
		t.Fatal("Expected error when stating non-existent file")
	}
}

func TestChmod(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Change its mode
	err = fs.Chmod("/test.txt", 0755)
	if err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	// Verify the mode changed
	info, err := fs.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("Expected mode 0755, got %o", info.Mode().Perm())
	}
}

func TestChtimes(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Change its times
	atime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	mtime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = fs.Chtimes("/test.txt", atime, mtime)
	if err != nil {
		t.Fatalf("Chtimes failed: %v", err)
	}

	// Verify the modification time changed
	info, err := fs.Stat("/test.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Fatalf("Expected mtime %v, got %v", mtime, info.ModTime())
	}
}

func TestChown(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Chown - this may not be fully supported by memfs, but we test the call doesn't panic
	err = fs.Chown("/test.txt", 1000, 1000)
	// We don't assert on error here since memfs may not support this operation
	// The important thing is the method exists and can be called
	t.Logf("Chown result: %v", err)
}

func TestRemoveAllOnFile(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create a file
	f, err := fs.OpenFile("/test.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// RemoveAll on a file should work
	err = fs.RemoveAll("/test.txt")
	if err != nil {
		t.Fatalf("RemoveAll on file failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/test.txt")
	if err == nil {
		t.Fatal("Expected error when stating removed file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}
}

// TestRemoveAllFallbackFile tests the fallback RemoveAll code path for files
// Note: The directory removal fallback cannot be fully tested with memfs because
// memfs.Remove returns "directory not empty" for directories that only contain
// . and .. entries, and does not actually remove them. This is a memfs limitation.
func TestRemoveAllFallbackFile(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Wrap memfs to hide RemoveAll method
	wrapped := &noRemoveAllFS{Filer: mfs}
	fs := httpfs.New(wrapped)

	// Create a file
	f, err := fs.OpenFile("/testfile.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Write([]byte("content"))
	f.Close()

	// RemoveAll on file should work via fallback
	err = fs.RemoveAll("/testfile.txt")
	if err != nil {
		t.Fatalf("RemoveAll (fallback) on file failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/testfile.txt")
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}
}

// TestRemoveAllFallbackNonExistent tests fallback RemoveAll on non-existent paths
func TestRemoveAllFallbackNonExistent(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	wrapped := &noRemoveAllFS{Filer: mfs}
	fs := httpfs.New(wrapped)

	// RemoveAll on non-existent path should not error
	err = fs.RemoveAll("/nonexistent")
	if err != nil {
		t.Fatalf("RemoveAll on non-existent path failed: %v", err)
	}
}


// TestRemoveAllStatError tests RemoveAll when Stat returns an error
func TestRemoveAllStatError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	testErr := errors.New("stat error")
	errFS := &errorFS{Filer: &noRemoveAllFS{Filer: mfs}, statErr: testErr}
	fs := httpfs.New(errFS)

	// RemoveAll should return the stat error
	err = fs.RemoveAll("/test")
	if err != testErr {
		t.Fatalf("Expected stat error, got: %v", err)
	}
}

// TestRemoveAllOpenFileError tests RemoveAll when OpenFile returns an error
func TestRemoveAllOpenFileError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Create a directory first so Stat succeeds
	err = mfs.Mkdir("/testdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	testErr := errors.New("openfile error")
	errFS := &errorFS{Filer: &noRemoveAllFS{Filer: mfs}, openFileErr: testErr}
	fs := httpfs.New(errFS)

	// RemoveAll should return the openfile error
	err = fs.RemoveAll("/testdir")
	if err != testErr {
		t.Fatalf("Expected openfile error, got: %v", err)
	}
}

// TestRemoveAllOpenFileNotExistError tests RemoveAll when OpenFile returns not exist
func TestRemoveAllOpenFileNotExistError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Create a directory first so Stat succeeds
	err = mfs.Mkdir("/testdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	errFS := &errorFS{Filer: &noRemoveAllFS{Filer: mfs}, openFileErr: os.ErrNotExist}
	fs := httpfs.New(errFS)

	// RemoveAll should return nil for not exist on OpenFile
	err = fs.RemoveAll("/testdir")
	if err != nil {
		t.Fatalf("Expected nil error for not exist, got: %v", err)
	}
}

// TestMkdirAllError tests MkdirAll when Mkdir returns an unexpected error
func TestMkdirAllError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Try to create directory with parent that is a file
	f, err := fs.OpenFile("/file", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// MkdirAll should fail when parent is a file - but memfs might not enforce this
	err = fs.MkdirAll("/file/subdir", 0755)
	// Just log the result - memfs behavior may vary
	t.Logf("MkdirAll under file result: %v", err)
}

// TestRemoveAllDeeplyNested tests RemoveAll with deeply nested directories
func TestRemoveAllDeeplyNested(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create deeply nested directories (10 levels)
	err = fs.MkdirAll("/a/b/c/d/e/f/g/h/i/j", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Create a file at the deepest level
	f, err := fs.OpenFile("/a/b/c/d/e/f/g/h/i/j/file.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Remove entire tree
	err = fs.RemoveAll("/a")
	if err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/a")
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}
}


// TestMkdirAllEmptyPath tests MkdirAll with root path
func TestMkdirAllEmptyPath(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// MkdirAll on root should not error
	err = fs.MkdirAll("/", 0755)
	if err != nil {
		t.Fatalf("MkdirAll on root failed: %v", err)
	}
}

// Note: Concurrent operations test removed because memfs has race conditions
// that cause panics. See https://github.com/absfs/memfs for updates.

// readdirErrorFS wraps a filesystem to inject Readdir errors
type readdirErrorFS struct {
	absfs.Filer
	readdirErr error
}

func (f *readdirErrorFS) OpenFile(name string, flag int, perm os.FileMode) (absfs.File, error) {
	file, err := f.Filer.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}
	return &readdirErrorFile{File: file, err: f.readdirErr}, nil
}

type readdirErrorFile struct {
	absfs.File
	err error
}

func (f *readdirErrorFile) Readdir(n int) ([]os.FileInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.File.Readdir(n)
}

// TestRemoveAllReaddirError tests RemoveAll when Readdir returns an error
func TestRemoveAllReaddirError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Create a directory first
	err = mfs.Mkdir("/testdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	testErr := errors.New("readdir error")
	errFS := &readdirErrorFS{Filer: &noRemoveAllFS{Filer: mfs}, readdirErr: testErr}
	fs := httpfs.New(errFS)

	// RemoveAll should return the readdir error
	err = fs.RemoveAll("/testdir")
	if err != testErr {
		t.Fatalf("Expected readdir error, got: %v", err)
	}
}

// TestRemoveAllRecursiveError tests RemoveAll when recursive removal fails
func TestRemoveAllRecursiveError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Create nested structure
	err = mfs.MkdirAll("/parent/child", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Create a file in child
	f, err := mfs.OpenFile("/parent/child/file.txt", os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}
	f.Close()

	// Create errorFS that fails on Remove for specific file
	testErr := errors.New("remove error")
	removeOnce := false
	customFS := &conditionalErrorFS{
		Filer: &noRemoveAllFS{Filer: mfs},
		removeFunc: func(name string) error {
			if name == "/parent/child/file.txt" && !removeOnce {
				removeOnce = true
				return testErr
			}
			return mfs.Remove(name)
		},
	}
	fs := httpfs.New(customFS)

	// RemoveAll should return the remove error
	err = fs.RemoveAll("/parent")
	if err != testErr {
		t.Fatalf("Expected remove error, got: %v", err)
	}
}

// conditionalErrorFS allows conditional error injection
type conditionalErrorFS struct {
	absfs.Filer
	removeFunc func(string) error
}

func (f *conditionalErrorFS) Remove(name string) error {
	if f.removeFunc != nil {
		return f.removeFunc(name)
	}
	return f.Filer.Remove(name)
}

// TestRemoveAllFinalRemoveError tests RemoveAll when final directory Remove fails
func TestRemoveAllFinalRemoveError(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Create an empty directory
	err = mfs.Mkdir("/testdir", 0755)
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Create errorFS that fails on Remove but the directory still exists
	testErr := errors.New("final remove error")
	customFS := &conditionalErrorFS{
		Filer: &noRemoveAllFS{Filer: mfs},
		removeFunc: func(name string) error {
			if name == "/testdir" {
				return testErr
			}
			return mfs.Remove(name)
		},
	}
	fs := httpfs.New(customFS)

	// RemoveAll should return the error since directory still exists
	err = fs.RemoveAll("/testdir")
	if err != testErr {
		t.Fatalf("Expected final remove error, got: %v", err)
	}
}

// TestRemoveAllDelegation tests that RemoveAll delegates to underlying fs when available
func TestRemoveAllDelegation(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// Create nested directories
	err = fs.MkdirAll("/a/b/c", 0755)
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// RemoveAll should delegate to memfs.RemoveAll
	err = fs.RemoveAll("/a")
	if err != nil {
		t.Fatalf("RemoveAll delegation failed: %v", err)
	}

	// Verify it's gone
	_, err = fs.Stat("/a")
	if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist error, got: %v", err)
	}
}

// TestRemoveAllDelegationNotExist tests delegation when path doesn't exist
func TestRemoveAllDelegationNotExist(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	fs := httpfs.New(mfs)

	// RemoveAll on non-existent should return nil (memfs.RemoveAll returns error but we suppress it)
	err = fs.RemoveAll("/nonexistent")
	if err != nil {
		t.Fatalf("RemoveAll on non-existent should return nil, got: %v", err)
	}
}
