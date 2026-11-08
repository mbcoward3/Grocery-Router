// Package registry strictly parses and validates the shared catalog site registry
// (catalog/sites.yaml, CATALOG_INGEST_SPEC.md §7.1) and syncs it into ingest.sites.
package registry

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the registry location relative to the repository root.
const DefaultPath = "catalog/sites.yaml"

// MinimumInterval is the floor between requests to one host (§7.3). A site may only
// configure a stricter interval.
const MinimumInterval = 5 * time.Second

// Measurement conventions (§5.8).
const (
	ConventionUS     = "us"
	ConventionMetric = "metric"
)

var (
	siteIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	hostPattern   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)+$`)
)

// Registry is a validated site registry in registry order.
type Registry struct {
	Sites []Site
}

// Site is one validated registry entry.
type Site struct {
	ID                    string
	Name                  string
	Hosts                 []string
	Sitemaps              []string
	Include               []string
	Exclude               []string
	MinInterval           time.Duration // zero means MinimumInterval
	MeasurementConvention string
	Active                bool
	Eligibility           *Eligibility

	include []*regexp.Regexp
	exclude []*regexp.Regexp
}

// Eligibility records the review that allows a site to be active.
type Eligibility struct {
	IntendedUse string
	ReviewedOn  time.Time
	Notes       string
}

type fileRegistry struct {
	Version *int       `yaml:"version"`
	Sites   []fileSite `yaml:"sites"`
}

type fileSite struct {
	ID                    string           `yaml:"id"`
	Name                  string           `yaml:"name"`
	Hosts                 []string         `yaml:"hosts"`
	Sitemaps              []string         `yaml:"sitemaps"`
	Include               []string         `yaml:"include"`
	Exclude               []string         `yaml:"exclude"`
	MinInterval           string           `yaml:"min_interval"`
	MeasurementConvention string           `yaml:"measurement_convention"`
	Active                *bool            `yaml:"active"`
	Eligibility           *fileEligibility `yaml:"eligibility"`
}

type fileEligibility struct {
	IntendedUse string `yaml:"intended_use"`
	ReviewedOn  string `yaml:"reviewed_on"`
	Notes       string `yaml:"notes"`
}

// ReadFile parses and validates a registry file.
func ReadFile(path string) (Registry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Registry{}, fmt.Errorf("read site registry: %w", err)
	}
	registry, err := Parse(content)
	if err != nil {
		return Registry{}, fmt.Errorf("%s: %w", path, err)
	}
	return registry, nil
}

// Parse strictly decodes and validates a registry document. Unknown fields, a second YAML
// document, and every rule in §7.1 are rejected; all violations are reported together.
func Parse(content []byte) (Registry, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	var file fileRegistry
	if err := decoder.Decode(&file); err != nil {
		if errors.Is(err, io.EOF) {
			return Registry{}, errors.New("site registry is empty")
		}
		return Registry{}, fmt.Errorf("decode site registry: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Registry{}, errors.New("site registry must be a single YAML document")
	}
	return validate(file)
}

func validate(file fileRegistry) (Registry, error) {
	var problems []error
	report := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf(format, args...))
	}
	if file.Version == nil || *file.Version != 1 {
		report("version must be 1")
	}
	if len(file.Sites) == 0 {
		report("sites must list at least one site")
	}

	registry := Registry{Sites: make([]Site, 0, len(file.Sites))}
	siteIDs := map[string]bool{}
	hostOwners := map[string]string{}
	for index, raw := range file.Sites {
		label := fmt.Sprintf("sites[%d]", index)
		if raw.ID != "" {
			label = fmt.Sprintf("site %q", raw.ID)
		}
		siteReport := func(format string, args ...any) {
			report("%s: %s", label, fmt.Sprintf(format, args...))
		}
		site := Site{
			ID:                    raw.ID,
			Name:                  strings.TrimSpace(raw.Name),
			Hosts:                 nonNil(raw.Hosts),
			Sitemaps:              nonNil(raw.Sitemaps),
			Include:               nonNil(raw.Include),
			Exclude:               nonNil(raw.Exclude),
			MeasurementConvention: raw.MeasurementConvention,
		}

		switch {
		case raw.ID == "":
			siteReport("id is required")
		case !siteIDPattern.MatchString(raw.ID) || len(raw.ID) > 64:
			siteReport("id must be a lowercase slug of at most 64 characters")
		case siteIDs[raw.ID]:
			siteReport("duplicate site id")
		}
		siteIDs[raw.ID] = true
		if site.Name == "" {
			siteReport("name is required")
		}

		if len(raw.Hosts) == 0 {
			siteReport("hosts must list at least one host")
		}
		for _, host := range raw.Hosts {
			if !hostPattern.MatchString(host) {
				siteReport("host %q must be a lowercase DNS name without scheme, port, or path", host)
				continue
			}
			if owner, claimed := hostOwners[host]; claimed {
				if owner == raw.ID {
					siteReport("host %q is listed twice", host)
				} else {
					siteReport("host %q is already claimed by site %q", host, owner)
				}
				continue
			}
			hostOwners[host] = raw.ID
		}

		seenSitemaps := map[string]bool{}
		for _, sitemap := range raw.Sitemaps {
			if err := validateSiteURL(sitemap, raw.Hosts); err != nil {
				siteReport("sitemap %q: %v", sitemap, err)
			}
			if seenSitemaps[sitemap] {
				siteReport("sitemap %q is listed twice", sitemap)
			}
			seenSitemaps[sitemap] = true
		}

		var err error
		if site.include, err = compilePatterns(raw.Include); err != nil {
			siteReport("include: %v", err)
		}
		if site.exclude, err = compilePatterns(raw.Exclude); err != nil {
			siteReport("exclude: %v", err)
		}

		if raw.MinInterval != "" {
			interval, err := time.ParseDuration(raw.MinInterval)
			switch {
			case err != nil:
				siteReport("min_interval %q is not a duration such as 10s", raw.MinInterval)
			case interval <= MinimumInterval:
				siteReport("min_interval must be stricter than the %s floor", MinimumInterval)
			case interval%time.Millisecond != 0 || interval > time.Hour:
				siteReport("min_interval must be whole milliseconds and at most 1h")
			default:
				site.MinInterval = interval
			}
		}

		if raw.MeasurementConvention != ConventionUS && raw.MeasurementConvention != ConventionMetric {
			siteReport("measurement_convention must be %q or %q", ConventionUS, ConventionMetric)
		}

		if raw.Active == nil {
			siteReport("active must be set explicitly")
		} else {
			site.Active = *raw.Active
		}
		if raw.Eligibility != nil {
			eligibility, problems := validateEligibility(*raw.Eligibility)
			for _, problem := range problems {
				siteReport("eligibility: %s", problem)
			}
			site.Eligibility = eligibility
		}
		if site.Active && raw.Eligibility == nil {
			siteReport("an active site requires an eligibility record")
		}
		if site.Active && len(raw.Sitemaps) == 0 {
			siteReport("an active site requires at least one sitemap")
		}
		registry.Sites = append(registry.Sites, site)
	}
	if len(problems) > 0 {
		return Registry{}, fmt.Errorf("invalid site registry:\n%w", errors.Join(problems...))
	}
	return registry, nil
}

func validateEligibility(raw fileEligibility) (*Eligibility, []string) {
	var problems []string
	eligibility := &Eligibility{
		IntendedUse: strings.TrimSpace(raw.IntendedUse),
		Notes:       strings.TrimSpace(raw.Notes),
	}
	if eligibility.IntendedUse == "" {
		problems = append(problems, "intended_use is required")
	}
	reviewedOn, err := time.Parse(time.DateOnly, raw.ReviewedOn)
	if err != nil {
		problems = append(problems, "reviewed_on must be a YYYY-MM-DD date")
	}
	eligibility.ReviewedOn = reviewedOn
	return eligibility, problems
}

// validateSiteURL requires a default-port HTTPS URL without credentials on one of hosts.
func validateSiteURL(raw string, hosts []string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return errors.New("is not a URL")
	}
	if parsed.Scheme != "https" {
		return errors.New("must use https")
	}
	if parsed.User != nil || parsed.Port() != "" || parsed.Fragment != "" {
		return errors.New("must not contain credentials, a port, or a fragment")
	}
	if !slices.Contains(hosts, parsed.Host) {
		return fmt.Errorf("host %q is not one of the site's hosts", parsed.Host)
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") {
		return errors.New("must have an absolute path")
	}
	return nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			return nil, errors.New("patterns must not be empty")
		}
		expression, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		compiled = append(compiled, expression)
	}
	return compiled, nil
}

// Interval is the minimum time between requests to one of the site's hosts.
func (site Site) Interval() time.Duration {
	return max(site.MinInterval, MinimumInterval)
}

// HasHost reports whether host is explicitly registered to the site.
func (site Site) HasHost(host string) bool {
	return slices.Contains(site.Hosts, strings.ToLower(host))
}

// AllowsPage reports whether a discovered page URL belongs to the site and passes its
// include and exclude patterns. Patterns match the URL's escaped path and query.
func (site Site) AllowsPage(page *url.URL) bool {
	if page == nil || page.Scheme != "https" || page.Port() != "" || page.User != nil || !site.HasHost(page.Hostname()) {
		return false
	}
	target := page.RequestURI()
	for _, expression := range site.exclude {
		if expression.MatchString(target) {
			return false
		}
	}
	if len(site.include) == 0 {
		return true
	}
	for _, expression := range site.include {
		if expression.MatchString(target) {
			return true
		}
	}
	return false
}

// Site returns the registry entry with id.
func (registry Registry) Site(id string) (Site, bool) {
	for _, site := range registry.Sites {
		if site.ID == id {
			return site, true
		}
	}
	return Site{}, false
}
