package voyager

import (
	"testing"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

// A dash profile payload, shaped exactly like the live one: data names the
// subject, every section hangs off a CollectionResponse, and referenced entities
// (company, school, geo, industry, employment type) are separate objects.
//
// The connection profile is listed *before* the subject on purpose — included
// carries other people, so anything that takes the first Profile is wrong.
const dashProfilePayload = `{
  "data": {
    "$type": "com.linkedin.restli.common.CollectionResponse",
    "*elements": ["urn:li:fsd_profile:SUBJECT"],
    "paging": { "count": 10, "start": 0 }
  },
  "included": [
    {
      "$type": "com.linkedin.voyager.dash.identity.profile.Profile",
      "entityUrn": "urn:li:fsd_profile:OTHER",
      "firstName": "Rahul",
      "lastName": "Verma",
      "publicIdentifier": "rahul-verma"
    },
    {
      "$type": "com.linkedin.voyager.dash.identity.profile.Profile",
      "entityUrn": "urn:li:fsd_profile:SUBJECT",
      "firstName": "Priya",
      "lastName": "Nair",
      "headline": "Senior Backend Engineer",
      "summary": "Backend engineer focused on payment rails.",
      "publicIdentifier": "priya-nair-eng",
      "*industry": "urn:li:fsd_industry:4",
      "geoLocation": { "*geo": "urn:li:fsd_geo:1" },
      "pronounUnion": { "standardizedPronoun": "SHE_HER" },
      "volunteerCauses": ["SCIENCE_AND_TECHNOLOGY", "EDUCATION", "EDUCATION"],
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
      },
      "*profilePositionGroups": "urn:li:collectionResponse:GROUPS",
      "*profileEducations": "urn:li:collectionResponse:EDUCATIONS",
      "*profileSkills": "urn:li:collectionResponse:SKILLS",
      "*profileCertifications": "urn:li:collectionResponse:CERTS",
      "*profileProjects": "urn:li:collectionResponse:PROJECTS",
      "*profilePublications": "urn:li:collectionResponse:PUBS",
      "*profileHonors": "urn:li:collectionResponse:HONORS",
      "*profileCourses": "urn:li:collectionResponse:COURSES",
      "*profileLanguages": "urn:li:collectionResponse:LANGUAGES",
      "*profileVolunteerExperiences": "urn:li:collectionResponse:VOLUNTEER",
      "*profilePatents": "urn:li:collectionResponse:PATENTS"
    },
    { "$type": "com.linkedin.voyager.dash.common.Industry",
      "entityUrn": "urn:li:fsd_industry:4", "name": "Financial Services" },
    { "$type": "com.linkedin.voyager.dash.common.Geo",
      "entityUrn": "urn:li:fsd_geo:1", "defaultLocalizedName": "Bengaluru, Karnataka, India" },
    { "$type": "com.linkedin.voyager.dash.identity.profile.EmploymentType",
      "entityUrn": "urn:li:fsd_employmentType:1", "name": "Full-time" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:GROUPS",
      "*elements": ["urn:li:fsd_positionGroup:1", "urn:li:fsd_positionGroup:2"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.PositionGroup",
      "entityUrn": "urn:li:fsd_positionGroup:1",
      "companyName": "Razorstack",
      "*company": "urn:li:fsd_company:1",
      "*profilePositionInPositionGroup": "urn:li:collectionResponse:POS1" },
    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:POS1",
      "*elements": ["urn:li:fsd_position:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Position",
      "entityUrn": "urn:li:fsd_position:1",
      "title": "Staff Engineer",
      "*employmentType": "urn:li:fsd_employmentType:1",
      "geoLocationName": "Bengaluru, Karnataka, India",
      "dateRange": { "start": { "month": 3, "year": 2022 } } },
    { "$type": "com.linkedin.voyager.dash.identity.profile.PositionGroup",
      "entityUrn": "urn:li:fsd_positionGroup:2",
      "companyName": "Fintrail",
      "*profilePositionInPositionGroup": "urn:li:collectionResponse:POS2" },
    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:POS2",
      "*elements": ["urn:li:fsd_position:2"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Position",
      "entityUrn": "urn:li:fsd_position:2",
      "title": "Backend Engineer",
      "companyName": "Fintrail",
      "dateRange": { "start": { "year": 2019 }, "end": { "year": 2022 } } },
    { "$type": "com.linkedin.voyager.dash.organization.Company",
      "entityUrn": "urn:li:fsd_company:1",
      "name": "Razorstack",
      "logo": { "vectorImage": { "rootUrl": "https://media.licdn.com/logo/",
        "artifacts": [{ "width": 200, "height": 200, "fileIdentifyingUrlPathSegment": "200_200/l.png" }] } } },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:EDUCATIONS",
      "*elements": ["urn:li:fsd_education:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Education",
      "entityUrn": "urn:li:fsd_education:1",
      "schoolName": "BITS Pilani",
      "*school": "urn:li:fsd_school:1",
      "degreeName": "B.E. Computer Science",
      "fieldOfStudy": "Computer Science",
      "grade": "8.9 CGPA",
      "dateRange": { "start": { "year": 2013 }, "end": { "year": 2017 } } },
    { "$type": "com.linkedin.voyager.dash.organization.School",
      "entityUrn": "urn:li:fsd_school:1", "name": "BITS Pilani" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:SKILLS",
      "*elements": ["urn:li:fsd_skill:1", "urn:li:fsd_skill:2", "urn:li:fsd_skill:3"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Skill",
      "entityUrn": "urn:li:fsd_skill:1", "name": "Go" },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Skill",
      "entityUrn": "urn:li:fsd_skill:2", "name": "Kubernetes" },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Skill",
      "entityUrn": "urn:li:fsd_skill:3", "name": "Go" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:CERTS",
      "*elements": ["urn:li:fsd_certification:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Certification",
      "entityUrn": "urn:li:fsd_certification:1",
      "name": "Certified Kubernetes Administrator",
      "authority": "CNCF",
      "licenseNumber": "CKA-1234",
      "dateRange": { "start": { "month": 6, "year": 2021 } } },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:PROJECTS",
      "*elements": ["urn:li:fsd_project:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Project",
      "entityUrn": "urn:li:fsd_project:1",
      "title": "Ledger rewrite",
      "description": "Moved settlement onto an append-only ledger.",
      "contributors": [
        { "standardizedContributor": { "*profile": "urn:li:fsd_profile:OTHER" } },
        { "customContributor": { "name": "Meera Iyer" } }
      ],
      "dateRange": { "start": { "year": 2023 } } },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:PUBS",
      "*elements": ["urn:li:fsd_publication:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Publication",
      "entityUrn": "urn:li:fsd_publication:1",
      "name": "Idempotent settlement at scale",
      "publisher": "ACM Queue",
      "publishedOn": { "day": 12, "month": 4, "year": 2024 } },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:HONORS",
      "*elements": ["urn:li:fsd_honor:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Honor",
      "entityUrn": "urn:li:fsd_honor:1",
      "title": "Engineering Excellence Award",
      "issuer": "Razorstack",
      "issuedOn": { "month": 11, "year": 2023 } },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:COURSES",
      "*elements": ["urn:li:fsd_course:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Course",
      "entityUrn": "urn:li:fsd_course:1", "name": "Distributed Systems" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:LANGUAGES",
      "*elements": ["urn:li:fsd_language:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.Language",
      "entityUrn": "urn:li:fsd_language:1",
      "name": "Malayalam", "proficiency": "NATIVE_OR_BILINGUAL" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:VOLUNTEER",
      "*elements": ["urn:li:fsd_volunteer:1"] },
    { "$type": "com.linkedin.voyager.dash.identity.profile.VolunteerExperience",
      "entityUrn": "urn:li:fsd_volunteer:1",
      "role": "Mentor", "companyName": "Code for Bengaluru", "cause": "EDUCATION" },

    { "$type": "com.linkedin.restli.common.CollectionResponse",
      "entityUrn": "urn:li:collectionResponse:PATENTS",
      "*elements": [] }
  ]
}`

// assembleFixture assembles the payload above, failing the test if it cannot.
func assembleFixture(t *testing.T) *models.Profile {
	t.Helper()
	profile, err := Assemble("priya-nair-eng", map[string][]byte{"dashProfile": []byte(dashProfilePayload)})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}
	return profile
}

