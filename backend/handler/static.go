package handler

import (
	"net/http"
	"os"
)

// NoDirFileSystem wraps an http.FileSystem so that directories are reported as
// not found. http.FileServer then serves files normally but returns 404
// instead of an auto-generated directory listing (or a directory index.html).
type NoDirFileSystem struct {
	FS http.FileSystem
}

func (n NoDirFileSystem) Open(name string) (http.File, error) {
	f, err := n.FS.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}
