// Package requests holds the inbound DTOs; validation lives in pkg/validation.
package requests

// RequestScrapeProfile is the body of POST /v1/profile.
type RequestScrapeProfile struct {
	// ProfileURL is any reasonable LinkedIn member profile URL.
	ProfileURL string `json:"profileUrl" binding:"required"`
	// Sections overrides the deployment default one section at a time. Absent
	// means "leave as configured"; false means "switch off".
	Sections map[string]bool `json:"sections"`
	// Refresh forces a live scrape. It still counts against the daily budget.
	Refresh bool `json:"refresh"`
}
