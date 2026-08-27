package voyager

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

// Assemble turns the bodies from FetchAll into a domain Profile. A section that is
// missing or malformed yields an empty slice, so one weak section cannot fail it.
func Assemble(publicID string, bodies map[string][]byte) (*models.Profile, error) {
	profile := &models.Profile{
		PublicID:  publicID,
		FetchedAt: time.Now().UTC(),
	}

	graph, err := NewGraph(bodies["dashProfile"])
	if err != nil {
		return nil, err
	}

	applyDashProfile(profile, graph, publicID)
	applyRecommendations(profile, bodies["recommendations"])

	// No endpoint we can still reach exposes these, so they stay empty rather
	// than being inferred from something that only resembles them.
	profile.OpenTo = []string{}
	profile.Featured = []models.Featured{}
	profile.Services = []string{}
	profile.CareerBreaks = []models.CareerBreak{}

	return profile, nil
}

// applyDashProfile reads the one call that carries the whole profile.
func applyDashProfile(out *models.Profile, graph *Graph, publicID string) {
	core, found := subjectProfile(graph, publicID)
	if !found {
		return
	}

	applyCore(out, core, graph)

	out.Experience = mapExperience(positions(graph, core))
	out.Education = mapEducation(educations(graph, core))
	out.Skills = mapSkills(Collection[Skill](graph, core.Skills))
	out.Certification = mapCertifications(certifications(graph, core))
	out.Projects = mapProjects(projects(graph, core))
	out.Publications = mapPublications(publications(graph, core))
	out.Patents = mapPatents(Collection[Patent](graph, core.Patents))
	out.Honors = mapHonors(Collection[Honor](graph, core.Honors))
	out.Courses = mapCourses(Collection[Course](graph, core.Courses))
	out.Languages = mapLanguages(Collection[Language](graph, core.Languages))
	out.Organizations = mapOrganizations(Collection[Organization](graph, core.Organizations))
	out.Volunteering = mapVolunteering(volunteering(graph, core))
	out.TestScores = mapTestScores(Collection[TestScore](graph, core.TestScores))
	out.Causes = causes(core.VolunteerCauses)
}

// subjectProfile resolves the member the query was for. data names them
// explicitly, and that matters: included also carries the profiles of
// connections, contributors and recommenders, so taking the first Profile it
// finds can return the wrong person.
func subjectProfile(graph *Graph, publicID string) (Profile, bool) {
	var list dashProfileList
	if json.Unmarshal(graph.Data, &list) == nil {
		for _, urn := range list.Elements {
			var core Profile
			if graph.Resolve(urn, &core) {
				return core, true
			}
		}
	}

	// Falling back, match on the slug we asked for rather than position.
	for _, core := range ByType[Profile](graph, ".Profile") {
		if strings.EqualFold(core.PublicIdentifier, publicID) {
			return core, true
		}
	}

	return Profile{}, false
}

func applyCore(out *models.Profile, core Profile, graph *Graph) {
	out.Identity = models.Identity{
		FirstName: core.FirstName,
		LastName:  core.LastName,
		Name:      strings.TrimSpace(core.FirstName + " " + core.LastName),
		Pronouns:  pronouns(core),
	}
	out.Headline = core.Headline
	out.About = core.Summary
	out.Industry = industryName(graph, core.IndustryUrn)
	out.Location = locationName(graph, core)

	out.Images = models.Images{
		ProfilePhoto:    optionalURL(core.ProfilePicture.URL()),
		BackgroundPhoto: optionalURL(core.BackgroundImage.URL()),
	}

	if core.PublicIdentifier != "" {
		out.PublicID = core.PublicIdentifier
	}
}

// positions flattens dash's grouped experience: the profile lists one group per
// company, and each group lists the roles held there.
func positions(graph *Graph, core Profile) []Position {
	groups := Collection[PositionGroup](graph, core.PositionGroups)

	out := make([]Position, 0, len(groups))
	for _, group := range groups {
		groupCompany := company(graph, group.CompanyUrn)

		for _, role := range Collection[Position](graph, group.Positions) {
			if role.CompanyName == "" {
				role.CompanyName = group.CompanyName
			}
			// Only some roles carry a company of their own.
			if role.Company = company(graph, role.CompanyUrn); role.Company == nil {
				role.Company = groupCompany
			}
			role.EmploymentType = employmentTypeName(graph, role.EmploymentTypeUrn)
			out = append(out, role)
		}
	}

	// A profile whose groups are absent can still list its positions directly.
	if len(out) == 0 {
		for _, role := range ByType[Position](graph, ".Position") {
			role.Company = company(graph, role.CompanyUrn)
			role.EmploymentType = employmentTypeName(graph, role.EmploymentTypeUrn)
			out = append(out, role)
		}
	}

	return out
}

