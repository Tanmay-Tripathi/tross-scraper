package validation

import "testing"

func TestParseProfileURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"canonical", "https://www.linkedin.com/in/priya-nair-eng/", "priya-nair-eng"},
		{"no trailing slash", "https://www.linkedin.com/in/priya-nair-eng", "priya-nair-eng"},
		{"no www", "https://linkedin.com/in/priya", "priya"},
		{"no scheme", "linkedin.com/in/priya", "priya"},
		{"www no scheme", "www.linkedin.com/in/priya", "priya"},
		{"http", "http://www.linkedin.com/in/priya/", "priya"},
		{"country subdomain", "https://in.linkedin.com/in/priya", "priya"},
		{"tracking query", "https://www.linkedin.com/in/priya?trk=people-search", "priya"},
		{"fragment", "https://www.linkedin.com/in/priya#experience", "priya"},
		{"details subpage", "https://www.linkedin.com/in/priya/details/experience/", "priya"},
		{"recent-activity subpage", "https://www.linkedin.com/in/priya/recent-activity/all/", "priya"},
		{"locale prefix", "https://www.linkedin.com/en-us/in/priya", "priya"},
		{"short locale prefix", "https://www.linkedin.com/de/in/priya", "priya"},
		{"legacy pub format", "https://www.linkedin.com/pub/priya/1/2/3", "priya"},
		{"percent encoded", "https://www.linkedin.com/in/jos%C3%A9-garcia", "josé-garcia"},
		{"surrounding whitespace", "  https://www.linkedin.com/in/priya/  ", "priya"},
		{"uppercase host", "https://WWW.LINKEDIN.COM/in/priya", "priya"},
		{"numeric slug", "https://www.linkedin.com/in/priya-nair-123456789", "priya-nair-123456789"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, appErr := ParseProfileURL(tc.input)
			if appErr != nil {
				t.Fatalf("ParseProfileURL(%q) returned %v, want %q", tc.input, appErr, tc.want)
			}
			if got != tc.want {
				t.Errorf("ParseProfileURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseProfileURLRejects(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"not a url", "hello world"},
		{"wrong site", "https://example.com/in/priya"},
		// The two below are the ones a naive strings.Contains check would let through.
		{"lookalike host", "https://notlinkedin.com/in/priya"},
		{"host prefix attack", "https://linkedin.com.evil.net/in/priya"},
		{"company page", "https://www.linkedin.com/company/acme/"},
		{"school page", "https://www.linkedin.com/school/bits-pilani/"},
		{"job posting", "https://www.linkedin.com/jobs/view/123456"},
		{"a post", "https://www.linkedin.com/posts/priya_activity-123"},
		{"feed", "https://www.linkedin.com/feed/"},
		{"bare domain", "https://www.linkedin.com/"},
		{"in with no slug", "https://www.linkedin.com/in/"},
		{"slug too long", "https://www.linkedin.com/in/" + longSlug(200)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, appErr := ParseProfileURL(tc.input)
			if appErr == nil {
				t.Fatalf("ParseProfileURL(%q) = %q, want an error", tc.input, got)
			}
		})
	}
}

// A company or job URL is valid LinkedIn but not a person; we distinguish it so
// the caller can say why, not just that it failed.
func TestIsNonProfileLinkedInURL(t *testing.T) {
	nonProfiles := []string{
		"https://www.linkedin.com/company/acme/",
		"https://www.linkedin.com/jobs/view/123",
		"https://www.linkedin.com/school/bits-pilani",
		"linkedin.com/feed/",
	}
	for _, input := range nonProfiles {
		if !IsNonProfileLinkedInURL(input) {
			t.Errorf("IsNonProfileLinkedInURL(%q) = false, want true", input)
		}
	}

	others := []string{
		"https://www.linkedin.com/in/priya",
		"https://example.com/company/acme",
		"not a url at all",
	}
	for _, input := range others {
		if IsNonProfileLinkedInURL(input) {
			t.Errorf("IsNonProfileLinkedInURL(%q) = true, want false", input)
		}
	}
}

func longSlug(n int) string {
	s := make([]byte, n)
	for i := range s {
		s[i] = 'a'
	}
	return string(s)
}
