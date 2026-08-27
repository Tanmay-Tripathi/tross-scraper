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

	view, err := NewGraph(bodies["profileView"])
	if err != nil {
		return nil, err
	}

	applyProfileView(profile, view)
	applyStandaloneProfile(profile, bodies["profile"])
	applyContactInfo(profile, bodies["contactInfo"])
	applySkills(profile, bodies["skills"])
	applyRecommendations(profile, bodies["recommendations"])

	// Only exposed by LinkedIn's newer dash endpoints; empty rather than fabricated.
	profile.Featured = []models.Featured{}
	profile.Services = []string{}
	profile.CareerBreaks = []models.CareerBreak{}
	profile.Causes = causesFrom(profile.Volunteering)

	return profile, nil
}

// applyProfileView reads the one call that carries most of the profile.
func applyProfileView(out *models.Profile, graph *Graph) {
	var data ProfileViewData
	_ = json.Unmarshal(graph.Data, &data)

	if data.Profile != "" {
		var core Profile
		if graph.Resolve(data.Profile, &core) {
			applyCore(out, core)
		}
	}
	// Some payloads omit the pointer.
	if out.Identity.Name == "" {
		if cores := ByType[Profile](graph, ".Profile"); len(cores) > 0 {
			applyCore(out, cores[0])
		}
	}

	out.Experience = mapExperience(resolveOrScan[Position](graph, data.PositionView.Elements, ".Position"))
	out.Education = mapEducation(resolveOrScan[Education](graph, data.EducationView.Elements, ".Education"))
	out.Certification = mapCertifications(resolveOrScan[Certification](graph, data.CertificationView.Elements, ".Certification"))
	out.Projects = mapProjects(resolveOrScan[Project](graph, data.ProjectView.Elements, ".Project"))
	out.Publications = mapPublications(resolveOrScan[Publication](graph, data.PublicationView.Elements, ".Publication"))
	out.Patents = mapPatents(resolveOrScan[Patent](graph, data.PatentView.Elements, ".Patent"))
	out.Honors = mapHonors(resolveOrScan[Honor](graph, data.HonorView.Elements, ".Honor"))
	out.Courses = mapCourses(resolveOrScan[Course](graph, data.CourseView.Elements, ".Course"))
	out.Languages = mapLanguages(resolveOrScan[Language](graph, data.LanguageView.Elements, ".Language"))
	out.Organizations = mapOrganizations(resolveOrScan[Organization](graph, data.OrganizationView.Elements, ".Organization"))
	out.Volunteering = mapVolunteering(resolveOrScan[VolunteerExperience](graph, data.VolunteerExperienceView.Elements, ".VolunteerExperience"))
	out.TestScores = mapTestScores(resolveOrScan[TestScore](graph, data.TestScoreView.Elements, ".TestScore"))

	if len(out.Skills) == 0 {
		out.Skills = mapSkills(resolveOrScan[Skill](graph, data.SkillView.Elements, ".Skill"))
	}
}

// resolveOrScan follows the listed urns, falling back to scanning Included by
// type. The fallback stops a decoration change silently emptying a section.
func resolveOrScan[T any](graph *Graph, urns []string, typeSuffix string) []T {
	if len(urns) > 0 {
		if resolved := ResolveAll[T](graph, urns); len(resolved) > 0 {
			return resolved
		}
	}
	return ByType[T](graph, typeSuffix)
}

func applyCore(out *models.Profile, core Profile) {
	out.Identity = models.Identity{
		FirstName: core.FirstName,
		LastName:  core.LastName,
		Name:      strings.TrimSpace(core.FirstName + " " + core.LastName),
	}
	out.Headline = core.Headline
	out.About = core.Summary
	out.Industry = core.IndustryName
	out.Location = firstNonEmpty(core.GeoLocationName, core.LocationName, core.GeoCountryName)

	out.Images = models.Images{
		ProfilePhoto:    optionalURL(core.ProfilePicture.URL()),
		BackgroundPhoto: optionalURL(core.BackgroundImage.URL()),
	}

	if core.PublicIdentifier != "" {
		out.PublicID = core.PublicIdentifier
	}
}

// applyStandaloneProfile fills what profileView left blank.
func applyStandaloneProfile(out *models.Profile, body []byte) {
	if len(body) == 0 {
		return
	}
	graph, err := NewGraph(body)
	if err != nil {
		return
	}

	var core Profile
	if json.Unmarshal(graph.Data, &core) != nil {
		return
	}

	if out.Identity.Name == "" && (core.FirstName != "" || core.LastName != "") {
		out.Identity = models.Identity{
			FirstName: core.FirstName,
			LastName:  core.LastName,
			Name:      strings.TrimSpace(core.FirstName + " " + core.LastName),
		}
	}
	out.Headline = firstNonEmpty(out.Headline, core.Headline)
	out.About = firstNonEmpty(out.About, core.Summary)
	out.Industry = firstNonEmpty(out.Industry, core.IndustryName)
	out.Location = firstNonEmpty(out.Location, core.GeoLocationName, core.LocationName)

	if out.Images.ProfilePhoto == nil {
		out.Images.ProfilePhoto = optionalURL(core.ProfilePicture.URL())
	}
	if out.Images.BackgroundPhoto == nil {
		out.Images.BackgroundPhoto = optionalURL(core.BackgroundImage.URL())
	}
}

func applyContactInfo(out *models.Profile, body []byte) {
	if len(body) == 0 {
		return
	}
	graph, err := NewGraph(body)
	if err != nil {
		return
	}

	var raw ContactInfo
	if json.Unmarshal(graph.Data, &raw) != nil {
		return
	}

	contact := &models.ContactInfo{
		Email:   raw.EmailAddress,
		Address: raw.Address,
	}
	if !raw.BirthDateOn.IsZero() {
		contact.Birthday = formatDate(raw.BirthDateOn)
	}
	for _, phone := range raw.PhoneNumbers {
		contact.PhoneNumbers = append(contact.PhoneNumbers, models.Phone{Number: phone.Number, Type: phone.Type})
	}
	for _, site := range raw.Websites {
		category := ""
		if site.Type.Standard != nil {
			category = site.Type.Standard.Category
		} else if site.Type.Custom != nil {
			category = site.Type.Custom.Label
		}
		contact.Websites = append(contact.Websites, models.Website{URL: site.URL, Category: category})
	}
	for _, handle := range raw.TwitterHandles {
		contact.Twitter = append(contact.Twitter, handle.Name)
	}

	// An object of blanks is not data; the contract wants null.
	if !contact.IsEmpty() {
		out.Contact = contact
	}
}

func applySkills(out *models.Profile, body []byte) {
	if len(body) == 0 {
		return
	}
	graph, err := NewGraph(body)
	if err != nil {
		return
	}
	if skills := ByType[Skill](graph, ".Skill"); len(skills) > 0 {
		out.Skills = mapSkills(skills)
	}
}

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

// causesFrom derives causes from volunteering entries — the only source this API
// exposes, and it reports only causes the profile actually stated.
func causesFrom(volunteering []models.Volunteering) []string {
	seen := map[string]bool{}
	causes := []string{}
	for _, entry := range volunteering {
		cause := strings.TrimSpace(entry.Cause)
		if cause == "" || seen[cause] {
			continue
		}
		seen[cause] = true
		causes = append(causes, cause)
	}
	return causes
}
