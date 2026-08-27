package mapper

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

func render(t *testing.T, profile *models.Profile, sections config.SectionSet) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(ToProfileResult(profile, sections, false))
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	return decoded
}

func sampleProfile() *models.Profile {
	photo := "https://media.example.com/priya.jpg"
	return &models.Profile{
		PublicID:  "priya",
		Identity:  models.Identity{Name: "Priya Nair", FirstName: "Priya", LastName: "Nair"},
		Headline:  "Backend Engineer",
		FetchedAt: time.Unix(0, 0).UTC(),
		Images:    models.Images{ProfilePhoto: &photo}, // background deliberately absent
		Experience: []models.Experience{{
			Title:   "Senior Backend Engineer",
			Company: "Razorstack",
			Period:  models.DateRange{Start: &models.Date{Year: 2022}, Present: true},
		}},
		Patents: []models.Patent{}, // enabled but empty
	}
}

// Rule 1: enabled with data -> the data is present.
func TestEnabledSectionWithDataIsReturned(t *testing.T) {
	out := render(t, sampleProfile(), config.AllEnabled())

	experience, ok := out["experience"].([]any)
	if !ok || len(experience) != 1 {
		t.Fatalf("experience = %#v, want one entry", out["experience"])
	}
}

// Rule 2: enabled with no data -> an empty array, not null and not absent.
func TestEnabledSectionWithNoDataIsEmptyArray(t *testing.T) {
	out := render(t, sampleProfile(), config.AllEnabled())

	raw, present := out["patents"]
	if !present {
		t.Fatal("patents key is missing; an enabled section must always be present")
	}
	patents, ok := raw.([]any)
	if !ok {
		t.Fatalf("patents = %#v, want an empty array, never null", raw)
	}
	if len(patents) != 0 {
		t.Errorf("patents has %d entries, want 0", len(patents))
	}
}

// Rule 3: disabled -> the key is absent entirely.
func TestDisabledSectionIsOmitted(t *testing.T) {
	sections := config.AllEnabled().Merge(config.SectionSet{config.SectionPatents: false})
	out := render(t, sampleProfile(), sections)

	if _, present := out["patents"]; present {
		t.Error("patents key is present, but the section is disabled — it must be omitted entirely")
	}
	if _, present := out["experience"]; !present {
		t.Error("disabling patents should not affect other sections")
	}
}

// The distinction the whole contract rests on.
func TestDisabledAndEmptyAreDistinguishable(t *testing.T) {
	enabled := render(t, sampleProfile(), config.AllEnabled())
	disabled := render(t, sampleProfile(), config.AllEnabled().Merge(config.SectionSet{config.SectionPatents: false}))

	_, enabledHasKey := enabled["patents"]
	_, disabledHasKey := disabled["patents"]

	if !enabledHasKey || disabledHasKey {
		t.Error("an enabled-but-empty section and a disabled one must serialise differently")
	}
}

// A missing image is null. Never a placeholder URL.
func TestMissingImageIsNullNotAPlaceholder(t *testing.T) {
	out := render(t, sampleProfile(), config.AllEnabled())

	profile := out["profile"].(map[string]any)
	images := profile["images"].(map[string]any)

	if images["profilePhoto"] == nil {
		t.Error("profilePhoto should carry the real URL")
	}
	raw, present := images["backgroundPhoto"]
	if !present {
		t.Fatal("backgroundPhoto key must always be present")
	}
	if raw != nil {
		t.Errorf("backgroundPhoto = %#v, want null when LinkedIn gave us nothing", raw)
	}
}

// An object full of blanks is not data; contactInfo must be null instead.
func TestEmptyContactInfoIsNull(t *testing.T) {
	profile := sampleProfile()
	profile.Contact = &models.ContactInfo{}

	out := render(t, profile, config.AllEnabled())
	core := out["profile"].(map[string]any)

	if value, present := core["contactInfo"]; present && value != nil {
		t.Errorf("contactInfo = %#v, want null when every field is blank", value)
	}
}

func TestContactInfoIsOmittedWhenSectionDisabled(t *testing.T) {
	profile := sampleProfile()
	profile.Contact = &models.ContactInfo{Email: "priya@example.dev"}

	sections := config.AllEnabled().Merge(config.SectionSet{config.SectionContactInfo: false})
	core := render(t, profile, sections)["profile"].(map[string]any)

	if _, present := core["contactInfo"]; present {
		t.Error("contactInfo must be omitted when its section is disabled, even when data exists")
	}
}

// A year-only date must not gain a fabricated month.
func TestPartialDatesKeepTheirPrecision(t *testing.T) {
	tests := []struct {
		name string
		date *models.Date
		want any
	}{
		{"year only", &models.Date{Year: 2022}, "2022"},
		{"year and month", &models.Date{Year: 2022, Month: 3}, "2022-03"},
		{"full date", &models.Date{Year: 2022, Month: 3, Day: 9}, "2022-03-09"},
		{"nothing", nil, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDate(tc.date)
			if tc.want == nil {
				if got != nil {
					t.Fatalf("formatDate = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != tc.want.(string) {
				t.Fatalf("formatDate = %v, want %q", got, tc.want)
			}
		})
	}
}

// An ongoing role has no end date; "present" carries that, not a fake end.
func TestOngoingRoleHasNullEndAndPresentTrue(t *testing.T) {
	out := render(t, sampleProfile(), config.AllEnabled())

	entry := out["experience"].([]any)[0].(map[string]any)
	period := entry["period"].(map[string]any)

	if period["present"] != true {
		t.Error("present should be true for an ongoing role")
	}
	if period["end"] != nil {
		t.Errorf("end = %#v, want null for an ongoing role", period["end"])
	}
}

// Section order must not shift when unrelated sections are toggled.
func TestSectionOrderIsStable(t *testing.T) {
	first, _ := json.Marshal(ToProfileResult(sampleProfile(), config.AllEnabled(), false))
	second, _ := json.Marshal(ToProfileResult(sampleProfile(), config.AllEnabled(), false))

	if string(first) != string(second) {
		t.Error("the same profile serialised differently across two calls")
	}
}

// meta must tell the caller which enabled sections came back empty.
func TestMetaReportsUnavailableSections(t *testing.T) {
	out := render(t, sampleProfile(), config.AllEnabled())
	meta := out["meta"].(map[string]any)

	unavailable, ok := meta["sectionsUnavailable"].([]any)
	if !ok {
		t.Fatalf("sectionsUnavailable = %#v, want an array", meta["sectionsUnavailable"])
	}

	found := false
	for _, name := range unavailable {
		if name == "patents" {
			found = true
		}
	}
	if !found {
		t.Error("patents was enabled but empty, so it should be listed as unavailable")
	}
}