func educations(graph *Graph, core Profile) []Education {
	entries := Collection[Education](graph, core.Educations)
	for i := range entries {
		entries[i].School = school(graph, entries[i].SchoolUrn)
	}
	return entries
}

func certifications(graph *Graph, core Profile) []Certification {
	entries := Collection[Certification](graph, core.Certifications)
	for i := range entries {
		entries[i].Company = company(graph, entries[i].CompanyUrn)
	}
	return entries
}

func volunteering(graph *Graph, core Profile) []VolunteerExperience {
	entries := Collection[VolunteerExperience](graph, core.VolunteerExperiences)
	for i := range entries {
		entries[i].Company = company(graph, entries[i].CompanyUrn)
	}
	return entries
}

func projects(graph *Graph, core Profile) []Project {
	entries := Collection[Project](graph, core.Projects)
	for i := range entries {
		entries[i].Members = contributorNames(graph, entries[i].Contributors)
	}
	return entries
}

func publications(graph *Graph, core Profile) []Publication {
	entries := Collection[Publication](graph, core.Publications)
	for i := range entries {
		entries[i].AuthorNames = contributorNames(graph, entries[i].Authors)
	}
	return entries
}

// contributorNames turns credited people into names, following a urn to their
// profile when dash gives one and using the free-text name when it does not.
func contributorNames(graph *Graph, list []contributor) []string {
	names := make([]string, 0, len(list))
	for _, item := range list {
		if item.Custom != nil {
			if name := strings.TrimSpace(item.Custom.Name); name != "" {
				names = append(names, name)
			}
			continue
		}
		if item.Standardized == nil {
			continue
		}

		var profile Profile
		if !graph.Resolve(item.Standardized.ProfileUrn, &profile) {
			continue
		}
		if name := strings.TrimSpace(profile.FirstName + " " + profile.LastName); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// causes reads the profile's stated causes. dash exposes them as a real field,
// so they no longer have to be inferred from volunteering entries.
func causes(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, item := range raw {
		cause := humanizeEnum(item)
		if cause == "" || seen[cause] {
			continue
		}
		seen[cause] = true
		out = append(out, cause)
	}
	return out
}

func pronouns(core Profile) string {
	if core.PronounUnion == nil {
		return ""
	}
	if custom := strings.TrimSpace(core.PronounUnion.Custom); custom != "" {
		return custom
	}
	// HE_HIM reads as "He/him", not "He him".
	return strings.ReplaceAll(humanizeEnum(core.PronounUnion.Standardized), " ", "/")
}

// locationName prefers the resolved geo, which is what the profile page shows.
func locationName(graph *Graph, core Profile) string {
	if core.GeoLocation != nil {
		var geo Geo
		if graph.Resolve(core.GeoLocation.GeoUrn, &geo) && geo.Name != "" {
			return geo.Name
		}
	}
	return core.LocationName
}

func industryName(graph *Graph, urn string) string {
	var industry Industry
	if urn == "" || !graph.Resolve(urn, &industry) {
		return ""
	}
	return industry.Name
}

func employmentTypeName(graph *Graph, urn string) string {
	var employment EmploymentType
	if urn == "" || !graph.Resolve(urn, &employment) {
		return ""
	}
	return employment.Name
}

func company(graph *Graph, urn string) *Company {
	var resolved Company
	if urn == "" || !graph.Resolve(urn, &resolved) {
		return nil
	}
	return &resolved
}

func school(graph *Graph, urn string) *School {
	var resolved School
	if urn == "" || !graph.Resolve(urn, &resolved) {
		return nil
	}
	return &resolved
}

// applyRecommendations reads the one legacy endpoint that still answers, so this
// is the only place the older fs_ shape is still parsed.
func applyRecommendations(out *models.Profile, body []byte) {
	if len(body) == 0 {
		out.Recommends = []models.Recommendation{}
		return
	}
	graph, err := NewGraph(body)
	if err != nil {
		out.Recommends = []models.Recommendation{}
		return
	}

	raw := ByType[Recommendation](graph, ".Recommendation")
	out.Recommends = make([]models.Recommendation, 0, len(raw))
	for _, item := range raw {
		if strings.TrimSpace(item.RecommendationText) == "" {
			continue
		}

		recommender := item.Recommender
		if recommender == nil && item.RecommenderUrn != "" {
			var resolved MiniProfile
			if graph.Resolve(item.RecommenderUrn, &resolved) {
				recommender = &resolved
			}
		}

		out.Recommends = append(out.Recommends, models.Recommendation{
			Text:                item.RecommendationText,
			Relationship:        item.Relationship,
			RecommenderName:     miniName(recommender),
			RecommenderHeadline: miniOccupation(recommender),
			RecommenderPhoto:    miniPhoto(recommender),
		})
	}
}
