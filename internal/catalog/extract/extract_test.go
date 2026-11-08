package extract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/acquire"
	"github.com/mbcoward3/grocery-router/internal/catalog/clock"
	"github.com/mbcoward3/grocery-router/internal/catalog/evidence"
	"github.com/mbcoward3/grocery-router/internal/catalog/extract"
	"github.com/mbcoward3/grocery-router/internal/catalog/fetchtest"
	"github.com/mbcoward3/grocery-router/internal/catalog/ratelimit"
	"github.com/mbcoward3/grocery-router/internal/catalog/robots"
)

var hosts = []string{"www.a.test", "a.test"}

const final = "https://www.a.test/soup/"

const graphPage = `<!doctype html><html><head>
<link rel="canonical" href="/soup">
<script type="application/ld+json; charset=utf-8">{
  "@context": "https://schema.org",
  "@graph": [
    {"@type": "WebPage", "@id": "https://www.a.test/soup#webpage",
     "mainEntity": {"@type": "Recipe", "name": "Nested, not extracted"}},
    {"@type": ["Thing", "Recipe"], "@id": "https://www.a.test/soup#recipe", "name": "Soup",
     "recipeIngredient": ["1 onion"], "customSourceField": {"kept": true}, "ratio": 1.50},
    {"@type": "http://schema.org/Recipe", "name": "Bread"}
  ]
}</script>
<script type="application/ld+json">[{"@type":"Person"},{"@type":"Recipe","name":"Pie"}]</script>
<script type="application/ld+json">{"@type": "Recipe", broken</script>
<script type="application/json">{"@type":"Recipe","name":"Not JSON-LD"}</script>
<script>var x = {"@type":"Recipe"};</script>
</head><body><a href="/other">other</a></body></html>`

func TestExtractGraphArraysAndMalformedBlocks(t *testing.T) {
	result, err := extract.Extract([]byte(graphPage), final, hosts)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != extract.OutcomeRecipes || result.Version != extract.Version {
		t.Fatalf("outcome = %q", result.Outcome)
	}
	var locations []string
	for _, node := range result.Nodes {
		locations = append(locations, node.Location)
	}
	if strings.Join(locations, " ") != "/@graph/1 /@graph/2 /1" {
		t.Fatalf("locations = %v", locations)
	}
	soup := result.Nodes[0]
	if soup.ID != "https://www.a.test/soup#recipe" || soup.BlockIndex != 0 {
		t.Errorf("soup = %+v", soup)
	}
	if !strings.Contains(string(soup.Raw), `"customSourceField": {"kept": true}`) || !strings.Contains(string(soup.Raw), `1.50`) {
		t.Errorf("raw node not preserved verbatim: %s", soup.Raw)
	}
	if len(result.Blocks) != 3 || result.Blocks[2].ParseError == "" || len(result.Warnings) != 1 {
		t.Fatalf("blocks = %+v warnings = %v", result.Blocks, result.Warnings)
	}
	if !strings.Contains(result.Blocks[2].Raw, "broken") {
		t.Errorf("malformed block not kept verbatim")
	}
	if result.CanonicalURL != "https://www.a.test/soup" || result.CanonicalSource != extract.CanonicalFromLink {
		t.Errorf("canonical = %q %q", result.CanonicalURL, result.CanonicalSource)
	}
}

func TestNodeIdentityIgnoresPositionAndFormatting(t *testing.T) {
	first := `<script type="application/ld+json">{"@type":"Recipe","name":"Soup","@id":"#r","n":1.0}</script>`
	second := `<script type="application/ld+json">{"@type":"WebSite"}</script>
<script type="application/ld+json">{"@graph":[{"@type":"Thing"},{ "n" : 1.0, "name":"Soup", "@type":"Recipe", "@id":"#r" }]}</script>`
	a, _ := extract.Extract([]byte(first), final, hosts)
	b, _ := extract.Extract([]byte(second), final, hosts)
	if len(a.Nodes) != 1 || len(b.Nodes) != 1 {
		t.Fatalf("nodes = %d %d", len(a.Nodes), len(b.Nodes))
	}
	if a.Nodes[0].Identity() != b.Nodes[0].Identity() || a.Nodes[0].Location == b.Nodes[0].Location {
		t.Fatalf("identity depends on position: %q vs %q", a.Nodes[0].Identity(), b.Nodes[0].Identity())
	}
	changed := strings.Replace(first, `"n":1.0`, `"n":1`, 1)
	c, _ := extract.Extract([]byte(changed), final, hosts)
	if c.Nodes[0].ContentDigest == a.Nodes[0].ContentDigest {
		t.Fatal("number spelling change kept the same digest")
	}
	noID := strings.Replace(first, `,"@id":"#r"`, "", 1)
	d, _ := extract.Extract([]byte(noID), final, hosts)
	if d.Nodes[0].ID != "" || d.Nodes[0].Identity() == a.Nodes[0].Identity() {
		t.Fatalf("no-@id node = %+v", d.Nodes[0])
	}
	duplicate := first + first
	e, _ := extract.Extract([]byte(duplicate), final, hosts)
	if len(e.Nodes) != 1 || len(e.Blocks) != 2 || len(e.Warnings) != 1 {
		t.Fatalf("duplicate node: %d nodes %v", len(e.Nodes), e.Warnings)
	}
}

