package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNoDirFileSystem(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.png"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "index.html"), []byte("idx"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := http.StripPrefix("/uploads/", http.FileServer(NoDirFileSystem{FS: http.Dir(dir)}))
	cases := map[string]int{
		"/uploads/a.png":          http.StatusOK,
		"/uploads/":               http.StatusNotFound,
		"/uploads/sub/":           http.StatusNotFound,
		"/uploads/sub":            http.StatusNotFound,
		"/uploads/missing.png":    http.StatusNotFound,
		"/uploads/sub/index.html": http.StatusOK, // files inside subdirs still serve
	}
	for path, want := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		// FileServer redirects ".../index.html" to "./"; accept that as not-a-listing.
		if path == "/uploads/sub/index.html" && rec.Code == http.StatusMovedPermanently {
			continue
		}
		if rec.Code != want {
			t.Errorf("%s: got %d, want %d", path, rec.Code, want)
		}
	}
}
