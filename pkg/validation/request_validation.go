// Package validation holds one validator per request type. They normalise input,
// return an *exceptions.ApplicationError, and never touch Gin or HTTP codes.
package validation

import (
	"strings"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
)

// RequireNonEmpty returns the trimmed value, or InvalidParams when blank.
func RequireNonEmpty(value string) (string, *exceptions.ApplicationError) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", exceptions.New(exceptions.InvalidParams)
	}
	return trimmed, nil
}