func TestCanonicalFallsBackToFinalURL(t *testing.T) {
	recipe := `<script type="application/ld+json">{"@type":"Recipe","name":"Soup"}</script>`
	cases := map[string]string{
		"none":       ``,
		"off site":   `<link rel="canonical" href="https://mirror.test/soup">`,
		"http":       `<link rel="canonical" href="http://www.a.test/soup">`,
		"other port": `<link rel="canonical" href="https://www.a.test:8443/soup">`,
		"conflict":   `<link rel="canonical" href="/soup"><link rel="canonical" href="/soup-2">`,
		"invalid":    `<link rel="canonical" href="https://www.a.test/%zz">`,
	}
	for name, links := range cases {
		result, err := extract.Extract([]byte(links+recipe), final, hosts)
		if err != nil {
			t.Fatal(err)
		}
		if result.CanonicalURL != final || result.CanonicalSource != extract.CanonicalFromFinalURL {
			t.Errorf("%s: canonical = %q (%s)", name, result.CanonicalURL, result.CanonicalSource)
		}
	}
	other, _ := extract.Extract([]byte(`<link rel="canonical" href="https://a.test/soup">`+recipe), final, hosts)
	if other.CanonicalURL != "https://a.test/soup" {
		t.Errorf("second site host refused: %q", other.CanonicalURL)
	}
}

func TestExtractNoRecipe(t *testing.T) {
	result, err := extract.Extract([]byte(`<script type="application/ld+json">{"@type":"WebPage"}</script><p>hi</p>`), final, hosts)
	if err != nil || result.Outcome != extract.OutcomeNoRecipe || len(result.Blocks) != 1 || len(result.Nodes) != 0 {
		t.Fatalf("%v %+v", err, result)
	}
	empty, err := extract.Extract(nil, final, hosts)
	if err != nil || empty.Outcome != extract.OutcomeNoRecipe {
		t.Fatalf("empty body: %v %+v", err, empty)
	}
}

func TestReplayFromStoredEvidenceWithoutNetwork(t *testing.T) {
	mux := fetchtest.NewMux()
	network := fetchtest.New(t, mux, "www.a.test")
	mux.Handle("www.a.test", "/robots.txt", fetchtest.Text(404, "text/plain", ""))
	mux.Handle("www.a.test", "/soup/", fetchtest.Text(http.StatusOK, "text/html", graphPage))
	clk := clock.NewFake(time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC))
	client := &acquire.Client{
		Fetcher: network.Client(t), Limiter: ratelimit.NewMemory(clk), Robots: robots.NewMemoryCache(),
		Pauses: acquire.NewMemoryPauses(), Recorder: &acquire.MemoryRecorder{}, Clock: clk,
	}
	site := acquire.Site{ID: "a", Hosts: hosts}
	fetched, err := client.GetPage(context.Background(), site, 1, final)
	if err != nil || fetched.Outcome != acquire.OutcomeFetched {
		t.Fatalf("fetch: %v %+v", err, fetched)
	}
	record, err := evidence.New(site.ID, fetched.Response)
	if err != nil {
		t.Fatal(err)
	}
	evidenceStore := evidence.NewMemoryStore()
	stored, err := evidenceStore.Insert(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}

	network.Server.Close() // no network from here on
	extractions := extract.NewMemoryStore()
	first, err := extract.Replay(context.Background(), evidenceStore, extractions, stored.ID, hosts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := extract.Replay(context.Background(), evidenceStore, extractions, stored.ID, hosts)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Nodes) != 3 || len(extractions.Results(stored.ID)) != 2 {
		t.Fatalf("nodes = %d saved = %d", len(first.Nodes), len(extractions.Results(stored.ID)))
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Fatal("replay is not deterministic")
	}
}
