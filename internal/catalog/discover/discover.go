// Package discover finds candidate recipe pages from a site's registered
// sitemaps only (CATALOG_INGEST_SPEC.md §7.2). It never follows links found
// in page bodies.
package discover

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// Limits from the sitemap fetch profile.
const (
	// MaxDepth is the deepest sitemap level fetched; registered sitemaps are
	// level 1, so an index may nest two further levels.
	MaxDepth = 3
	// MaxURLs is the most <loc> entries one sitemap file may hold.
	MaxURLs = 50_000
	// MaxDecodedBytes bounds a sitemap after both transfer and file gzip.
	MaxDecodedBytes = 50 << 20
)

// SitemapGetter fetches one sitemap politely (implemented by *acquire.Client).
type SitemapGetter interface {
	GetSitemap(ctx context.Context, site acquire.Site, rawURL string) (acquire.Result, error)
}

// URLFilter applies a site's include/exclude patterns.
type URLFilter interface {
	Allow(u *url.URL) bool
}

// Site is the subset of a registry entry discovery needs.
type Site struct {
	acquire.Site
	Sitemaps []string
	// Filter applies include/exclude patterns; nil allows everything.
	Filter URLFilter
}

// Entry is one discovered page URL.
type Entry struct {
	URL     string
	LastMod string
	Sitemap string
}

// SitemapFile reports what happened to one sitemap URL.
type SitemapFile struct {
	URL     string
	Depth   int
	Outcome acquire.Outcome
	// Type is "sitemapindex" or "urlset" when parsed.
	Type  string
	Locs  int
	Error string
}

// Report is the result of discovering one site.
type Report struct {
	Entries []Entry
	Files   []SitemapFile
	// UnregisteredHost counts page and child-sitemap entries whose host is
	// not one of the site's hosts; they are ignored.
	UnregisteredHost int
	// Invalid counts entries that are not safe absolute HTTPS URLs.
	Invalid int
	// Excluded counts on-host page entries removed by the filter.
	Excluded int
	// Duplicates counts repeated page URLs.
	Duplicates int
	// Cycles counts child sitemaps already visited.
	Cycles int
	// TooDeep counts child sitemaps beyond MaxDepth, not fetched.
	TooDeep int
	// Paused is set when the site paused during discovery; discovery stops.
	Paused *acquire.Pause
}

// Discover walks site's registered sitemaps breadth-first.
func Discover(ctx context.Context, getter SitemapGetter, site Site) (Report, error) {
	report := Report{}
	type queued struct {
		url   string
		depth int
	}
	queue := make([]queued, 0, len(site.Sitemaps))
	visited := map[string]bool{}
	for _, raw := range site.Sitemaps {
		key := normalize(raw)
		if visited[key] {
			continue
		}
		visited[key] = true
		queue = append(queue, queued{url: raw, depth: 1})
	}
	seenPages := map[string]bool{}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		file := SitemapFile{URL: current.url, Depth: current.depth}
		result, err := getter.GetSitemap(ctx, site.Site, current.url)
		if err != nil {
			return report, fmt.Errorf("fetch sitemap %s: %w", current.url, err)
		}
		file.Outcome = result.Outcome
		if result.Pause != nil {
			report.Paused = result.Pause
		}
		if result.Outcome != acquire.OutcomeFetched {
			if result.Err != nil {
				file.Error = result.Err.Error()
			}
			report.Files = append(report.Files, file)
			if result.Outcome == acquire.OutcomePaused {
				return report, nil
			}
			continue
		}
		if final := normalize(result.Response.FinalURL); final != "" {
			visited[final] = true
		}
		parsed, err := Parse(result.Response.Body)
		if err != nil {
			file.Outcome = acquire.OutcomeRejected
			file.Error = err.Error()
			report.Files = append(report.Files, file)
			continue
		}
		file.Type, file.Locs = parsed.Type, len(parsed.Locs)
		report.Files = append(report.Files, file)
		for _, loc := range parsed.Locs {
			target, err := safefetch.ValidateURL(loc.Loc)
			if err != nil {
				report.Invalid++
				continue
			}
			if !safefetch.HostAllowed(target, site.Hosts) {
				report.UnregisteredHost++
				continue
			}
			key := normalize(target.String())
			if parsed.Type == "sitemapindex" {
				switch {
				case visited[key]:
					report.Cycles++
				case current.depth+1 > MaxDepth:
					report.TooDeep++
				default:
					visited[key] = true
					queue = append(queue, queued{url: target.String(), depth: current.depth + 1})
				}
				continue
			}
			if site.Filter != nil && !site.Filter.Allow(target) {
				report.Excluded++
				continue
			}
			if seenPages[key] {
				report.Duplicates++
				continue
			}
			seenPages[key] = true
			report.Entries = append(report.Entries, Entry{URL: target.String(), LastMod: loc.LastMod, Sitemap: current.url})
		}
	}
	return report, nil
}

