package safefetch

import (
	"fmt"
	"net/netip"
	"net/url"
	"strings"
)

// ValidateURL enforces the structural fetch policy: absolute default-port
// HTTPS with a DNS host name, no credentials, and no fragment.
func ValidateURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("URL %q must use https", raw)
	}
	if parsed.Opaque != "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("URL %q must be absolute and include a host", raw)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("URL %q must not carry credentials", raw)
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("URL %q must not carry a fragment", raw)
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return nil, fmt.Errorf("URL %q uses non-default port %q", raw, port)
	}
	host := parsed.Hostname()
	if strings.Contains(host, "%") {
		return nil, fmt.Errorf("URL %q must not use an IPv6 zone", raw)
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return nil, fmt.Errorf("URL %q must use a host name, not an IP literal", raw)
	}
	return parsed, nil
}

// NormalizeHost lowercases a host name and drops a trailing root dot.
func NormalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(host), ".")
}

// HostAllowed reports whether u's host is exactly one of hosts. Subdomains are
// never implied.
func HostAllowed(u *url.URL, hosts []string) bool {
	host := NormalizeHost(u.Hostname())
	for _, allowed := range hosts {
		if NormalizeHost(allowed) == host {
			return true
		}
	}
	return false
}

// blockedPrefixes lists special-purpose ranges that are not covered by the
// netip predicates used in IsPublicAddr (CGNAT, documentation, benchmarking,
// translation, and reserved ranges).
var blockedPrefixes = mustPrefixes(
	"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
	"192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
	"240.0.0.0/4", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64",
	"2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20", "fec0::/10",
)

// IsPublicAddr reports whether addr is a globally routable unicast address.
func IsPublicAddr(addr netip.Addr) bool {
	addr = addr.Unmap()
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() ||
		addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() ||
		addr.IsUnspecified() || addr.Zone() != "" {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

func mustPrefixes(raw ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(raw))
	for _, value := range raw {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}
