package exceptions

import "net/http"

// Profile feature error codes.
const (
	// InvalidProfileURL means the input was not a LinkedIn member profile URL.
	InvalidProfileURL ErrorCode = "PRF01"
	// ProfileNotFound means LinkedIn has no such profile, or our account cannot see it.
	ProfileNotFound ErrorCode = "PRF02"
	// UpstreamShapeChanged means LinkedIn answered in a shape we cannot parse.
	UpstreamShapeChanged ErrorCode = "PRF03"
	// SessionExpired means our LinkedIn cookies are no longer valid. This is the
	// one failure that needs a human, so it is deliberately distinct from a 500.
	SessionExpired ErrorCode = "PRF04"
	// UpstreamRateLimited means LinkedIn is throttling us.
	UpstreamRateLimited ErrorCode = "PRF05"
	// UnknownSection means the request asked to toggle a section we do not support.
	UnknownSection ErrorCode = "PRF06"
	// ScrapeBudgetExhausted means our own daily cap is spent, not LinkedIn's.
	ScrapeBudgetExhausted ErrorCode = "PRF07"
)

func init() {
	register(map[ErrorCode]ApplicationError{
		InvalidProfileURL: {
			ErrorCode:    InvalidProfileURL,
			ErrorMessage: "not a valid LinkedIn member profile URL",
			HttpCode:     http.StatusBadRequest,
		},
		ProfileNotFound: {
			ErrorCode:    ProfileNotFound,
			ErrorMessage: "profile not found, private, or not visible to this account",
			HttpCode:     http.StatusNotFound,
		},
		UpstreamShapeChanged: {
			ErrorCode:    UpstreamShapeChanged,
			ErrorMessage: "linkedin returned an unexpected response shape",
			HttpCode:     http.StatusBadGateway,
		},
		SessionExpired: {
			ErrorCode:    SessionExpired,
			ErrorMessage: "the linkedin session has expired and needs new cookies",
			HttpCode:     http.StatusServiceUnavailable,
		},
		UpstreamRateLimited: {
			ErrorCode:    UpstreamRateLimited,
			ErrorMessage: "linkedin is rate limiting this account, retry later",
			HttpCode:     http.StatusTooManyRequests,
		},
		UnknownSection: {
			ErrorCode:    UnknownSection,
			ErrorMessage: "request referenced an unsupported profile section",
			HttpCode:     http.StatusBadRequest,
		},
		ScrapeBudgetExhausted: {
			ErrorCode:    ScrapeBudgetExhausted,
			ErrorMessage: "the daily scrape budget is exhausted, retry tomorrow",
			HttpCode:     http.StatusTooManyRequests,
		},
	})
}
