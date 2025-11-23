package httpfs_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/absfs/httpfs"
	"github.com/absfs/memfs"
)

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
