package recipepilot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var sourceIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ReadManifest decodes and validates one YAML document and returns its SHA-256.
func ReadManifest(data []byte) (Manifest, string, error) {
	var manifest Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, "", fmt.Errorf("decode manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return Manifest{}, "", fmt.Errorf("decode manifest: multiple YAML documents are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return Manifest{}, "", fmt.Errorf("decode trailing manifest content: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, "", err
	}
	digest := sha256.Sum256(data)
	return manifest, hex.EncodeToString(digest[:]), nil
}

// ValidateManifest checks the complete manifest before any network request.
func ValidateManifest(manifest Manifest) error {
	if manifest.Version != ManifestVersion {
		return fmt.Errorf("manifest version %d is unsupported; want %d", manifest.Version, ManifestVersion)
	}
	if len(manifest.Sources) == 0 {
		return fmt.Errorf("manifest must contain at least one source")
	}
	ids := make(map[string]struct{}, len(manifest.Sources))
	urls := make(map[string]struct{}, len(manifest.Sources))
	for index, source := range manifest.Sources {
		if !sourceIDPattern.MatchString(source.ID) {
			return fmt.Errorf("source %d id %q must be lowercase kebab-case", index+1, source.ID)
		}
		if _, exists := ids[source.ID]; exists {
			return fmt.Errorf("source %d duplicates id %q", index+1, source.ID)
		}
		ids[source.ID] = struct{}{}
		parsed, err := ValidateFetchURL(source.URL)
		if err != nil {
			return fmt.Errorf("source %q URL: %w", source.ID, err)
		}
		canonical := parsed.String()
		if _, exists := urls[canonical]; exists {
			return fmt.Errorf("source %q duplicates URL %q", source.ID, source.URL)
		}
		urls[canonical] = struct{}{}
	}
	return nil
}

// ValidateFetchURL enforces the pilot's structural HTTPS URL policy.
func ValidateFetchURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if parsed.Scheme != "https" {
		return nil, fmt.Errorf("must use https")
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("must be absolute and include a host")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("credentials are not allowed")
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("fragments are not allowed")
	}
	if port := parsed.Port(); port != "" && port != "443" {
		return nil, fmt.Errorf("port %q is not allowed", port)
	}
	if strings.Contains(parsed.Hostname(), "%") {
		return nil, fmt.Errorf("IPv6 zones are not allowed")
	}
	return parsed, nil
}

func isPublicIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return false
	}
	blocked := []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"240.0.0.0/4", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64",
		"2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20",
	}
	for _, raw := range blocked {
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			panic(err)
		}
		if network.Contains(ip) {
			return false
		}
	}
	return true
}
