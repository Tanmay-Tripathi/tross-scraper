package mapper

import (
	"fmt"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/response"
)

// ToProfileResult renders a profile against a section config, and is the single
// place the contract holds: enabled+empty gives [] or null, disabled omits the key.
func ToProfileResult(profile *models.Profile, sections config.SectionSet, cached bool) response.ProfileResult {
	result := response.ProfileResult{
		Profile:      toProfileCore(profile, sections),
		Sections:     map[string]any{},
		SectionOrder: sectionOrder(),
		Meta: response.ProfileMeta{
			PublicID:   profile.PublicID,
			ProfileURL: "https://www.linkedin.com/in/" + profile.PublicID + "/",
			Cached:     cached,
			FetchedAt:  profile.FetchedAt.Format(time.RFC3339),
		},
	}

	// add records a section only when enabled; values are non-nil, so empty is [].
	add := func(section config.Section, value any, populated bool) {
		if !sections.Enabled(section) {
			return
		}
		result.Sections[string(section)] = value
		result.Meta.Sections = append(result.Meta.Sections, string(section))
		if !populated {
			result.Meta.Unavailable = append(result.Meta.Unavailable, string(section))
		}
	}

	experience := toExperience(profile.Experience)
	add(config.SectionExperience, experience, len(experience) > 0)

	education := toEducation(profile.Education)
	add(config.SectionEducation, education, len(education) > 0)

	skills := toSkills(profile.Skills)
	add(config.SectionSkills, skills, len(skills) > 0)

	featured := toFeatured(profile.Featured)
	add(config.SectionFeatured, featured, len(featured) > 0)

	projects := toProjects(profile.Projects)
	add(config.SectionProjects, projects, len(projects) > 0)

	certifications := toCertifications(profile.Certification)
	add(config.SectionLicensesAndCertifications, certifications, len(certifications) > 0)

	courses := toCourses(profile.Courses)
	add(config.SectionCourses, courses, len(courses) > 0)

	recommendations := toRecommendations(profile.Recommends)
	add(config.SectionRecommendations, recommendations, len(recommendations) > 0)

	volunteering := toVolunteering(profile.Volunteering)
	add(config.SectionVolunteerExperience, volunteering, len(volunteering) > 0)

	publications := toPublications(profile.Publications)
	add(config.SectionPublications, publications, len(publications) > 0)

	patents := toPatents(profile.Patents)
	add(config.SectionPatents, patents, len(patents) > 0)

	honors := toHonors(profile.Honors)
	add(config.SectionHonorsAndAwards, honors, len(honors) > 0)

	testScores := toTestScores(profile.TestScores)
	add(config.SectionTestScores, testScores, len(testScores) > 0)

	languages := toLanguages(profile.Languages)
	add(config.SectionLanguages, languages, len(languages) > 0)

	organizations := toOrganizations(profile.Organizations)
	add(config.SectionOrganizations, organizations, len(organizations) > 0)

	services := nonNil(profile.Services)
	add(config.SectionServices, services, len(services) > 0)

	careerBreaks := toCareerBreaks(profile.CareerBreaks)
	add(config.SectionCareerBreaks, careerBreaks, len(careerBreaks) > 0)

	causes := nonNil(profile.Causes)
	add(config.SectionCauses, causes, len(causes) > 0)

	// Profile-level sections report availability too.
	if sections.Enabled(config.SectionAbout) {
		result.Meta.Sections = append(result.Meta.Sections, string(config.SectionAbout))
		if profile.About == "" {
			result.Meta.Unavailable = append(result.Meta.Unavailable, string(config.SectionAbout))
		}
	}
	if sections.Enabled(config.SectionContactInfo) {
		result.Meta.Sections = append(result.Meta.Sections, string(config.SectionContactInfo))
		if profile.Contact.IsEmpty() {
			result.Meta.Unavailable = append(result.Meta.Unavailable, string(config.SectionContactInfo))
		}
	}
	if sections.Enabled(config.SectionOpenTo) {
		result.Meta.Sections = append(result.Meta.Sections, string(config.SectionOpenTo))
		if len(profile.OpenTo) == 0 {
			result.Meta.Unavailable = append(result.Meta.Unavailable, string(config.SectionOpenTo))
		}
	}

	if result.Meta.Unavailable == nil {
		result.Meta.Unavailable = []string{}
	}

	return result
}

// sectionOrder is the canonical serialisation order for top-level sections.
func sectionOrder() []string {
	order := make([]string, 0, len(config.AllSections))
	for _, section := range config.AllSections {
		if section.IsProfileLevel() {
			continue
		}
		order = append(order, string(section))
	}
	return order
}

