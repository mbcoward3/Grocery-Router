package registry

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const validRegistry = `version: 1
sites:
  - id: example-kitchen
    name: Example Kitchen
    hosts: [www.example.com, example.com]
    sitemaps: [https://www.example.com/sitemap_index.xml]
    include: ['^/recipes/[a-z0-9-]+/$']
    exclude: ['^/recipes/roundup-']
    min_interval: 10s
    measurement_convention: us
    active: true
    eligibility:
      intended_use: private household use
      reviewed_on: 2026-11-08
      notes: reviewed
  - id: other-kitchen
    name: Other Kitchen
    hosts: [other.example.org]
    sitemaps: []
    measurement_convention: metric
    active: false
`

func TestParseValidRegistry(t *testing.T) {
	registry, err := Parse([]byte(validRegistry))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(registry.Sites) != 2 {
		t.Fatalf("sites = %d, want 2", len(registry.Sites))
	}
	site := registry.Sites[0]
	if site.ID != "example-kitchen" || site.Name != "Example Kitchen" || !site.Active {
		t.Fatalf("unexpected site %+v", site)
	}
	if site.Interval() != 10*time.Second {
		t.Errorf("Interval = %s, want 10s", site.Interval())
	}
	if site.Eligibility == nil || site.Eligibility.ReviewedOn.Format(time.DateOnly) != "2026-11-08" {
		t.Errorf("eligibility = %+v", site.Eligibility)
	}
	other, ok := registry.Site("other-kitchen")
	if !ok || other.Active || other.Eligibility != nil || other.Interval() != MinimumInterval {
		t.Errorf("other site = %+v", other)
	}
}

func TestAllowsPage(t *testing.T) {
	registry, err := Parse([]byte(validRegistry))
	if err != nil {
		t.Fatal(err)
	}
	site := registry.Sites[0]
	for raw, want := range map[string]bool{
		"https://www.example.com/recipes/lentil-soup/":       true,
		"https://example.com/recipes/lentil-soup/":           true,
		"https://WWW.EXAMPLE.COM/recipes/lentil-soup/":       true,
		"https://www.example.com/recipes/roundup-soups/":     false, // excluded
		"https://www.example.com/about/":                     false, // not included
		"https://cdn.example.com/recipes/lentil-soup/":       false, // implicit subdomain
		"http://www.example.com/recipes/lentil-soup/":        false, // not HTTPS
		"https://www.example.com:8443/recipes/lentil-soup/":  false, // non-default port
		"https://www.example.com/recipes/lentil-soup/?print": false, // query is matched
	} {
		page, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := site.AllowsPage(page); got != want {
			t.Errorf("AllowsPage(%s) = %v, want %v", raw, got, want)
		}
	}
	if !registry.Sites[1].AllowsPage(mustURL(t, "https://other.example.org/anything")) {
		t.Error("a site without include patterns allows every path on its hosts")
	}
}

