// Package exceptions defines ApplicationError, the single error type that
// crosses layer boundaries in this service. Validators, services and
// repositories return *ApplicationError; controllers hand it to
// utils.SendApiResponseV2, which turns it into the HTTP error envelope.
package exceptions

import (
	"fmt"
	"net/http"
)

// ErrorCode is a short, stable identifier for a failure mode. Clients switch
// on it, so codes must never be renamed once released.
type ErrorCode string

// SuccessCode is the code returned on the success envelope.
type SuccessCode string

const (
	// ApiSuccessCode is the code every successful response carries.
	ApiSuccessCode SuccessCode = "00000"
	// ApiSuccessMessage is the message every successful response carries.
	ApiSuccessMessage = "success"
)

// ApplicationError carries a client-facing failure across layers.
type ApplicationError struct {
	ErrorCode    ErrorCode `json:"code"`
	ErrorMessage string    `json:"message"`
	HttpCode     int       `json:"-"`
}

func (e *ApplicationError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.ErrorCode) + ": " + e.ErrorMessage
}

// Generic error codes shared by every feature. Feature-specific codes live in
// their own errors_<feature>.go file in this package.
const (
	ErrorMessageNotFound ErrorCode = "EMNF01"
	SomethingWentWrong   ErrorCode = "SWW01"
	InvalidParams        ErrorCode = "INV01"
	InvalidRequest       ErrorCode = "IR01"
	EmptyBody            ErrorCode = "EB01"
	Unauthorized         ErrorCode = "UA01"
	TooManyRequests      ErrorCode = "TMR01"
	ServiceUnavailable   ErrorCode = "SU01"
)

// errorCatalogue maps every known code to its user-facing message and status.
// Feature files register their codes into it from an init function.
var errorCatalogue = map[ErrorCode]ApplicationError{
	ErrorMessageNotFound: {
		ErrorCode:    ErrorMessageNotFound,
		ErrorMessage: "message not defined for error code",
		HttpCode:     http.StatusBadRequest,
	},
	SomethingWentWrong: {
		ErrorCode:    SomethingWentWrong,
		ErrorMessage: "something went wrong",
		HttpCode:     http.StatusInternalServerError,
	},
	InvalidParams: {
		ErrorCode:    InvalidParams,
		ErrorMessage: "invalid params",
		HttpCode:     http.StatusBadRequest,
	},
	InvalidRequest: {
		ErrorCode:    InvalidRequest,
		ErrorMessage: "invalid request",
		HttpCode:     http.StatusBadRequest,
	},
	EmptyBody: {
		ErrorCode:    EmptyBody,
		ErrorMessage: "request body is empty",
		HttpCode:     http.StatusBadRequest,
	},
	Unauthorized: {
		ErrorCode:    Unauthorized,
		ErrorMessage: "unauthorized",
		HttpCode:     http.StatusUnauthorized,
	},
	TooManyRequests: {
		ErrorCode:    TooManyRequests,
		ErrorMessage: "too many requests, please retry later",
		HttpCode:     http.StatusTooManyRequests,
	},
	ServiceUnavailable: {
		ErrorCode:    ServiceUnavailable,
		ErrorMessage: "service is temporarily unavailable",
		HttpCode:     http.StatusServiceUnavailable,
	},
}

// register adds a feature's codes to the catalogue. It panics on a duplicate
// code so collisions surface at startup rather than in production responses.
func register(errors map[ErrorCode]ApplicationError) {
	for code, appErr := range errors {
		if _, exists := errorCatalogue[code]; exists {
			panic(fmt.Sprintf("exceptions: duplicate error code %q", code))
		}
		errorCatalogue[code] = appErr
	}
}

// New returns the catalogued error for code. An unknown code falls back to
// ErrorMessageNotFound so a response is always well-formed.
func New(code ErrorCode) *ApplicationError {
	appErr, ok := errorCatalogue[code]
	if !ok {
		fallback := errorCatalogue[ErrorMessageNotFound]
		return &fallback
	}
	return &appErr
}

// Wrap returns the catalogued error for code after logging the internal detail
// through logf. Use it at the layer that detects the failure so the operator
// sees the cause while the client only sees the catalogued message.
//
//	return nil, exceptions.Wrap(logger.Errorf, exceptions.SomethingWentWrong,
//	    "fetch profile %s", profileID, err)
func Wrap(logf func(string, ...any), code ErrorCode, format string, args ...any) *ApplicationError {
	if logf != nil {
		logf("["+string(code)+"] "+format, args...)
	}
	return New(code)
}
