package voyager

import (
	"context"
	"fmt"
	"net/url"
)

// dashProfileDecoration asks for the profile with every section collection
// attached. Without it the response is the bare member record and every section
// pointer is absent.
const dashProfileDecoration = "com.linkedin.voyager.dash.deco.identity.profile.FullProfileWithEntities-100"

// Endpoint is one Voyager call we make when assembling a profile.
type Endpoint struct {
	// Name identifies the call in logs, fixtures and the spike report.
	Name string
	// Path builds the URL path for a public identifier.
	Path func(publicID string) string
	// Query builds the query string, if any.
	Query func(publicID string) map[string]string
	// Essential marks the call the profile cannot be assembled without.
	Essential bool
	// Sections notes which response sections this call feeds.
	Sections []string
}

// URL renders the full request URL for a public identifier.
func (e Endpoint) URL(publicID string) string {
	full := BaseURL + e.Path(publicID)
	if e.Query == nil {
		return full
	}
	values := url.Values{}
	for key, value := range e.Query(publicID) {
		values.Set(key, value)
	}
	if encoded := values.Encode(); encoded != "" {
		full += "?" + encoded
	}
	return full
}

// ProfileEndpoints are the calls made for one profile.
//
// LinkedIn has retired its legacy identity API route by route: profileView,
// profiles/{id}, profileContactInfo and profiles/{id}/skills all answer 410 Gone,
// while recommendations under the same prefix still answers 200. What replaced
// them is the dash profile below, which carries every section in one response.
var ProfileEndpoints = []Endpoint{
	{
		Name: "dashProfile",
		Path: func(string) string { return "/identity/dash/profiles" },
		Query: func(id string) map[string]string {
			return map[string]string{
				"q":              "memberIdentity",
				"memberIdentity": id,
				"decorationId":   dashProfileDecoration,
			}
		},
		Essential: true,
		Sections: []string{
			"identity", "headline", "location", "industry", "about", "images",
			"experience", "education", "skills", "licensesAndCertifications",
			"projects", "courses", "publications", "patents", "honorsAndAwards",
			"testScores", "languages", "organizations", "volunteerExperience",
			"causes",
		},
	},
	{
		Name:     "recommendations",
		Path:     func(id string) string { return "/identity/profiles/" + url.PathEscape(id) + "/recommendations" },
		Query:    func(string) map[string]string { return map[string]string{"q": "received", "count": "50"} },
		Sections: []string{"recommendations"},
	},
}

// EssentialEndpointName is the endpoint a profile cannot be assembled without.
// Callers key off this rather than a literal, so swapping the essential endpoint
// is a one-line change here instead of a silent mismatch somewhere else.
func EssentialEndpointName() string {
	for _, endpoint := range ProfileEndpoints {
		if endpoint.Essential {
			return endpoint.Name
		}
	}
	return ""
}

// Fetch calls one endpoint for a public identifier.
func (c *Client) Fetch(ctx context.Context, endpoint Endpoint, publicID string) ([]byte, error) {
	var query map[string]string
	if endpoint.Query != nil {
		query = endpoint.Query(publicID)
	}
	return c.Get(ctx, endpoint.Path(publicID), query)
}

// FetchAll calls every endpoint and returns the raw bodies keyed by name, plus
// failures. A non-essential failure is recorded and the run continues.
func (c *Client) FetchAll(ctx context.Context, publicID string) (map[string][]byte, map[string]error) {
	bodies := make(map[string][]byte, len(ProfileEndpoints))
	failures := make(map[string]error)

	for _, endpoint := range ProfileEndpoints {
		body, err := c.Fetch(ctx, endpoint, publicID)
		if err != nil {
			failures[endpoint.Name] = err
			if endpoint.Essential {
				return bodies, failures
			}
			c.logger.Warnf("voyager: non-essential endpoint %q failed for %q: %v", endpoint.Name, publicID, err)
			continue
		}

		bodies[endpoint.Name] = body
	}

	if len(bodies) == 0 && len(failures) == 0 {
		failures["_"] = fmt.Errorf("no endpoints configured")
	}

	return bodies, failures
}
