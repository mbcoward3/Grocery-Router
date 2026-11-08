// Package robots parses robots.txt (RFC 9309 plus Crawl-delay) and decides
// whether a URL may be fetched. Fetching robots.txt lives in the acquisition
// layer so it shares rate limiting and attempt recording with other requests.
package robots

import (
	"bufio"
	"bytes"
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TTL is how long a fetched robots policy stays fresh.
const TTL = 24 * time.Hour

// Mode is how a host's robots outcome restricts fetching.
type Mode string

// Policy modes.
const (
	// ModeRules applies parsed rules (robots.txt answered 2xx).
	ModeRules Mode = "rules"
	// ModeAllowAll means robots.txt answered a 4xx other than 429.
	ModeAllowAll Mode = "allow_all"
	// ModeDisallowAll means robots.txt answered 429 or 5xx or was unreachable.
	ModeDisallowAll Mode = "disallow_all"
)

// Rule is one Allow or Disallow line from the matched group(s).
type Rule struct {
	Allow   bool   `json:"allow"`
	Pattern string `json:"pattern"`
}

// Policy is the robots outcome for one host.
type Policy struct {
	Host       string        `json:"host"`
	Mode       Mode          `json:"mode"`
	Rules      []Rule        `json:"rules,omitempty"`
	CrawlDelay time.Duration `json:"crawl_delay,omitempty"`
	Status     int           `json:"status"`
	FetchedAt  time.Time     `json:"fetched_at"`
}

// Fresh reports whether the policy is still within TTL at now.
func (policy Policy) Fresh(now time.Time) bool {
	return !policy.FetchedAt.IsZero() && now.Before(policy.FetchedAt.Add(TTL))
}

// Allowed reports whether u may be fetched under the policy.
func (policy Policy) Allowed(u *url.URL) bool {
	switch policy.Mode {
	case ModeAllowAll:
		return true
	case ModeRules:
	default:
		return false
	}
	target := u.EscapedPath()
	if target == "" {
		target = "/"
	}
	if target == "/robots.txt" {
		return true
	}
	if u.RawQuery != "" {
		target += "?" + u.RawQuery
	}
	best, allowed := -1, true
	for _, rule := range policy.Rules {
		if !matches(rule.Pattern, target) {
			continue
		}
		length := len(rule.Pattern)
		if length > best || (length == best && rule.Allow) {
			best, allowed = length, rule.Allow
		}
	}
	return allowed
}

// Parse reads robots.txt and returns the rules and crawl delay of the groups
// matching productToken, or of the "*" groups when none match.
func Parse(body []byte, productToken string) ([]Rule, time.Duration) {
	type group struct {
		agents     []string
		rules      []Rule
		crawlDelay time.Duration
		hasDelay   bool
	}
	var groups []*group
	var current *group
	inAgents := false
	scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))))
	scanner.Buffer(make([]byte, 0, 64<<10), len(body)+1)
	for scanner.Scan() {
		line := scanner.Text()
		if index := strings.IndexByte(line, '#'); index >= 0 {
			line = line[:index]
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch key {
		case "user-agent":
			if current == nil || !inAgents {
				current = &group{}
				groups = append(groups, current)
			}
			current.agents = append(current.agents, value)
			inAgents = true
		case "allow", "disallow":
			inAgents = false
			if current == nil || value == "" {
				continue
			}
			current.rules = append(current.rules, Rule{Allow: key == "allow", Pattern: value})
		case "crawl-delay":
			inAgents = false
			if current == nil {
				continue
			}
			if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 && seconds < 86400 {
				current.crawlDelay = time.Duration(seconds * float64(time.Second))
				current.hasDelay = true
			}
		default:
			// Sitemap and unknown lines neither start nor end a group.
		}
	}
	token := strings.ToLower(productToken)
	collect := func(match func(string) bool) ([]Rule, time.Duration, bool) {
		var rules []Rule
		var delay time.Duration
		found := false
		for _, candidate := range groups {
			for _, agent := range candidate.agents {
				if match(agent) {
					found = true
					rules = append(rules, candidate.rules...)
					if candidate.hasDelay && candidate.crawlDelay > delay {
						delay = candidate.crawlDelay
					}
					break
				}
			}
		}
		return rules, delay, found
	}
	if rules, delay, found := collect(func(agent string) bool { return agentToken(agent) == token }); found {
		return rules, delay
	}
	rules, delay, _ := collect(func(agent string) bool { return agent == "*" })
	return rules, delay
}

func agentToken(agent string) string {
	agent = strings.ToLower(strings.TrimSpace(agent))
	if index := strings.IndexAny(agent, "/ "); index >= 0 {
		agent = agent[:index]
	}
	return agent
}

// matches applies RFC 9309 pattern matching: "*" matches any sequence and a
// trailing "$" anchors the end.
func matches(pattern, target string) bool {
	anchored := strings.HasSuffix(pattern, "$")
	if anchored {
		pattern = strings.TrimSuffix(pattern, "$")
	}
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(target, parts[0]) {
		return false
	}
	position := len(parts[0])
	for index, part := range parts[1:] {
		last := index == len(parts)-2
		if last && anchored {
			return len(target)-position >= len(part) && strings.HasSuffix(target, part)
		}
		found := strings.Index(target[position:], part)
		if found < 0 {
			return false
		}
		position += found + len(part)
	}
	if anchored {
		return position == len(target)
	}
	return true
}

// Cache stores robots policies by host. The database layer implements it;
// freshness is decided by the caller using Policy.Fresh.
type Cache interface {
	Get(ctx context.Context, host string) (Policy, bool, error)
	Put(ctx context.Context, policy Policy) error
}

// MemoryCache is an in-memory Cache.
type MemoryCache struct {
	mu       sync.Mutex
	policies map[string]Policy
}

// NewMemoryCache returns an empty MemoryCache.
func NewMemoryCache() *MemoryCache { return &MemoryCache{policies: map[string]Policy{}} }

// Get returns the stored policy for host.
func (cache *MemoryCache) Get(_ context.Context, host string) (Policy, bool, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	policy, ok := cache.policies[host]
	return policy, ok, nil
}

// Put stores policy under its host.
func (cache *MemoryCache) Put(_ context.Context, policy Policy) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.policies[policy.Host] = policy
	return nil
}