func normalize(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.Host = safefetch.NormalizeHost(parsed.Host)
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	return parsed.String()
}

// Loc is one <loc> with its optional <lastmod>.
type Loc struct {
	Loc     string
	LastMod string
}

// Sitemap is a parsed sitemap file.
type Sitemap struct {
	Type string
	Locs []Loc
}

// ErrTooManyURLs means a sitemap file exceeded MaxURLs.
var ErrTooManyURLs = errors.New("sitemap exceeds URL limit")

// Parse decodes one sitemap file, gunzipping it when it is a gzip file, and
// enforces MaxDecodedBytes and MaxURLs.
func Parse(body []byte) (Sitemap, error) {
	reader := io.Reader(bytes.NewReader(body))
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return Sitemap{}, fmt.Errorf("open gzip sitemap: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	limited := &limitReader{reader: reader, remaining: MaxDecodedBytes}
	decoder := xml.NewDecoder(limited)
	var sitemap Sitemap
	var element string
	var entry *Loc
	depth := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if limited.exceeded {
				return Sitemap{}, fmt.Errorf("sitemap exceeds %d decoded bytes", MaxDecodedBytes)
			}
			return Sitemap{}, fmt.Errorf("parse sitemap XML: %w", err)
		}
		switch typed := token.(type) {
		case xml.StartElement:
			depth++
			name := typed.Name.Local
			switch {
			case depth == 1:
				if name != "sitemapindex" && name != "urlset" {
					return Sitemap{}, fmt.Errorf("unexpected sitemap root <%s>", name)
				}
				sitemap.Type = name
			case depth == 2 && (name == "sitemap" || name == "url"):
				entry = &Loc{}
			}
			element = name
		case xml.CharData:
			if entry == nil || depth != 3 {
				continue
			}
			switch element {
			case "loc":
				entry.Loc += strings.TrimSpace(string(typed))
			case "lastmod":
				entry.LastMod += strings.TrimSpace(string(typed))
			}
		case xml.EndElement:
			if depth == 2 && entry != nil {
				if entry.Loc != "" {
					if len(sitemap.Locs) == MaxURLs {
						return Sitemap{}, fmt.Errorf("%w of %d", ErrTooManyURLs, MaxURLs)
					}
					sitemap.Locs = append(sitemap.Locs, *entry)
				}
				entry = nil
			}
			depth--
			element = ""
		}
	}
	if sitemap.Type == "" {
		return Sitemap{}, fmt.Errorf("sitemap has no root element")
	}
	return sitemap, nil
}

type limitReader struct {
	reader    io.Reader
	remaining int64
	exceeded  bool
}

func (limited *limitReader) Read(buffer []byte) (int, error) {
	if limited.remaining <= 0 {
		// Probe for one more byte to tell "exactly at limit" from "over".
		var probe [1]byte
		count, err := limited.reader.Read(probe[:])
		if count > 0 {
			limited.exceeded = true
			return 0, errors.New("decoded size limit exceeded")
		}
		return 0, err
	}
	if int64(len(buffer)) > limited.remaining {
		buffer = buffer[:limited.remaining]
	}
	count, err := limited.reader.Read(buffer)
	limited.remaining -= int64(count)
	return count, err
}

// PatternFilter is a URLFilter from regular expressions matched against the
// URL path plus query. A URL is allowed when it matches at least one include
// pattern (or there are none) and no exclude pattern.
type PatternFilter struct {
	Include []*regexp.Regexp
	Exclude []*regexp.Regexp
}

// NewPatternFilter compiles include and exclude patterns.
func NewPatternFilter(include, exclude []string) (*PatternFilter, error) {
	compile := func(patterns []string) ([]*regexp.Regexp, error) {
		compiled := make([]*regexp.Regexp, 0, len(patterns))
		for _, pattern := range patterns {
			expression, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("compile pattern %q: %w", pattern, err)
			}
			compiled = append(compiled, expression)
		}
		return compiled, nil
	}
	includes, err := compile(include)
	if err != nil {
		return nil, err
	}
	excludes, err := compile(exclude)
	if err != nil {
		return nil, err
	}
	return &PatternFilter{Include: includes, Exclude: excludes}, nil
}

// Allow implements URLFilter.
func (filter *PatternFilter) Allow(u *url.URL) bool {
	target := u.EscapedPath()
	if u.RawQuery != "" {
		target += "?" + u.RawQuery
	}
	included := len(filter.Include) == 0
	for _, expression := range filter.Include {
		if expression.MatchString(target) {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, expression := range filter.Exclude {
		if expression.MatchString(target) {
			return false
		}
	}
	return true
}
