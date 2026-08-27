package voyager

import (
	"testing"
)

// A realistic profileView payload: data holds pointers, included the objects.
const profileViewPayload = `{
  "data": {
    "*profile": "urn:li:fs_profile:ACoAAA",
    "*positionView": { "*elements": ["urn:li:fs_position:1", "urn:li:fs_position:2"] },
    "*educationView": { "*elements": ["urn:li:fs_education:1"] },
    "*skillView": { "*elements": ["urn:li:fs_skill:1"] },
    "*patentView": { "*elements": [] }
  },
  "included": [
    {
      "$type": "com.linkedin.voyager.identity.profile.Profile",
      "entityUrn": "urn:li:fs_profile:ACoAAA",
      "firstName": "Priya",
      "lastName": "Nair",
      "headline": "Senior Backend Engineer",
      "summary": "Backend engineer focused on payment rails.",
      "industryName": "Financial Services",
      "geoLocationName": "Bengaluru, Karnataka, India",
      "publicIdentifier": "priya-nair-eng",
      "profilePicture": {
        "displayImageReference": {
          "vectorImage": {
            "rootUrl": "https://media.licdn.com/dms/image/",
            "artifacts": [
              { "width": 100, "height": 100, "fileIdentifyingUrlPathSegment": "100_100/x.jpg" },
              { "width": 800, "height": 800, "fileIdentifyingUrlPathSegment": "800_800/x.jpg" },
              { "width": 400, "height": 400, "fileIdentifyingUrlPathSegment": "400_400/x.jpg" }
            ]
          }
        }
      }
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Position",
      "entityUrn": "urn:li:fs_position:1",
      "title": "Senior Backend Engineer",
      "companyName": "Razorstack",
      "employmentType": "FULL_TIME",
      "geoLocationName": "Bengaluru",
      "description": "Own the settlement platform.",
      "timePeriod": { "startDate": { "month": 4, "year": 2022 } }
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Position",
      "entityUrn": "urn:li:fs_position:2",
      "title": "Backend Engineer",
      "companyName": "Finlytics",
      "timePeriod": {
        "startDate": { "month": 1, "year": 2019 },
        "endDate": { "month": 3, "year": 2022 }
      }
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Education",
      "entityUrn": "urn:li:fs_education:1",
      "schoolName": "BITS Pilani",
      "degreeName": "B.E. Computer Science",
      "timePeriod": { "startDate": { "year": 2013 }, "endDate": { "year": 2017 } }
    },
    {
      "$type": "com.linkedin.voyager.identity.profile.Skill",
      "entityUrn": "urn:li:fs_skill:1",
      "name": "Go"
    }
  ]
}`

func TestAssembleCoreIdentity(t *testing.T) {
	profile, err := Assemble("priya-nair-eng", map[string][]byte{"profileView": []byte(profileViewPayload)})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}

	if profile.Identity.Name != "Priya Nair" {
		t.Errorf("Name = %q, want %q", profile.Identity.Name, "Priya Nair")
	}
	if profile.Headline != "Senior Backend Engineer" {
		t.Errorf("Headline = %q", profile.Headline)
	}
	if profile.Location != "Bengaluru, Karnataka, India" {
		t.Errorf("Location = %q", profile.Location)
	}
	if profile.About == "" {
		t.Error("About should carry the summary")
	}
}

// The image URL is rootUrl + the *largest* artifact, not the first one.
func TestAssemblePicksHighestResolutionImage(t *testing.T) {
	profile, _ := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})

	if profile.Images.ProfilePhoto == nil {
		t.Fatal("profile photo should be present")
	}
	want := "https://media.licdn.com/dms/image/800_800/x.jpg"
	if *profile.Images.ProfilePhoto != want {
		t.Errorf("ProfilePhoto = %q, want the 800px artifact %q", *profile.Images.ProfilePhoto, want)
	}
}

