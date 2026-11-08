package acquire

import (
	"bytes"
	"net/http"
	"strings"
)

// strongChallengeMarkers identify interstitial challenge pages themselves.
// Weak markers such as Cloudflare's injected challenge-platform script are
// excluded because they also appear on ordinary pages.
var strongChallengeMarkers = [][]byte{
	[]byte("<title>just a moment...</title>"),
	[]byte("attention required! | cloudflare"),
	[]byte("window._cf_chl_opt"),
	[]byte("captcha-delivery.com"),
	[]byte("px-captcha"),
	[]byte("_incapsula_resource"),
}

// IsBotChallenge reports whether a response is a bot challenge rather than
// content. Challenges are never solved or bypassed; they pause the site.
func IsBotChallenge(header http.Header, body []byte) bool {
	if strings.EqualFold(header.Get("Cf-Mitigated"), "challenge") {
		return true
	}
	switch strings.ToLower(header.Get("X-Amzn-Waf-Action")) {
	case "captcha", "challenge":
		return true
	}
	if len(body) == 0 {
		return false
	}
	sample := bytes.ToLower(body[:min(len(body), 64<<10)])
	for _, marker := range strongChallengeMarkers {
		if bytes.Contains(sample, marker) {
			return true
		}
	}
	return false
}
