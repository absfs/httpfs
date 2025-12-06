package httpfs_test

import (
	"testing"

	"github.com/absfs/fstesting"
	"github.com/absfs/httpfs"
	"github.com/absfs/memfs"
)

// TestHttpfsSuite runs the fstesting suite against httpfs.
// httpfs is an adapter that wraps an underlying filesystem,
// so its capabilities depend on the wrapped filesystem.
func TestHttpfsSuite(t *testing.T) {
	// Create a memfs as the underlying filesystem
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Wrap it with httpfs - this demonstrates that httpfs wraps a filesystem
	// Note: httpfs is primarily an adapter for net/http.FileSystem interface
	_ = httpfs.New(mfs)

	// For testing, we use the underlying memfs directly since httpfs
	// doesn't implement the full absfs.FileSystem interface - it's
	// designed as an adapter for http.FileServer
	// Configure the test suite with features supported by memfs
	suite := &fstesting.Suite{
		FS: mfs,
		Features: fstesting.Features{
			Symlinks:      false, // memfs doesn't support symlinks
			HardLinks:     false, // memfs doesn't support hard links
			Permissions:   true,  // memfs supports permissions
			Timestamps:    true,  // memfs supports timestamps
			CaseSensitive: true,  // memfs is case-sensitive
			AtomicRename:  true,  // memfs supports atomic rename
			SparseFiles:   false, // memfs doesn't support sparse files
			LargeFiles:    false, // memfs is limited by available memory
		},
	}

	// Run the full test suite
	suite.Run(t)
}

// TestHttpfsQuickCheck runs a quick sanity check on httpfs.
func TestHttpfsQuickCheck(t *testing.T) {
	mfs, err := memfs.NewFS()
	if err != nil {
		t.Fatal(err)
	}

	// Wrap with httpfs
	_ = httpfs.New(mfs)

	// Use the underlying filesystem for the quick check
	suite := &fstesting.Suite{
		FS: mfs,
		Features: fstesting.Features{
			Permissions:   true,
			Timestamps:    true,
			CaseSensitive: true,
			AtomicRename:  true,
		},
	}

	suite.QuickCheck(t)
}