// No background image in the payload means nil, never a placeholder.
func TestAssembleMissingImageStaysNil(t *testing.T) {
	profile, _ := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})

	if profile.Images.BackgroundPhoto != nil {
		t.Errorf("BackgroundPhoto = %q, want nil", *profile.Images.BackgroundPhoto)
	}
}

func TestAssembleExperience(t *testing.T) {
	profile, _ := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})

	if len(profile.Experience) != 2 {
		t.Fatalf("got %d experience entries, want 2", len(profile.Experience))
	}

	current := profile.Experience[0]
	if current.Company != "Razorstack" {
		t.Errorf("Company = %q", current.Company)
	}
	// SCREAMING_SNAKE becomes readable text.
	if current.EmploymentType != "Full time" {
		t.Errorf("EmploymentType = %q, want %q", current.EmploymentType, "Full time")
	}
	// A start with no end means the role is current.
	if !current.Period.Present {
		t.Error("a role with no end date should be marked Present")
	}
	if current.Period.End != nil {
		t.Error("an ongoing role must not gain an invented end date")
	}

	past := profile.Experience[1]
	if past.Period.Present {
		t.Error("a role with an end date must not be marked Present")
	}
	if past.Period.End == nil || past.Period.End.Year != 2022 {
		t.Errorf("past role end = %+v, want 2022", past.Period.End)
	}
}

func TestAssembleEducationAndSkills(t *testing.T) {
	profile, _ := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})

	if len(profile.Education) != 1 || profile.Education[0].School != "BITS Pilani" {
		t.Errorf("education = %+v", profile.Education)
	}
	if len(profile.Skills) != 1 || profile.Skills[0].Name != "Go" {
		t.Errorf("skills = %+v", profile.Skills)
	}
}

// A section the profile does not have must be an empty slice, never nil, so it
// serialises as [] rather than null.
func TestAssembleAbsentSectionsAreEmptyNotNil(t *testing.T) {
	profile, _ := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})

	if profile.Patents == nil {
		t.Error("Patents is nil; an absent section must be an empty slice")
	}
	if len(profile.Patents) != 0 {
		t.Errorf("Patents has %d entries, want 0", len(profile.Patents))
	}
	if profile.Publications == nil || profile.Honors == nil || profile.Courses == nil {
		t.Error("every list section must be non-nil after assembly")
	}
}

// Secondary calls failing must degrade one section, not the whole profile.
func TestAssembleSurvivesMissingSecondaryCalls(t *testing.T) {
	profile, err := Assemble("priya", map[string][]byte{"profileView": []byte(profileViewPayload)})
	if err != nil {
		t.Fatalf("Assemble failed with only the essential call present: %v", err)
	}
	if profile.Contact != nil {
		t.Error("contact should be nil when the contactInfo call did not run")
	}
	if profile.Recommends == nil {
		t.Error("recommendations should be an empty slice, not nil")
	}
}

// A profile whose sections are in "included" but unlisted in "data" must still
// be read — otherwise a decoration change silently empties the response.
func TestAssembleFallsBackToScanningIncluded(t *testing.T) {
	payload := `{
      "data": { "*profile": "urn:li:fs_profile:X" },
      "included": [
        { "$type": "com.linkedin.voyager.identity.profile.Profile", "entityUrn": "urn:li:fs_profile:X",
          "firstName": "Sam", "lastName": "Lee" },
        { "$type": "com.linkedin.voyager.identity.profile.Position", "entityUrn": "urn:li:fs_position:9",
          "title": "Engineer", "companyName": "Acme" }
      ]
    }`

	profile, err := Assemble("sam", map[string][]byte{"profileView": []byte(payload)})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}
	if len(profile.Experience) != 1 || profile.Experience[0].Company != "Acme" {
		t.Errorf("fallback scan failed: experience = %+v", profile.Experience)
	}
}

func TestAssembleRejectsGarbage(t *testing.T) {
	if _, err := Assemble("x", map[string][]byte{"profileView": []byte("<html>blocked</html>")}); err == nil {
		t.Fatal("Assemble accepted an HTML body, want an error")
	}
}
