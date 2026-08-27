package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSectionSetMergeDoesNotMutateTheDefault(t *testing.T) {
	base := SectionSet{SectionExperience: true, SectionPatents: true}
	merged := base.Merge(SectionSet{SectionPatents: false})

	if merged.Enabled(SectionPatents) {
		t.Error("override did not turn patents off")
	}
	if !merged.Enabled(SectionExperience) {
		t.Error("merge dropped a section the base had enabled")
	}
	// The deployment default is shared across requests, so a per-request
	// override must never write through to it.
	if !base.Enabled(SectionPatents) {
		t.Error("Merge mutated the base set")
	}
}

func TestSectionSetAbsentMeansOff(t *testing.T) {
	set := SectionSet{SectionExperience: true}
	if set.Enabled(SectionPatents) {
		t.Error("a section that was never mentioned must be off")
	}
}

func TestAllEnabledCoversEveryKnownSection(t *testing.T) {
	set := AllEnabled()
	for _, section := range AllSections {
		if !set.Enabled(section) {
			t.Errorf("AllEnabled did not enable %q", section)
		}
	}
}

// A typo in config must fail loudly at startup, not silently return nothing.
func TestSectionSetRejectsUnknownNames(t *testing.T) {
	var set SectionSet
	err := yaml.Unmarshal([]byte("experiance: true\n"), &set)
	if err == nil {
		t.Fatal("unmarshal accepted an unknown section name, want an error")
	}
}

func TestSectionSetUnmarshalsKnownNames(t *testing.T) {
	var set SectionSet
	if err := yaml.Unmarshal([]byte("experience: true\npatents: false\n"), &set); err != nil {
		t.Fatalf("unmarshal returned an error: %v", err)
	}
	if !set.Enabled(SectionExperience) || set.Enabled(SectionPatents) {
		t.Errorf("parsed %+v, want experience on and patents off", set)
	}
}

func TestProfileLevelSectionsAreClassified(t *testing.T) {
	for _, section := range []Section{SectionAbout, SectionContactInfo, SectionOpenTo} {
		if !section.IsProfileLevel() {
			t.Errorf("%q should nest under profile", section)
		}
	}
	for _, section := range []Section{SectionExperience, SectionPatents, SectionSkills} {
		if section.IsProfileLevel() {
			t.Errorf("%q is a top-level array, not profile-level", section)
		}
	}
}

// AllSections is the canonical list every other part of the service reads; a
// duplicate or a stray entry there would corrupt the response contract.
func TestAllSectionsIsCleanAndComplete(t *testing.T) {
	seen := map[Section]bool{}
	for _, section := range AllSections {
		if seen[section] {
			t.Errorf("%q appears twice in AllSections", section)
		}
		seen[section] = true

		if !section.IsKnown() {
			t.Errorf("%q is in AllSections but IsKnown says otherwise", section)
		}
	}

	// The PRD defines 21 sections.
	if len(AllSections) != 21 {
		t.Errorf("AllSections has %d entries, want the 21 from the PRD", len(AllSections))
	}
}

// The config default must survive a real config file, unknown-name check included.
func TestSectionsLoadFromExampleConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@h:5432/d?sslmode=disable")
	t.Setenv("REDIS_HOST", "redis")

	cfg, err := Load("../../config/local.example.yml")
	if err != nil {
		t.Fatalf("the committed example config does not load: %v", err)
	}
	for _, section := range AllSections {
		if !cfg.Sections.Enabled(section) {
			t.Errorf("local.example.yml does not list %q — it must document every section", section)
		}
	}
}
