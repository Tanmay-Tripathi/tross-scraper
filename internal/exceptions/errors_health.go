package exceptions

import "net/http"

// Health feature error codes.
const (
	HealthDependencyDown ErrorCode = "HLT01"
)

func init() {
	register(map[ErrorCode]ApplicationError{
		HealthDependencyDown: {
			ErrorCode:    HealthDependencyDown,
			ErrorMessage: "one or more downstream dependencies are unavailable",
			HttpCode:     http.StatusServiceUnavailable,
		},
	})
}
