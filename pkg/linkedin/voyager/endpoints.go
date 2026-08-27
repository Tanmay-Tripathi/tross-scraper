package voyager

import (
	"context"
	"fmt"
	"net/url"
)

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

// ProfileEndpoints are the calls made for one profile. profileView is the
// workhorse — what the page itself requests — and the rest fill its gaps.
var ProfileEndpoints = []Endpoint{
	{
		Name:      "profileView",
		Path:      func(id string) string { return "/identity/profiles/" + url.PathEscape(id) + "/profileView" },
		Essential: true,
		Sections: []string{
			"identity", "headline", "location", "industry", "about", "images",
			"experience", "education", "skills", "licensesAndCertifications",
			"projects", "courses", "publications", "patents", "honorsAndAwards",
			"testScores", "languages", "organizations", "volunteerExperience",
		},
	},
	{
		Name:     "profile",
		Path:     func(id string) string { return "/identity/profiles/" + url.PathEscape(id) },
		Sections: []string{"identity", "headline", "location", "industry", "images"},
	},
	{
		Name:     "contactInfo",
		Path:     func(id string) string { return "/identity/profiles/" + url.PathEscape(id) + "/profileContactInfo" },
		Sections: []string{"contactInfo"},
	},
	{
		Name: "skills",
		Path: func(id string) string { return "/identity/profiles/" + url.PathEscape(id) + "/skills" },
		Query: func(string) map[string]string {
			return map[string]string{"count": "100", "start": "0"}
		},
		Sections: []string{"skills"},
	},
	{
		Name:     "recommendations",
		Path:     func(id string) string { return "/identity/profiles/" + url.PathEscape(id) + "/recommendations" },
		Query:    func(string) map[string]string { return map[string]string{"q": "received", "count": "50"} },
		Sections: []string{"recommendations"},
	},
	{
		Name: "dashProfile",
		Path: func(string) string { return "/identity/dash/profiles" },
		Query: func(id string) map[string]string {
			return map[string]string{
				"q":              "memberIdentity",
				"memberIdentity": id,
				"decorationId":   "com.linkedin.voyager.dash.deco.identity.profile.FullProfileWithEntities-100",
			}
		},
		Sections: []string{"openTo", "featured", "services", "careerBreaks", "causes"},
	},
}

// FetchAll calls every endpoint and returns the raw bodies keyed by name, plus
// failures. A non-essential failure is recorded and the run continues.
func (c *Client) FetchAll(ctx context.Context, publicID string) (map[string][]byte, map[string]error) {
	bodies := make(map[string][]byte, len(ProfileEndpoints))
	failures := make(map[string]error)

	for _, endpoint := range ProfileEndpoints {
		var query map[string]string
		if endpoint.Query != nil {
			query = endpoint.Query(publicID)
		}

		body, err := c.Get(ctx, endpoint.Path(publicID), query)
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