func toProfileCore(profile *models.Profile, sections config.SectionSet) response.ProfileCore {
	core := response.ProfileCore{
		Identity: response.Identity{
			Name:      profile.Identity.Name,
			FirstName: profile.Identity.FirstName,
			LastName:  profile.Identity.LastName,
			Pronouns:  optional(profile.Identity.Pronouns),
		},
		Headline: optional(profile.Headline),
		Location: optional(profile.Location),
		Industry: optional(profile.Industry),
		Images: response.Images{
			ProfilePhoto:    profile.Images.ProfilePhoto,
			BackgroundPhoto: profile.Images.BackgroundPhoto,
		},
	}

	// These three are togglable, so attach them only when enabled.
	if sections.Enabled(config.SectionAbout) {
		core.About = optional(profile.About)
	}
	if sections.Enabled(config.SectionContactInfo) {
		core.ContactInfo = toContactInfo(profile.Contact)
	}
	if sections.Enabled(config.SectionOpenTo) {
		core.OpenTo = nonNil(profile.OpenTo)
	}

	return core
}

// toContactInfo returns nil when empty, so the field is null not a blank object.
func toContactInfo(contact *models.ContactInfo) *response.ContactInfo {
	if contact.IsEmpty() {
		return nil
	}

	out := &response.ContactInfo{
		Email:        optional(contact.Email),
		Address:      optional(contact.Address),
		Birthday:     optional(contact.Birthday),
		PhoneNumbers: []response.Phone{},
		Websites:     []response.Website{},
		Twitter:      nonNil(contact.Twitter),
	}
	for _, phone := range contact.PhoneNumbers {
		out.PhoneNumbers = append(out.PhoneNumbers, response.Phone{Number: phone.Number, Type: phone.Type})
	}
	for _, site := range contact.Websites {
		out.Websites = append(out.Websites, response.Website{URL: site.URL, Category: site.Category})
	}
	return out
}

// ---- section converters ----

func toExperience(items []models.Experience) []response.Experience {
	out := make([]response.Experience, 0, len(items))
	for _, item := range items {
		out = append(out, response.Experience{
			Title:          item.Title,
			Company:        item.Company,
			CompanyLogo:    item.CompanyLogo,
			EmploymentType: item.EmploymentType,
			Location:       item.Location,
			Description:    item.Description,
			Period:         toDateRange(item.Period),
			Media:          toMedia(item.Media),
		})
	}
	return out
}

func toEducation(items []models.Education) []response.Education {
	out := make([]response.Education, 0, len(items))
	for _, item := range items {
		out = append(out, response.Education{
			School:       item.School,
			SchoolLogo:   item.SchoolLogo,
			Degree:       item.Degree,
			FieldOfStudy: item.FieldOfStudy,
			Grade:        item.Grade,
			Activities:   item.Activities,
			Description:  item.Description,
			Period:       toDateRange(item.Period),
			Media:        toMedia(item.Media),
		})
	}
	return out
}

func toSkills(items []models.Skill) []response.Skill {
	out := make([]response.Skill, 0, len(items))
	for _, item := range items {
		out = append(out, response.Skill{Name: item.Name, Endorsements: item.Endorsements})
	}
	return out
}

func toFeatured(items []models.Featured) []response.Featured {
	out := make([]response.Featured, 0, len(items))
	for _, item := range items {
		out = append(out, response.Featured{
			Title:       item.Title,
			Subtitle:    item.Subtitle,
			Description: item.Description,
			URL:         item.URL,
			Media:       toMedia(item.Media),
		})
	}
	return out
}

func toCertifications(items []models.Certification) []response.Certification {
	out := make([]response.Certification, 0, len(items))
	for _, item := range items {
		out = append(out, response.Certification{
			Name:          item.Name,
			Authority:     item.Authority,
			AuthorityLogo: item.AuthorityLogo,
			LicenseNumber: item.LicenseNumber,
			URL:           item.URL,
			Period:        toDateRange(item.Period),
			Media:         toMedia(item.Media),
		})
	}
	return out
}

func toProjects(items []models.Project) []response.Project {
	out := make([]response.Project, 0, len(items))
	for _, item := range items {
		out = append(out, response.Project{
			Title:       item.Title,
			Description: item.Description,
			URL:         item.URL,
			Period:      toDateRange(item.Period),
			Members:     nonNil(item.Members),
			Media:       toMedia(item.Media),
		})
	}
	return out
}

func toCourses(items []models.Course) []response.Course {
	out := make([]response.Course, 0, len(items))
	for _, item := range items {
		out = append(out, response.Course{Name: item.Name, Number: item.Number})
	}
	return out
}