func TestAssembleCoreIdentity(t *testing.T) {
	profile := assembleFixture(t)

	if profile.Identity.Name != "Priya Nair" {
		t.Errorf("Name = %q, want %q", profile.Identity.Name, "Priya Nair")
	}
	if profile.Headline != "Senior Backend Engineer" {
		t.Errorf("Headline = %q", profile.Headline)
	}
	if profile.About == "" {
		t.Error("About should carry the summary")
	}
	// SHE_HER reads as a pronoun pair, not a sentence.
	if profile.Identity.Pronouns != "She/her" {
		t.Errorf("Pronouns = %q, want %q", profile.Identity.Pronouns, "She/her")
	}
}

// Location and industry are urns in dash; both have to be followed.
func TestAssembleResolvesReferencedNames(t *testing.T) {
	profile := assembleFixture(t)

	if profile.Location != "Bengaluru, Karnataka, India" {
		t.Errorf("Location = %q, want the resolved geo name", profile.Location)
	}
	if profile.Industry != "Financial Services" {
		t.Errorf("Industry = %q, want the resolved industry name", profile.Industry)
	}
}

// included carries connections and contributors too, so the subject must be read
// from data rather than picked off the front of the list.
func TestAssemblePicksTheSubjectNotTheFirstProfile(t *testing.T) {
	profile := assembleFixture(t)

	if profile.PublicID != "priya-nair-eng" {
		t.Errorf("PublicID = %q, want the subject", profile.PublicID)
	}
	if profile.Identity.FirstName == "Rahul" {
		t.Fatal("assembled a connection's profile instead of the subject")
	}
}

