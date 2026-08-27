package validation

import (
	"net/url"
	"strings"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
)

// maxPublicIDLength caps input so an absurd slug never reaches the upstream client.
const maxPublicIDLength = 120

// profilePathPrefixes introduce a member profile; "pub" is the legacy format.
var profilePathPrefixes = map[string]bool{"in": true, "pub": true}

// nonProfilePathPrefixes are valid LinkedIn URLs that are not people.
var nonProfilePathPrefixes = map[string]bool{
	"company": true, "school": true, "jobs": true, "posts": true,
	"feed": true, "groups": true, "events": true, "learning": true,
	"showcase": true, "newsletters": true, "pulse": true, "services": true,
}

// ParseProfileURL reduces a LinkedIn profile URL to the slug the API is keyed by.
// Forgiving about scheme, subdomain and query; strict about host and the /in/ segment.
func ParseProfileURL(raw string) (string, *exceptions.ApplicationError) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	// Without a scheme url.Parse reads the whole string as a path.
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	if !isLinkedInHost(parsed.Hostname()) {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	segments := pathSegments(parsed.Path)

	segments = stripLocalePrefix(segments)

	if len(segments) < 2 || !profilePathPrefixes[strings.ToLower(segments[0])] {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	// LinkedIn slugs may be unicode, and arrive percent-encoded.
	publicID, decodeErr := url.PathUnescape(segments[1])
	if decodeErr != nil {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	publicID = strings.TrimSpace(publicID)
	if publicID == "" || len(publicID) > maxPublicIDLength {
		return "", exceptions.New(exceptions.InvalidProfileURL)
	}

	return publicID, nil
}

// IsNonProfileLinkedInURL reports whether raw is a valid LinkedIn URL pointing at
// something other than a member, so the caller can say why it was rejected.
func IsNonProfileLinkedInURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || !isLinkedInHost(parsed.Hostname()) {
		return false
	}

	segments := stripLocalePrefix(pathSegments(parsed.Path))

	return len(segments) > 0 && nonProfilePathPrefixes[strings.ToLower(segments[0])]
}

// isLinkedInHost accepts linkedin.com and its country subdomains, rejecting
// lookalikes like "notlinkedin.com" and "linkedin.com.evil.net".
func isLinkedInHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com")
}

func pathSegments(path string) []string {
	var segments []string
	for _, segment := range strings.Split(path, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

// stripLocalePrefix drops a leading locale segment such as /en-us/in/slug. A
// profile prefix wins: "in" is two alpha chars and would look like a locale.
func stripLocalePrefix(segments []string) []string {
	if len(segments) == 0 {
		return segments
	}
	first := strings.ToLower(segments[0])
	if profilePathPrefixes[first] || nonProfilePathPrefixes[first] {
		return segments
	}
	if isLocaleSegment(first) {
		return segments[1:]
	}
	return segments
}

// isLocaleSegment matches shapes like "en", "en-us", "pt-br".
func isLocaleSegment(segment string) bool {
	segment = strings.ToLower(segment)
	if len(segment) == 2 {
		return isAlpha(segment)
	}
	if len(segment) == 5 && segment[2] == '-' {
		return isAlpha(segment[:2]) && isAlpha(segment[3:])
	}
	return false
}

func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// MergeSections layers a per-request override over the deployment default.
// An unknown name is rejected, so a typo tells the caller instead of returning nothing.
func MergeSections(defaults config.SectionSet, override map[string]bool) (config.SectionSet, *exceptions.ApplicationError) {
	if len(override) == 0 {
		return defaults, nil
	}

	requested := make(config.SectionSet, len(override))
	for name, on := range override {
		section := config.Section(strings.TrimSpace(name))
		if !section.IsKnown() {
			return nil, exceptions.New(exceptions.UnknownSection)
		}
		requested[section] = on
	}

	return defaults.Merge(requested), nil
}