func toRecommendations(items []models.Recommendation) []response.Recommendation {
	out := make([]response.Recommendation, 0, len(items))
	for _, item := range items {
		out = append(out, response.Recommendation{
			Text:                item.Text,
			RecommenderName:     item.RecommenderName,
			RecommenderHeadline: item.RecommenderHeadline,
			RecommenderPhoto:    item.RecommenderPhoto,
			Relationship:        item.Relationship,
		})
	}
	return out
}

func toVolunteering(items []models.Volunteering) []response.Volunteering {
	out := make([]response.Volunteering, 0, len(items))
	for _, item := range items {
		out = append(out, response.Volunteering{
			Role:         item.Role,
			Organization: item.Organization,
			Cause:        item.Cause,
			Description:  item.Description,
			Period:       toDateRange(item.Period),
			Media:        toMedia(item.Media),
		})
	}
	return out
}

func toPublications(items []models.Publication) []response.Publication {
	out := make([]response.Publication, 0, len(items))
	for _, item := range items {
		out = append(out, response.Publication{
			Name:        item.Name,
			Publisher:   item.Publisher,
			Description: item.Description,
			URL:         item.URL,
			Date:        formatDate(item.Date),
			Authors:     nonNil(item.Authors),
			Media:       toMedia(item.Media),
		})
	}
	return out
}

func toPatents(items []models.Patent) []response.Patent {
	out := make([]response.Patent, 0, len(items))
	for _, item := range items {
		out = append(out, response.Patent{
			Title:       item.Title,
			Issuer:      item.Issuer,
			Number:      item.Number,
			Description: item.Description,
			URL:         item.URL,
			Pending:     item.Pending,
			Date:        formatDate(item.Date),
			Inventors:   nonNil(item.Inventors),
			Media:       toMedia(item.Media),
		})
	}
	return out
}

func toHonors(items []models.Honor) []response.Honor {
	out := make([]response.Honor, 0, len(items))
	for _, item := range items {
		out = append(out, response.Honor{
			Title:       item.Title,
			Issuer:      item.Issuer,
			Description: item.Description,
			Date:        formatDate(item.Date),
			Media:       toMedia(item.Media),
		})
	}
	return out
}

func toTestScores(items []models.TestScore) []response.TestScore {
	out := make([]response.TestScore, 0, len(items))
	for _, item := range items {
		out = append(out, response.TestScore{
			Name:        item.Name,
			Score:       item.Score,
			Description: item.Description,
			Date:        formatDate(item.Date),
		})
	}
	return out
}

func toLanguages(items []models.Language) []response.Language {
	out := make([]response.Language, 0, len(items))
	for _, item := range items {
		out = append(out, response.Language{Name: item.Name, Proficiency: item.Proficiency})
	}
	return out
}

func toOrganizations(items []models.Organization) []response.Organization {
	out := make([]response.Organization, 0, len(items))
	for _, item := range items {
		out = append(out, response.Organization{
			Name:        item.Name,
			Role:        item.Role,
			Description: item.Description,
			Period:      toDateRange(item.Period),
		})
	}
	return out
}

func toCareerBreaks(items []models.CareerBreak) []response.CareerBreak {
	out := make([]response.CareerBreak, 0, len(items))
	for _, item := range items {
		out = append(out, response.CareerBreak{
			Reason:      item.Reason,
			Description: item.Description,
			Location:    item.Location,
			Period:      toDateRange(item.Period),
		})
	}
	return out
}

func toMedia(items []models.Media) []response.Media {
	out := make([]response.Media, 0, len(items))
	for _, item := range items {
		out = append(out, response.Media{
			Title:        item.Title,
			Description:  item.Description,
			URL:          item.URL,
			ThumbnailURL: item.ThumbnailURL,
			Type:         item.Type,
		})
	}
	return out
}

func toDateRange(period models.DateRange) response.DateRange {
	return response.DateRange{
		Start:   formatDate(period.Start),
		End:     formatDate(period.End),
		Present: period.Present,
	}
}

// formatDate keeps LinkedIn's precision: a year with no month stays "2022",
// never "2022-01", which would be fabrication.
func formatDate(date *models.Date) *string {
	if date == nil {
		return nil
	}
	var formatted string
	switch {
	case date.Year > 0 && date.Month > 0 && date.Day > 0:
		formatted = fmt.Sprintf("%04d-%02d-%02d", date.Year, date.Month, date.Day)
	case date.Year > 0 && date.Month > 0:
		formatted = fmt.Sprintf("%04d-%02d", date.Year, date.Month)
	case date.Year > 0:
		formatted = fmt.Sprintf("%04d", date.Year)
	default:
		return nil
	}
	return &formatted
}

// optional turns an empty string into null rather than "".
func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// nonNil guarantees a slice serialises as [], never null.
func nonNil[T any](items []T) []T {
	if items == nil {
		return []T{}
	}
	return items
}