// The image URL is rootUrl + the *largest* artifact, not the first one.
func TestAssemblePicksHighestResolutionImage(t *testing.T) {
	profile := assembleFixture(t)

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
	profile := assembleFixture(t)

	if profile.Images.BackgroundPhoto != nil {
		t.Errorf("BackgroundPhoto = %q, want nil", *profile.Images.BackgroundPhoto)
	}
}

// Experience is grouped by company in dash: profile -> group -> roles.
func TestAssembleFlattensGroupedExperience(t *testing.T) {
	profile := assembleFixture(t)

	if len(profile.Experience) != 2 {
		t.Fatalf("got %d experience entries, want 2", len(profile.Experience))
	}

	current := profile.Experience[0]
	// The role carries no company of its own; the group's must be used.
	if current.Company != "Razorstack" {
		t.Errorf("Company = %q, want the group's company", current.Company)
	}
	if current.CompanyLogo == nil {
		t.Error("company logo should be resolved from the group's company")
	}
	if current.EmploymentType != "Full-time" {
		t.Errorf("EmploymentType = %q, want %q", current.EmploymentType, "Full-time")
	}
	// A start with no end means the role is current.
	if !current.Period.Present {
		t.Error("a role with no end date should be marked Present")
	}
	if current.Period.End != nil {
		t.Error("an ongoing role must not gain an invented end date")
	}

	past := profile.Experience[1]
	if past.Company != "Fintrail" {
		t.Errorf("Company = %q", past.Company)
	}
	if past.Period.Present {
		t.Error("a role with an end date must not be marked Present")
	}
	if past.Period.End == nil || past.Period.End.Year != 2022 {
		t.Errorf("past role end = %+v, want 2022", past.Period.End)
	}
}

func TestAssembleEducationAndSkills(t *testing.T) {
	profile := assembleFixture(t)

	if len(profile.Education) != 1 {
		t.Fatalf("got %d education entries, want 1", len(profile.Education))
	}
	education := profile.Education[0]
	if education.School != "BITS Pilani" || education.Degree != "B.E. Computer Science" {
		t.Errorf("education = %+v", education)
	}
	if education.Grade != "8.9 CGPA" {
		t.Errorf("Grade = %q", education.Grade)
	}

	// The payload lists "Go" twice; the same skill must appear once.
	if len(profile.Skills) != 2 {
		t.Fatalf("skills = %+v, want 2 deduplicated entries", profile.Skills)
	}
}

// Every remaining section reaches its entities through a CollectionResponse.
func TestAssembleFollowsSectionCollections(t *testing.T) {
	profile := assembleFixture(t)

	if len(profile.Certification) != 1 || profile.Certification[0].Authority != "CNCF" {
		t.Errorf("certifications = %+v", profile.Certification)
	}
	if len(profile.Publications) != 1 || profile.Publications[0].Date == nil {
		t.Errorf("publications = %+v", profile.Publications)
	}
	if len(profile.Honors) != 1 || profile.Honors[0].Date == nil {
		t.Errorf("honors = %+v", profile.Honors)
	}
	if len(profile.Courses) != 1 || profile.Courses[0].Name != "Distributed Systems" {
		t.Errorf("courses = %+v", profile.Courses)
	}
	if len(profile.Languages) != 1 || profile.Languages[0].Proficiency != "Native or bilingual" {
		t.Errorf("languages = %+v", profile.Languages)
	}
	if len(profile.Volunteering) != 1 || profile.Volunteering[0].Organization != "Code for Bengaluru" {
		t.Errorf("volunteering = %+v", profile.Volunteering)
	}
}

// A contributor is either a urn to a profile or free text, and both become names.
func TestAssembleResolvesProjectContributors(t *testing.T) {
	profile := assembleFixture(t)

	if len(profile.Projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(profile.Projects))
	}
	members := profile.Projects[0].Members
	if len(members) != 2 {
		t.Fatalf("members = %+v, want both contributors named", members)
	}
	if members[0] != "Rahul Verma" {
		t.Errorf("members[0] = %q, want the resolved profile name", members[0])
	}
	if members[1] != "Meera Iyer" {
		t.Errorf("members[1] = %q, want the free-text name", members[1])
	}
}

