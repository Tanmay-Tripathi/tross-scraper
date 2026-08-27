// Package validation holds one validator per incoming request type. Validators
// normalise input (trim, lowercase, parse identifiers) and return a
// *exceptions.ApplicationError describing the first problem they find.
//
// Validators never touch Gin types or HTTP status codes — controllers call them
// and hand the returned error to the response helpers.
package validation

import (
	"strconv"
	"strings"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/exceptions"
)

// TrimmedString normalises free-text input by trimming surrounding whitespace.
func TrimmedString(value string) string {
	return strings.TrimSpace(value)
}

// NormalizedEnum trims and lowercases a value that is compared against a fixed
// set of allowed strings.
func NormalizedEnum(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// ParseUint64ID converts a string identifier into its numeric form. fieldName
// appears only in logs; the caller receives a catalogued InvalidParams error.
func ParseUint64ID(value string) (uint64, *exceptions.ApplicationError) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, exceptions.New(exceptions.InvalidParams)
	}

	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || parsed == 0 {
		return 0, exceptions.New(exceptions.InvalidParams)
	}

	return parsed, nil
}

// RequireNonEmpty reports an InvalidParams error when value is blank after
// trimming, and returns the trimmed value otherwise.
func RequireNonEmpty(value string) (string, *exceptions.ApplicationError) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", exceptions.New(exceptions.InvalidParams)
	}
	return trimmed, nil
}
