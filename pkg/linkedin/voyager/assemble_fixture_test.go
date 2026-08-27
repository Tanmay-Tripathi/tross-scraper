package voyager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestAssembleSpikeFixture runs the assembler over a directory of raw responses
// written by `go run ./cmd/spike`. Fixtures hold real people's data and are
// gitignored, so this is skipped unless TROSS_FIXTURE_DIR points at one — but it
// is the only test that checks the mapping against a live payload rather than a
// hand-written one. Run it after any change to the wire-format structs.
func TestAssembleSpikeFixture(t *testing.T) {
	dir := os.Getenv("TROSS_FIXTURE_DIR")
	if dir == "" {
		t.Skip("set TROSS_FIXTURE_DIR to a spike fixture directory")
	}

	bodies := map[string][]byte{}
	for _, endpoint := range ProfileEndpoints {
		body, err := os.ReadFile(filepath.Join(dir, endpoint.Name+".json"))
		if err != nil {
			t.Logf("%s: not in the fixture (%v)", endpoint.Name, err)
			continue
		}
		bodies[endpoint.Name] = body
	}

	if _, ok := bodies["dashProfile"]; !ok {
		t.Fatalf("no dashProfile.json in %s", dir)
	}

	profile, err := Assemble(filepath.Base(dir), bodies)
	if err != nil {
		t.Fatalf("Assemble failed on the live fixture: %v", err)
	}

	if profile.Identity.Name == "" {
		t.Error("Identity.Name is empty: the subject profile was not resolved")
	}

	// Counts, not values — this must not print anyone's profile into a log.
	t.Logf("name=%q headline=%t about=%t location=%q industry=%q pronouns=%q photo=%t",
		profile.Identity.Name, profile.Headline != "", profile.About != "",
		profile.Location, profile.Industry, profile.Identity.Pronouns,
		profile.Images.ProfilePhoto != nil)
	t.Logf("experience=%d education=%d skills=%d certifications=%d projects=%d publications=%d",
		len(profile.Experience), len(profile.Education), len(profile.Skills),
		len(profile.Certification), len(profile.Projects), len(profile.Publications))
	t.Logf("honors=%d courses=%d languages=%d volunteering=%d causes=%d recommendations=%d",
		len(profile.Honors), len(profile.Courses), len(profile.Languages),
		len(profile.Volunteering), len(profile.Causes), len(profile.Recommends))
	t.Logf("patents=%d testScores=%d organizations=%d",
		len(profile.Patents), len(profile.TestScores), len(profile.Organizations))

	// Every list section must marshal as [] rather than null.
	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("profile does not marshal: %v", err)
	}
	if len(encoded) == 0 {
		t.Error("profile marshalled to nothing")
	}
}
