package handler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"
)

// errBlockedDestination is returned when a URL (or a host it resolves to)
// points at an address the server must never reach on a user's behalf.
var errBlockedDestination = errors.New("destination address is not allowed")

const maxFetchRedirects = 3

// blockedPrefixes are special-purpose ranges not already covered by the
// netip.Addr predicates used in isBlockedIP.
var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),       // "this network"
	netip.MustParsePrefix("100.64.0.0/10"),   // carrier-grade NAT
	netip.MustParsePrefix("192.0.0.0/24"),    // IETF protocol assignments
	netip.MustParsePrefix("192.0.2.0/24"),    // TEST-NET-1
	netip.MustParsePrefix("198.18.0.0/15"),   // benchmarking
	netip.MustParsePrefix("198.51.100.0/24"), // TEST-NET-2
	netip.MustParsePrefix("203.0.113.0/24"),  // TEST-NET-3
	netip.MustParsePrefix("240.0.0.0/4"),     // reserved + broadcast
	netip.MustParsePrefix("64:ff9b::/96"),    // NAT64: may map onto internal IPv4
	netip.MustParsePrefix("64:ff9b:1::/48"),  // local-use NAT64
	netip.MustParsePrefix("2001:db8::/32"),   // documentation
	netip.MustParsePrefix("2002::/16"),       // 6to4: embeds an arbitrary IPv4
	netip.MustParsePrefix("100::/64"),        // discard-only
}

// isBlockedIP reports whether ip is loopback, private, link-local,
// unique-local, multicast, unspecified or otherwise not a public unicast
// address. IPv4-mapped IPv6 addresses are checked as IPv4.
func isBlockedIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	ip = ip.Unmap()
	if ip.IsLoopback() ||
		ip.IsPrivate() || // 10/8, 172.16/12, 192.168/16, fc00::/7 (unique-local)
		ip.IsLinkLocalUnicast() || // 169.254/16 (incl. cloud metadata), fe80::/10
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() {
		return true
	}
	if ip.Is6() && !ip.IsGlobalUnicast() {
		return true
	}
	for _, p := range blockedPrefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialControl runs after DNS resolution, right before connect(2), on the
// exact IP being dialled. Enforcing the policy here (rather than only when
// validating the URL) defeats DNS rebinding and redirects to internal hosts.
func safeDialControl(network, address string, _ syscall.RawConn) error {
	switch network {
	case "tcp4", "tcp6":
	default:
		return fmt.Errorf("%w: network %q", errBlockedDestination, network)
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("%w: %v", errBlockedDestination, err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || isBlockedIP(ip) {
		return fmt.Errorf("%w: %s", errBlockedDestination, host)
	}
	return nil
}

// checkFetchURL validates scheme and host and rejects hosts that resolve to a
// blocked address. It gives a clean early error; the dialer enforces the same
// policy again at connect time.
func checkFetchURL(ctx context.Context, u *url.URL, resolver *net.Resolver) error {
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("URL has no host")
	}
	if u.User != nil {
		return errors.New("URLs with credentials are not allowed")
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: %s", errBlockedDestination, host)
		}
		return nil
	}
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ips, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("resolve %s: no addresses", host)
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("%w: %s resolves to %s", errBlockedDestination, host, ip)
		}
	}
	return nil
}

// newSafeHTTPClient returns a client that can only reach public unicast
// addresses, ignores proxy env vars (a proxy would hide the real target from
// the dial check), and follows at most maxFetchRedirects http(s) redirects.
func newSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   safeDialControl,
	}
	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxFetchRedirects {
				return fmt.Errorf("stopped after %d redirects", maxFetchRedirects)
			}
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return fmt.Errorf("redirect to unsupported scheme %q", req.URL.Scheme)
			}
			return nil
		},
	}
}