// dash states causes outright, so they are no longer inferred from volunteering.
func TestAssembleReadsStatedCauses(t *testing.T) {
	profile := assembleFixture(t)

	if len(profile.Causes) != 2 {
		t.Fatalf("causes = %+v, want 2 deduplicated entries", profile.Causes)
	}
	if profile.Causes[0] != "Science and technology" {
		t.Errorf("causes[0] = %q", profile.Causes[0])
	}
}

// A section the profile does not have must be an empty slice, never nil, so it
// serialises as [] rather than null.
func TestAssembleAbsentSectionsAreEmptyNotNil(t *testing.T) {
	profile := assembleFixture(t)

	if profile.Patents == nil {
		t.Error("Patents is nil; an empty collection must give an empty slice")
	}
	if len(profile.Patents) != 0 {
		t.Errorf("Patents has %d entries, want 0", len(profile.Patents))
	}
	// These have no pointer on the profile at all.
	if profile.TestScores == nil || profile.Organizations == nil {
		t.Error("a section with no pointer must still be an empty slice")
	}
	if profile.Featured == nil || profile.Services == nil || profile.CareerBreaks == nil {
		t.Error("sections no reachable endpoint exposes must be empty, not nil")
	}
}

// The recommendations call failing must degrade one section, not the profile.
func TestAssembleSurvivesMissingSecondaryCalls(t *testing.T) {
	profile := assembleFixture(t)

	if profile.Contact != nil {
		t.Error("contact should be nil: no endpoint exposes it any more")
	}
	if profile.Recommends == nil {
		t.Error("recommendations should be an empty slice, not nil")
	}
}

// Recommendations still come from the legacy fs_ endpoint, which answers 200.
func TestAssembleReadsLegacyRecommendations(t *testing.T) {
	recommendations := `{
      "data": { "*elements": ["urn:li:fs_recommendation:1"] },
      "included": [
        { "$type": "com.linkedin.voyager.identity.profile.Recommendation",
          "entityUrn": "urn:li:fs_recommendation:1",
          "recommendationText": "Priya rebuilt our settlement pipeline.",
          "relationship": "MANAGER",
          "*recommender": "urn:li:fs_miniProfile:R1" },
        { "$type": "com.linkedin.voyager.identity.shared.MiniProfile",
          "entityUrn": "urn:li:fs_miniProfile:R1",
          "firstName": "Arjun", "lastName": "Rao", "occupation": "VP Engineering" }
      ]
    }`

	profile, err := Assemble("priya-nair-eng", map[string][]byte{
		"dashProfile":     []byte(dashProfilePayload),
		"recommendations": []byte(recommendations),
	})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}

	if len(profile.Recommends) != 1 {
		t.Fatalf("got %d recommendations, want 1", len(profile.Recommends))
	}
	if profile.Recommends[0].RecommenderName != "Arjun Rao" {
		t.Errorf("RecommenderName = %q", profile.Recommends[0].RecommenderName)
	}
}

// A profile whose position groups are missing must still yield its roles —
// otherwise a decoration change silently empties experience.
func TestAssembleFallsBackToScanningIncluded(t *testing.T) {
	payload := `{
      "data": { "*elements": ["urn:li:fsd_profile:X"] },
      "included": [
        { "$type": "com.linkedin.voyager.dash.identity.profile.Profile",
          "entityUrn": "urn:li:fsd_profile:X",
          "firstName": "Sam", "lastName": "Lee", "publicIdentifier": "sam" },
        { "$type": "com.linkedin.voyager.dash.identity.profile.Position",
          "entityUrn": "urn:li:fsd_position:9",
          "title": "Engineer", "companyName": "Acme" }
      ]
    }`

	profile, err := Assemble("sam", map[string][]byte{"dashProfile": []byte(payload)})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}
	if len(profile.Experience) != 1 || profile.Experience[0].Company != "Acme" {
		t.Errorf("fallback scan failed: experience = %+v", profile.Experience)
	}
}

// A payload carrying no identity is a blocked or deleted profile, and the caller
// must be able to tell: the name stays empty rather than being invented.
func TestAssembleUnknownSubjectLeavesIdentityEmpty(t *testing.T) {
	payload := `{ "data": { "*elements": [] }, "included": [] }`

	profile, err := Assemble("ghost", map[string][]byte{"dashProfile": []byte(payload)})
	if err != nil {
		t.Fatalf("Assemble returned an error: %v", err)
	}
	if profile.Identity.Name != "" {
		t.Errorf("Name = %q, want empty", profile.Identity.Name)
	}
}

func TestAssembleRejectsGarbage(t *testing.T) {
	if _, err := Assemble("x", map[string][]byte{"dashProfile": []byte("<html>blocked</html>")}); err == nil {
		t.Fatal("Assemble accepted an HTML body, want an error")
	}
}
