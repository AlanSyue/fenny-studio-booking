package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "127.8.9.10", "::1", // loopback
		"10.0.0.1", "172.16.5.4", "172.31.255.255", "192.168.1.1", // private
		"169.254.169.254", "fe80::1", // link-local (incl. cloud metadata)
		"fc00::1", "fd12:3456::1", // unique-local
		"224.0.0.1", "239.255.255.250", "ff02::1", "ff05::2", // multicast
		"0.0.0.0", "::", // unspecified
		"0.1.2.3", "100.64.0.1", "198.18.0.1", "240.0.0.1", "255.255.255.255",
		"::ffff:127.0.0.1", "::ffff:10.0.0.1", // IPv4-mapped
		"64:ff9b::a00:1", "2002:7f00:1::1", // NAT64 / 6to4 wrappers
	}
	allowed := []string{
		"1.1.1.1", "8.8.8.8", "172.32.0.1", "151.101.1.69",
		"2606:4700:4700::1111", "2001:4860:4860::8888", "::ffff:8.8.8.8",
	}
	for _, s := range blocked {
		if !isBlockedIP(netip.MustParseAddr(s)) {
			t.Errorf("%s should be blocked", s)
		}
	}
	for _, s := range allowed {
		if isBlockedIP(netip.MustParseAddr(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
	if !isBlockedIP(netip.Addr{}) {
		t.Error("zero Addr should be blocked")
	}
}

func TestCheckFetchURL(t *testing.T) {
	ctx := context.Background()
	bad := []string{
		"file:///etc/passwd",
		"gopher://example.com/",
		"ftp://example.com/x.png",
		"http:///nohost",
		"http://127.0.0.1/",
		"http://[::1]:8080/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.1.2.3/",
		"http://user:pass@example.com/",
		"http://localhost:8080/",
	}
	for _, raw := range bad {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		if err := checkFetchURL(ctx, u, nil); err == nil {
			t.Errorf("%s should be rejected", raw)
		}
	}
	u, _ := url.Parse("https://1.1.1.1/")
	if err := checkFetchURL(ctx, u, nil); err != nil {
		t.Errorf("public IP literal rejected: %v", err)
	}
}

// The dialer must refuse loopback even if URL validation were bypassed
// (e.g. DNS rebinding or a redirect to an internal address).
func TestSafeClientRefusesLoopbackAtDialTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request reached the loopback server")
	}))
	defer srv.Close()

	_, err := newSafeHTTPClient(5 * time.Second).Get(srv.URL)
	if err == nil || !errors.Is(err, errBlockedDestination) {
		t.Fatalf("expected blocked destination error, got %v", err)
	}
}

func TestSafeDialControl(t *testing.T) {
	if err := safeDialControl("tcp4", "127.0.0.1:80", nil); err == nil {
		t.Error("loopback allowed")
	}
	if err := safeDialControl("tcp6", "[fd00::1]:443", nil); err == nil {
		t.Error("unique-local allowed")
	}
	if err := safeDialControl("udp4", "8.8.8.8:53", nil); err == nil {
		t.Error("udp allowed")
	}
	if err := safeDialControl("tcp4", "8.8.8.8:443", nil); err != nil {
		t.Errorf("public address rejected: %v", err)
	}
}

func TestSafeClientLimitsRedirects(t *testing.T) {
	c := newSafeHTTPClient(time.Second)
	via := make([]*http.Request, maxFetchRedirects)
	req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	if err := c.CheckRedirect(req, via); err == nil {
		t.Error("redirect beyond limit allowed")
	}
	req = httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
	req.URL.Scheme = "file"
	if err := c.CheckRedirect(req, via[:1]); err == nil {
		t.Error("redirect to file scheme allowed")
	}
}

// End-to-end handler tests use an unrestricted client and a no-op URL check so
// the loopback httptest server is reachable; the restrictions themselves are
// covered above.
func newTestOGHandler(t *testing.T) (*OGImageHandler, string) {
	dir := t.TempDir()
	return &OGImageHandler{
		cfg:      &config.Config{UploadDir: dir},
		client:   &http.Client{Timeout: 5 * time.Second},
		checkURL: func(context.Context, *url.URL) error { return nil },
	}, dir
}

func postFetch(h *OGImageHandler, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/admin/fetch-image", strings.NewReader(`{"url":"`+target+`"}`))
	rec := httptest.NewRecorder()
	h.FetchFromURL(rec, req)
	return rec
}

func TestFetchFromURLSavesImage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><head><meta property="og:image" content="/img.png"></head></html>`))
	})
	mux.HandleFunc("/img.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("\x89PNG fake"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	h, dir := newTestOGHandler(t)
	rec := postFetch(h, srv.URL+"/page")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	files, _ := os.ReadDir(dir)
	if len(files) != 1 || filepath.Ext(files[0].Name()) != ".png" {
		t.Fatalf("unexpected files saved: %v", files)
	}
}

func TestFetchFromURLRejectsNonImageAndOversize(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/svg", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<meta property="og:image" content="/x.svg">`))
	})
	mux.HandleFunc("/x.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Write([]byte(`<svg onload="alert(1)"/>`))
	})
	mux.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<meta property="og:image" content="/x.html">`))
	})
	mux.HandleFunc("/x.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<script>alert(1)</script>`))
	})
	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<meta property="og:image" content="/big.jpg">`))
	})
	mux.HandleFunc("/big.jpg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		// No Content-Length (chunked), so the streaming cap must catch it.
		chunk := make([]byte, 1<<20)
		for i := 0; i <= maxOGImageBytes/len(chunk); i++ {
			w.Write(chunk)
			w.(http.Flusher).Flush()
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for _, path := range []string{"/svg", "/html", "/big"} {
		t.Run(path, func(t *testing.T) {
			h, dir := newTestOGHandler(t)
			rec := postFetch(h, srv.URL+path)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got %d: %s", rec.Code, rec.Body)
			}
			if files, _ := os.ReadDir(dir); len(files) != 0 {
				t.Fatalf("file left behind: %v", files)
			}
		})
	}
}

func TestFetchFromURLBlocksInternalWithRealPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request reached the loopback server")
	}))
	defer srv.Close()

	h := NewOGImageHandler(&config.Config{UploadDir: t.TempDir()})
	for _, target := range []string{srv.URL, "file:///etc/passwd", "http://169.254.169.254/"} {
		if rec := postFetch(h, target); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d, want 400", target, rec.Code)
		}
	}
}
