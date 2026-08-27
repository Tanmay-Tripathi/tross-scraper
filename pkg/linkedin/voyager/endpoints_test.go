package voyager

import (
	"net/http"
	"testing"
)

// Callers key the essential body off this name. If it ever returns "" or names an
// endpoint that is not in the list, every fetch fails with an empty cause — which
// is exactly what happened when the essential endpoint moved from profileView to
// dashProfile and a literal was left behind.
func TestEssentialEndpointNameMatchesTheList(t *testing.T) {
	name := EssentialEndpointName()
	if name == "" {
		t.Fatal("no endpoint is marked Essential; every fetch would fail with no cause")
	}

	essentials := 0
	found := false
	for _, endpoint := range ProfileEndpoints {
		if endpoint.Essential {
			essentials++
		}
		if endpoint.Name == name {
			found = true
		}
	}

	if !found {
		t.Errorf("EssentialEndpointName() = %q, which is not in ProfileEndpoints", name)
	}
	if essentials != 1 {
		t.Errorf("%d endpoints marked Essential, want exactly 1", essentials)
	}
}

// A redirect is Voyager's way of saying "log in", so it must surface as an expired
// session rather than an unexplained transport failure.
func TestStatusErrorTreatsRedirectsAsSessionExpiry(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{
		{http.StatusFound, true},
		{http.StatusMovedPermanently, true},
		{http.StatusTemporaryRedirect, true},
		{http.StatusUnauthorized, true},
		{http.StatusForbidden, true},
		{http.StatusNotFound, false},
		{http.StatusGone, false},
		{http.StatusOK, false},
	}

	for _, tc := range cases {
		err := &StatusError{StatusCode: tc.status}
		if got := err.SessionExpired(); got != tc.want {
			t.Errorf("status %d: SessionExpired() = %t, want %t", tc.status, got, tc.want)
		}
	}
}

// 410 is how LinkedIn retires a route, and it must not read as a missing profile
// forever — but while a retired route is still listed, NotFound is the honest map.
func TestStatusErrorClassifies(t *testing.T) {
	if !(&StatusError{StatusCode: http.StatusGone}).NotFound() {
		t.Error("410 should classify as not found")
	}
	if !(&StatusError{StatusCode: 999}).RateLimited() {
		t.Error("999 is LinkedIn's own throttling code")
	}
}
