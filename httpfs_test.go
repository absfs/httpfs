package httpfs_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
