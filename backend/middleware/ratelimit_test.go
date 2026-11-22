package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time      { return c.t }
func (c *fakeClock) Add(d time.Duration) { c.t = c.t.Add(d) }
func newTestLimiter(perMin float64, burst int) (*IPRateLimiter, *fakeClock) {
	clk := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	l := NewIPRateLimiter(perMin, burst)
	l.now = clk.Now
	return l, clk
}

func TestIPRateLimiterBurstThenRefill(t *testing.T) {
	l, clk := newTestLimiter(5, 5)

	for i := 0; i < 5; i++ {
		if ok, _ := l.Allow("1.2.3.4"); !ok {
			t.Fatalf("request %d within burst was rejected", i+1)
		}
	}
	ok, wait := l.Allow("1.2.3.4")
	if ok {
		t.Fatal("6th request should be rejected")
	}
	if wait <= 0 || wait > 12*time.Second {
		t.Fatalf("unexpected wait %v, want (0, 12s]", wait)
	}

	// 5/min => one token every 12s.
	clk.Add(11 * time.Second)
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Fatal("should still be limited after 11s")
	}
	clk.Add(1 * time.Second)
	if ok, _ := l.Allow("1.2.3.4"); !ok {
		t.Fatal("should allow one request after refill")
	}
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Fatal("only one token should have been refilled")
	}
}

func TestIPRateLimiterKeysAreIndependent(t *testing.T) {
	l, _ := newTestLimiter(1, 1)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("first request for a rejected")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("second request for a should be rejected")
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("b must not share a's bucket")
	}
}

func TestIPRateLimiterSweepsIdleBuckets(t *testing.T) {
	l, clk := newTestLimiter(1, 1)
	l.Allow("a")
	clk.Add(l.idleTTL + time.Second)
	l.Allow("b")
	if _, ok := l.buckets["a"]; ok {
		t.Fatal("idle bucket a should have been swept")
	}
}

func TestMiddlewareReturns429(t *testing.T) {
	l, _ := newTestLimiter(5, 2)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(realIP string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "10.0.0.1:5555"
		if realIP != "" {
			req.Header.Set("X-Real-IP", realIP)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	for i := 0; i < 2; i++ {
		if rec := do("203.0.113.7"); rec.Code != http.StatusOK {
			t.Fatalf("request %d: got %d", i+1, rec.Code)
		}
	}
	rec := do("203.0.113.7")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
	// Different X-Real-IP behind the same proxy is a different client.
	if rec := do("203.0.113.8"); rec.Code != http.StatusOK {
		t.Fatalf("other client: got %d", rec.Code)
	}
}

func TestClientIP(t *testing.T) {
	cases := []struct {
		name, realIP, remote, want string
	}{
		{"x-real-ip preferred", "198.51.100.9", "127.0.0.1:1234", "198.51.100.9"},
		{"x-real-ip ipv6", "2001:db8::1", "127.0.0.1:1234", "2001:db8::1"},
		{"x-real-ip trimmed", " 198.51.100.9 ", "127.0.0.1:1234", "198.51.100.9"},
		{"malformed x-real-ip falls back", "not-an-ip", "192.0.2.5:80", "192.0.2.5"},
		{"no header uses remote addr", "", "192.0.2.5:80", "192.0.2.5"},
		{"remote addr without port", "", "192.0.2.5", "192.0.2.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remote
			if c.realIP != "" {
				r.Header.Set("X-Real-IP", c.realIP)
			}
			if got := ClientIP(r); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
