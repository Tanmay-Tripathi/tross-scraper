package exceptions

import (
	"net/http"
	"testing"
)

func TestNewReturnsCataloguedError(t *testing.T) {
	appErr := New(InvalidParams)

	if appErr.ErrorCode != InvalidParams {
		t.Errorf("ErrorCode = %q, want %q", appErr.ErrorCode, InvalidParams)
	}
	if appErr.HttpCode != http.StatusBadRequest {
		t.Errorf("HttpCode = %d, want %d", appErr.HttpCode, http.StatusBadRequest)
	}
}

func TestNewFallsBackForUnknownCode(t *testing.T) {
	appErr := New(ErrorCode("NOT_A_REAL_CODE"))

	if appErr.ErrorCode != ErrorMessageNotFound {
		t.Errorf("ErrorCode = %q, want the %q fallback", appErr.ErrorCode, ErrorMessageNotFound)
	}
}

// New must hand out copies: a caller mutating one error must not corrupt the
// catalogue for every later caller.
func TestNewReturnsIndependentCopies(t *testing.T) {
	first := New(SomethingWentWrong)
	first.ErrorMessage = "mutated"

	if second := New(SomethingWentWrong); second.ErrorMessage == "mutated" {
		t.Error("New returned a shared pointer into the catalogue")
	}
}

// Feature files register their codes from init(), so codes declared outside
// this file must still resolve.
func TestFeatureCodesAreRegistered(t *testing.T) {
	appErr := New(HealthDependencyDown)

	if appErr.ErrorCode != HealthDependencyDown {
		t.Errorf("ErrorCode = %q, want %q", appErr.ErrorCode, HealthDependencyDown)
	}
	if appErr.HttpCode != http.StatusServiceUnavailable {
		t.Errorf("HttpCode = %d, want %d", appErr.HttpCode, http.StatusServiceUnavailable)
	}
}

func TestWrapLogsAndReturnsCataloguedError(t *testing.T) {
	var logged string
	logf := func(format string, args ...any) { logged = format }

	appErr := Wrap(logf, SomethingWentWrong, "fetch profile %s", "abc")

	if logged == "" {
		t.Error("Wrap did not call the log function")
	}
	if appErr.ErrorCode != SomethingWentWrong {
		t.Errorf("ErrorCode = %q, want %q", appErr.ErrorCode, SomethingWentWrong)
	}
	if appErr.ErrorMessage == "fetch profile %s" {
		t.Error("the internal detail leaked into the client-facing message")
	}
}
