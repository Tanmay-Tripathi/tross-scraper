package voyager

import (
	"strings"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

// Every converter returns a non-nil slice, so an absent section is [] not null.

func mapExperience(raw []Position) []models.Experience {
	out := make([]models.Experience, 0, len(raw))
	for _, item := range raw {
		entry := models.Experience{
			Title:          item.Title,
			Company:        firstNonEmpty(item.CompanyName, companyName(item.Company)),
			EmploymentType: humanizeEnum(item.EmploymentType),
			Location:       firstNonEmpty(item.GeoLocationName, item.LocationName),
			Description:    item.Description,
			Period:         mapPeriod(item.DateRange),
			Media:          []models.Media{},
		}
		if item.Company != nil {
			entry.CompanyLogo = optionalURL(item.Company.Logo.URL())
		}
		out = append(out, entry)
	}
	return out
}

func mapEducation(raw []Education) []models.Education {
	out := make([]models.Education, 0, len(raw))
	for _, item := range raw {
		entry := models.Education{
			School:       firstNonEmpty(item.SchoolName, schoolName(item.School)),
			Degree:       item.DegreeName,
			FieldOfStudy: item.FieldOfStudy,
			Grade:        item.Grade,
			Activities:   item.Activities,
			Description:  item.Description,
			Period:       mapPeriod(item.DateRange),
			Media:        []models.Media{},
		}
		if item.School != nil {
			entry.SchoolLogo = optionalURL(item.School.Logo.URL())
		}
		out = append(out, entry)
	}
	return out
}

func mapSkills(raw []Skill) []models.Skill {
	out := make([]models.Skill, 0, len(raw))
	seen := map[string]bool{}
	for _, item := range raw {
		name := strings.TrimSpace(item.Name)
		// dash can list the same skill under more than one collection.
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, models.Skill{Name: name})
	}
	return out
}

func mapCertifications(raw []Certification) []models.Certification {
	out := make([]models.Certification, 0, len(raw))
	for _, item := range raw {
		entry := models.Certification{
			Name:          item.Name,
			Authority:     firstNonEmpty(item.Authority, companyName(item.Company)),
			LicenseNumber: item.LicenseNumber,
			URL:           item.URL,
			Period:        mapPeriod(item.DateRange),
			Media:         []models.Media{},
		}
		if item.Company != nil {
			entry.AuthorityLogo = optionalURL(item.Company.Logo.URL())
		}
		out = append(out, entry)
	}
	return out
}

func mapProjects(raw []Project) []models.Project {
	out := make([]models.Project, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Project{
			Title:       item.Title,
			Description: item.Description,
			URL:         item.URL,
			Period:      mapPeriod(item.DateRange),
			Members:     nonNilNames(item.Members),
			Media:       []models.Media{},
		})
	}
	return out
}

func mapPublications(raw []Publication) []models.Publication {
	out := make([]models.Publication, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Publication{
			Name:        item.Name,
			Publisher:   item.Publisher,
			Description: item.Description,
			URL:         item.URL,
			Date:        mapDate(item.PublishedOn),
			Authors:     nonNilNames(item.AuthorNames),
			Media:       []models.Media{},
		})
	}
	return out
}

func mapPatents(raw []Patent) []models.Patent {
	out := make([]models.Patent, 0, len(raw))
	for _, item := range raw {
		// A granted patent has an issue date, a pending one only a filing date.
		date := item.IssuedOn
		if date.IsZero() {
			date = item.FiledOn
		}
		out = append(out, models.Patent{
			Title:       item.Title,
			Issuer:      item.Issuer,
			Number:      item.Number,
			Description: item.Description,
			URL:         item.URL,
			Pending:     item.Pending,
			Date:        mapDate(date),
			Inventors:   []string{},
			Media:       []models.Media{},
		})
	}
	return out
}

func mapHonors(raw []Honor) []models.Honor {
	out := make([]models.Honor, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Honor{
			Title:       item.Title,
			Issuer:      item.Issuer,
			Description: item.Description,
			Date:        mapDate(item.IssuedOn),
			Media:       []models.Media{},
		})
	}
	return out
}

func mapCourses(raw []Course) []models.Course {
	out := make([]models.Course, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Course{Name: item.Name, Number: item.Number})
	}
	return out
}

func mapLanguages(raw []Language) []models.Language {
	out := make([]models.Language, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Language{
			Name:        item.Name,
			Proficiency: humanizeEnum(item.Proficiency),
		})
	}
	return out
}

func mapOrganizations(raw []Organization) []models.Organization {
	out := make([]models.Organization, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Organization{
			Name:        item.Name,
			Role:        item.Position,
			Description: item.Description,
			Period:      mapPeriod(item.DateRange),
		})
	}
	return out
}

func mapVolunteering(raw []VolunteerExperience) []models.Volunteering {
	out := make([]models.Volunteering, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.Volunteering{
			Role:         item.Role,
			Organization: item.CompanyName,
			Cause:        humanizeEnum(item.Cause),
			Description:  item.Description,
			Period:       mapPeriod(item.DateRange),
			Media:        []models.Media{},
		})
	}
	return out
}

func mapTestScores(raw []TestScore) []models.TestScore {
	out := make([]models.TestScore, 0, len(raw))
	for _, item := range raw {
		out = append(out, models.TestScore{
			Name:        item.Name,
			Score:       item.Score,
			Description: item.Description,
			Date:        mapDate(item.DateOn),
		})
	}
	return out
}

// ---- shared helpers ----

func mapDate(date *Date) *models.Date {
	if date.IsZero() {
		return nil
	}
	return &models.Date{Day: date.Day, Month: date.Month, Year: date.Year}
}

// mapPeriod converts a date range. A start with no end is how LinkedIn encodes
// "Present", so it is resolved here rather than guessed at downstream.
func mapPeriod(period *DateRange) models.DateRange {
	if period == nil {
		return models.DateRange{}
	}
	start := mapDate(period.Start)
	end := mapDate(period.End)
	return models.DateRange{
		Start:   start,
		End:     end,
		Present: start != nil && end == nil,
	}
}

// humanizeEnum turns SCREAMING_SNAKE enums readable: "FULL_TIME" -> "Full time".
func humanizeEnum(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed != strings.ToUpper(trimmed) {
		return trimmed
	}
	words := strings.Split(strings.ToLower(strings.ReplaceAll(trimmed, "_", " ")), " ")
	if len(words) == 0 {
		return trimmed
	}
	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	return strings.Join(words, " ")
}

// optionalURL returns nil for an absent image, never a link that would not load.
func optionalURL(url string) *string {
	if strings.TrimSpace(url) == "" {
		return nil
	}
	return &url
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func companyName(company *Company) string {
	if company == nil {
		return ""
	}
	return company.Name
}

func schoolName(school *School) string {
	if school == nil {
		return ""
	}
	return school.Name
}

// nonNilNames keeps a name list serialising as [] rather than null.
func nonNilNames(names []string) []string {
	if names == nil {
		return []string{}
	}
	return names
}

func miniName(profile *MiniProfile) string {
	if profile == nil {
		return ""
	}
	return strings.TrimSpace(profile.FirstName + " " + profile.LastName)
}

func miniOccupation(profile *MiniProfile) string {
	if profile == nil {
		return ""
	}
	return profile.Occupation
}

func miniPhoto(profile *MiniProfile) *string {
	if profile == nil {
		return nil
	}
	return optionalURL(profile.Picture.URL())
}
