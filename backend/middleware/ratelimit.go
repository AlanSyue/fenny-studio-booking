package middleware

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter is an in-memory token-bucket rate limiter keyed on client IP.
// It is defense in depth behind the nginx limit_req rules, so it is
// deliberately simple: one process, one map, idle entries swept lazily.
type IPRateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      float64 // tokens refilled per second
	burst     float64 // bucket capacity
	idleTTL   time.Duration
	lastSweep time.Time
	now       func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewIPRateLimiter allows `perMinute` requests per minute per IP with a burst
// of up to `burst` requests.
func NewIPRateLimiter(perMinute float64, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		buckets: make(map[string]*bucket),
		rate:    perMinute / 60,
		burst:   float64(burst),
		idleTTL: 10 * time.Minute,
		now:     time.Now,
	}
}

// Allow consumes one token for key. When the bucket is empty it returns false
// together with how long the caller should wait before retrying.
func (l *IPRateLimiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = math.Min(l.burst, b.tokens+elapsed*l.rate)
		}
		b.last = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.rate * float64(time.Second))
	return false, wait
}

// sweep drops buckets that have been idle long enough to be full again.
// Caller must hold l.mu.
func (l *IPRateLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < l.idleTTL {
		return
	}
	l.lastSweep = now
	for k, b := range l.buckets {
		if now.Sub(b.last) >= l.idleTTL {
			delete(l.buckets, k)
		}
	}
}

// Middleware rejects requests over the limit with 429 Too Many Requests.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, wait := l.Allow(ClientIP(r))
		if !ok {
			secs := int(math.Ceil(wait.Seconds()))
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Too many requests, please try again later"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClientIP returns the client IP. In production nginx sets X-Real-IP to the
// real visitor address (restored from Cloudflare), so it is preferred; when it
// is absent or malformed we fall back to the TCP peer address.
func ClientIP(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
		if ip := net.ParseIP(v); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
