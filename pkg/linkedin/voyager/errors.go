package voyager

import (
	"fmt"
	"net/http"
)

// StatusError is a non-2xx answer from Voyager, with enough context to map it
// onto a service error code.
type StatusError struct {
	StatusCode int
	Path       string
	Body       string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("voyager %s returned %d: %s", e.Path, e.StatusCode, e.Body)
}

// SessionExpired reports whether LinkedIn rejected our credentials. 403 counts: it
// covers a failed CSRF check, and "refresh the cookies" beats a silent failure.
// So does any 3xx: Voyager answers a signed-in caller directly and redirects
// everyone else at the login page, so a redirect is a rejection wearing a hat.
func (e *StatusError) SessionExpired() bool {
	if e.StatusCode >= 300 && e.StatusCode < 400 {
		return true
	}
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// NotFound reports whether the profile does not exist or is invisible to us.
func (e *StatusError) NotFound() bool {
	return e.StatusCode == http.StatusNotFound || e.StatusCode == http.StatusGone
}

// RateLimited reports throttling; 999 is LinkedIn's own non-standard code for it.
func (e *StatusError) RateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode == 999
}
