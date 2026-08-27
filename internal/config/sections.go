package config

import (
	"fmt"
	"slices"
	"strings"
)

// Section names the profile sections the API can return. Config, validation, the
// response builder and the docs all read this one list.
type Section string

// Profile-level sections. These live inside the "profile" object.
const (
	SectionAbout       Section = "about"
	SectionContactInfo Section = "contactInfo"
	SectionOpenTo      Section = "openTo"
)

// List-shaped sections. These are top-level arrays in the response.
const (
	SectionExperience                Section = "experience"
	SectionEducation                 Section = "education"
	SectionServices                  Section = "services"
	SectionCareerBreaks              Section = "careerBreaks"
	SectionSkills                    Section = "skills"
	SectionFeatured                  Section = "featured"
	SectionLicensesAndCertifications Section = "licensesAndCertifications"
	SectionProjects                  Section = "projects"
	SectionCourses                   Section = "courses"
	SectionRecommendations           Section = "recommendations"
	SectionVolunteerExperience       Section = "volunteerExperience"
	SectionPublications              Section = "publications"
	SectionPatents                   Section = "patents"
	SectionHonorsAndAwards           Section = "honorsAndAwards"
	SectionTestScores                Section = "testScores"
	SectionLanguages                 Section = "languages"
	SectionOrganizations             Section = "organizations"
	SectionCauses                    Section = "causes"
)

// profileLevelSections nest under "profile" instead of being top-level arrays.
var profileLevelSections = []Section{
	SectionAbout, SectionContactInfo, SectionOpenTo,
}

// AllSections is the canonical order sections appear in the response.
var AllSections = []Section{
	SectionAbout,
	SectionContactInfo,
	SectionOpenTo,
	SectionExperience,
	SectionEducation,
	SectionSkills,
	SectionFeatured,
	SectionProjects,
	SectionLicensesAndCertifications,
	SectionCourses,
	SectionRecommendations,
	SectionVolunteerExperience,
	SectionPublications,
	SectionPatents,
	SectionHonorsAndAwards,
	SectionTestScores,
	SectionLanguages,
	SectionOrganizations,
	SectionServices,
	SectionCareerBreaks,
	SectionCauses,
}

// IsProfileLevel reports whether the section nests under "profile".
func (s Section) IsProfileLevel() bool {
	return slices.Contains(profileLevelSections, s)
}

// IsKnown reports whether s is a section the service supports.
func (s Section) IsKnown() bool {
	return slices.Contains(AllSections, s)
}

// SectionSet records which sections are switched on.
type SectionSet map[Section]bool

// Enabled reports whether the section appears in the response; absent means off.
func (s SectionSet) Enabled(section Section) bool {
	return s[section]
}

// Merge returns a copy of s with override applied, leaving the shared default intact.
func (s SectionSet) Merge(override SectionSet) SectionSet {
	merged := make(SectionSet, len(AllSections))
	for section, on := range s {
		merged[section] = on
	}
	for section, on := range override {
		merged[section] = on
	}
	return merged
}

// AllEnabled switches every supported section on — the default when config is silent.
func AllEnabled() SectionSet {
	set := make(SectionSet, len(AllSections))
	for _, section := range AllSections {
		set[section] = true
	}
	return set
}

// UnmarshalYAML rejects unsupported names, so a typo fails at startup.
func (s *SectionSet) UnmarshalYAML(unmarshal func(any) error) error {
	var raw map[string]bool
	if err := unmarshal(&raw); err != nil {
		return err
	}

	set := make(SectionSet, len(raw))
	var unknown []string
	for name, on := range raw {
		section := Section(strings.TrimSpace(name))
		if !section.IsKnown() {
			unknown = append(unknown, name)
			continue
		}
		set[section] = on
	}

	if len(unknown) > 0 {
		slices.Sort(unknown)
		return fmt.Errorf("unknown section(s): %s", strings.Join(unknown, ", "))
	}

	*s = set
	return nil
}