func TestParseRejects(t *testing.T) {
	for _, test := range []struct {
		name    string
		edit    func(string) string
		message string
	}{
		{"unknown site field", replace("    active: false\n", "    active: false\n    crawl: fast\n"), "field crawl not found"},
		{"unknown eligibility field", replace("      notes: reviewed\n", "      notes: reviewed\n      approved_by: me\n"), "field approved_by not found"},
		{"unknown top-level field", replace("version: 1\n", "version: 1\nowner: me\n"), "field owner not found"},
		{"second document", func(s string) string { return s + "---\nversion: 1\n" }, "single YAML document"},
		{"missing version", replace("version: 1\n", ""), "version must be 1"},
		{"duplicate id", replace("id: other-kitchen", "id: example-kitchen"), "duplicate site id"},
		{"invalid id", replace("id: other-kitchen", "id: Other_Kitchen"), "lowercase slug"},
		{"host claimed by two sites", replace("hosts: [other.example.org]", "hosts: [example.com]"), `host "example.com" is already claimed by site "example-kitchen"`},
		{"host listed twice", replace("hosts: [other.example.org]", "hosts: [other.example.org, other.example.org]"), "listed twice"},
		{"host with scheme", replace("hosts: [other.example.org]", "hosts: [https://other.example.org]"), "lowercase DNS name"},
		{"host with port", replace("hosts: [other.example.org]", "hosts: ['other.example.org:443']"), "lowercase DNS name"},
		{"no hosts", replace("hosts: [other.example.org]", "hosts: []"), "at least one host"},
		{"non-HTTPS sitemap", replace("https://www.example.com/sitemap_index.xml", "http://www.example.com/sitemap_index.xml"), "must use https"},
		{"sitemap with port", replace("https://www.example.com/sitemap_index.xml", "https://www.example.com:8443/sitemap.xml"), "port"},
		{"sitemap on unregistered host", replace("https://www.example.com/sitemap_index.xml", "https://cdn.example.com/sitemap.xml"), "not one of the site's hosts"},
		{"active site without eligibility", removeEligibility, "active site requires an eligibility record"},
		{"active site without sitemap", replace("sitemaps: [https://www.example.com/sitemap_index.xml]", "sitemaps: []"), "requires at least one sitemap"},
		{"eligibility without intended use", replace("intended_use: private household use", "intended_use: ' '"), "intended_use is required"},
		{"eligibility without review date", replace("reviewed_on: 2026-11-08", "reviewed_on: last week"), "reviewed_on must be a YYYY-MM-DD date"},
		{"missing active flag", replace("    active: false\n", ""), "active must be set explicitly"},
		{"bad convention", replace("measurement_convention: metric", "measurement_convention: imperial"), "measurement_convention must be"},
		{"interval not stricter", replace("min_interval: 10s", "min_interval: 5s"), "stricter than the 5s floor"},
		{"interval not a duration", replace("min_interval: 10s", "min_interval: ten"), "not a duration"},
		{"invalid pattern", replace(`'^/recipes/roundup-'`, `'^/recipes/(['`), "exclude: pattern"},
		{"missing name", replace("name: Other Kitchen", "name: ''"), "name is required"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.edit(validRegistry)))
			if err == nil {
				t.Fatalf("Parse accepted an invalid registry")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error %q does not contain %q", err, test.message)
			}
		})
	}
}

func TestParseReportsEveryProblem(t *testing.T) {
	edited := replace("id: other-kitchen", "id: example-kitchen")(validRegistry)
	edited = replace("measurement_convention: metric", "measurement_convention: imperial")(edited)
	_, err := Parse([]byte(edited))
	if err == nil || !strings.Contains(err.Error(), "duplicate site id") || !strings.Contains(err.Error(), "measurement_convention") {
		t.Fatalf("error = %v, want both problems", err)
	}
}

func TestRepositoryRegistryIsValid(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..")
	registry, err := ReadFile(filepath.Join(root, filepath.FromSlash(DefaultPath)))
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.Sites) != 13 {
		t.Errorf("sites = %d, want the 13 pilot origins", len(registry.Sites))
	}
	var active []string
	for _, site := range registry.Sites {
		if site.Active {
			active = append(active, site.ID)
			if site.MeasurementConvention != ConventionUS {
				t.Errorf("active site %s convention = %s, want us", site.ID, site.MeasurementConvention)
			}
		}
	}
	want := "budget-bytes,once-upon-a-chef,natashas-kitchen,cookie-and-kate"
	if got := strings.Join(active, ","); got != want {
		t.Errorf("active sites = %s, want %s", got, want)
	}
}

func replace(old, replacement string) func(string) string {
	return func(document string) string {
		if !strings.Contains(document, old) {
			panic("test edit target missing: " + old)
		}
		return strings.Replace(document, old, replacement, 1)
	}
}

func removeEligibility(document string) string {
	start := strings.Index(document, "    eligibility:\n")
	end := strings.Index(document, "  - id: other-kitchen")
	return document[:start] + document[end:]
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
