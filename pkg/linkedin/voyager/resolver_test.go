package voyager

import "testing"

// A miniature payload in the exact shape Voyager returns: "data" refers to
// objects by entityUrn, and the objects themselves sit flat in "included".
const samplePayload = `{
  "data": {
    "$type": "com.linkedin.voyager.identity.profile.ProfileView",
    "profile": "urn:li:fs_profile:ACoAAA",
    "positionView": {
      "elements": ["urn:li:fs_position:1", "urn:li:fs_position:2"]
    }
  },
  "included": [
    {
      "$type": "com.linkedin.voyager.identity.shared.MiniProfile",
      "entityUrn": "urn:li:fs_profile:ACoAAA",
      "firstName": "Priya",
      "lastName": "Nair"
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Position",
      "entityUrn": "urn:li:fs_position:1",
      "companyName": "Razorstack",
      "title": "Senior Backend Engineer"
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Position",
      "entityUrn": "urn:li:fs_position:2",
      "companyName": "Finlytics",
      "title": "Backend Engineer"
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Education",
      "entityUrn": "urn:li:fs_education:9",
      "schoolName": "BITS Pilani"
    },
    { "no_entity_urn": true },
    { "$type": "com.linkedin.voyager.identity.profile.Skill", "entityUrn": "urn:li:fs_skill:1", "name": "Go" }
  ]
}`

type position struct {
	CompanyName string `json:"companyName"`
	Title       string `json:"title"`
}

type miniProfile struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func TestNewGraphIndexesIncluded(t *testing.T) {
	graph, err := NewGraph([]byte(samplePayload))
	if err != nil {
		t.Fatalf("NewGraph returned an error: %v", err)
	}

	// The entity with no entityUrn is skipped, leaving five.
	if graph.Size() != 5 {
		t.Errorf("Size() = %d, want 5 (the entry without an entityUrn must be skipped)", graph.Size())
	}
}

func TestResolveFollowsAUrn(t *testing.T) {
	graph, _ := NewGraph([]byte(samplePayload))

	var profile miniProfile
	if !graph.Resolve("urn:li:fs_profile:ACoAAA", &profile) {
		t.Fatal("Resolve could not find the profile urn")
	}
	if profile.FirstName != "Priya" || profile.LastName != "Nair" {
		t.Errorf("resolved %+v, want Priya Nair", profile)
	}
}

func TestResolveMissingUrn(t *testing.T) {
	graph, _ := NewGraph([]byte(samplePayload))

	var profile miniProfile
	if graph.Resolve("urn:li:fs_profile:DOES_NOT_EXIST", &profile) {
		t.Error("Resolve reported success for a urn that is not in the payload")
	}
}

// This is the real access pattern: data gives a list of urns, we resolve them.
func TestResolveAllPreservesOrderAndSkipsMissing(t *testing.T) {
	graph, _ := NewGraph([]byte(samplePayload))

	urns := []string{"urn:li:fs_position:1", "urn:li:fs_position:missing", "urn:li:fs_position:2"}
	positions := ResolveAll[position](graph, urns)

	if len(positions) != 2 {
		t.Fatalf("got %d positions, want 2 (the missing urn should be skipped, not fatal)", len(positions))
	}
	if positions[0].CompanyName != "Razorstack" || positions[1].CompanyName != "Finlytics" {
		t.Errorf("order not preserved: %+v", positions)
	}
}

// Matching on the type suffix survives LinkedIn renaming its namespace, which it
// does more often than it renames the leaf type.
func TestByTypeMatchesOnSuffix(t *testing.T) {
	graph, _ := NewGraph([]byte(samplePayload))

	positions := ByType[position](graph, ".Position")
	if len(positions) != 2 {
		t.Fatalf("ByType found %d positions, want 2", len(positions))
	}
	if positions[0].Title != "Senior Backend Engineer" {
		t.Errorf("first position = %q, want the one LinkedIn listed first", positions[0].Title)
	}

	if got := ByType[position](graph, ".NotARealType"); len(got) != 0 {
		t.Errorf("ByType on an unknown suffix returned %d items, want 0", len(got))
	}
}

func TestTypesCountsEntities(t *testing.T) {
	graph, _ := NewGraph([]byte(samplePayload))
	types := graph.Types()

	if types["com.linkedin.voyager.identity.profile.Position"] != 2 {
		t.Errorf("Position count = %d, want 2", types["com.linkedin.voyager.identity.profile.Position"])
	}
	if types["com.linkedin.voyager.identity.profile.Education"] != 1 {
		t.Errorf("Education count = %d, want 1", types["com.linkedin.voyager.identity.profile.Education"])
	}
}

func TestNewGraphRejectsNonJSON(t *testing.T) {
	if _, err := NewGraph([]byte("<html>rate limited</html>")); err == nil {
		t.Fatal("NewGraph accepted an HTML body, want an error")
	}
}

// An empty or unauthenticated answer must not panic; it is simply an empty graph.
func TestNewGraphHandlesEmptyPayload(t *testing.T) {
	graph, err := NewGraph([]byte(`{"data":{},"included":[]}`))
	if err != nil {
		t.Fatalf("NewGraph returned an error: %v", err)
	}
	if graph.Size() != 0 {
		t.Errorf("Size() = %d, want 0", graph.Size())
	}
}
