package voyager

// These structs mirror LinkedIn's dash wire format and are intentionally
// permissive: every field is optional, and an unrecognised one must never break
// decoding.
//
// Two conventions matter. A field tagged `*something` holds a urn pointing into
// the response's included list rather than the value itself — dash normalises
// almost everything. A field tagged `json:"-"` is filled in by the assembler
// after it follows those urns, never by the decoder.

// dashProfileList is the collection the profile query answers with: a single
// element naming the subject. Reading it beats scanning included, which also
// carries the profiles of connections, contributors and recommenders.
type dashProfileList struct {
	Elements []string `json:"*elements"`
}

// collectionRef is dash's one level of indirection. A section pointer on the
// profile resolves to a CollectionResponse, whose elements are the section's
// entities — the hop legacy profileView did not have.
type collectionRef struct {
	Elements []string `json:"*elements"`
}

// Date is a partial date. LinkedIn frequently sends a year with no month.
type Date struct {
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

// IsZero reports whether the date carries nothing usable.
func (d *Date) IsZero() bool { return d == nil || (d.Year == 0 && d.Month == 0 && d.Day == 0) }

// DateRange is dash's start/end pair, replacing the legacy timePeriod. A nil End
// means "Present".
type DateRange struct {
	Start *Date `json:"start"`
	End   *Date `json:"end"`
}

// VectorImage is how LinkedIn delivers photos: a root URL plus sized artifacts.
// Neither half is a usable URL alone.
type VectorImage struct {
	RootURL   string          `json:"rootUrl"`
	Artifacts []ImageArtifact `json:"artifacts"`
}

type ImageArtifact struct {
	Width                         int    `json:"width"`
	Height                        int    `json:"height"`
	FileIdentifyingURLPathSegment string `json:"fileIdentifyingUrlPathSegment"`
}

// BestURL returns the highest-resolution URL, or "" — never a partial link that
// would fail to load.
func (v *VectorImage) BestURL() string {
	if v == nil || v.RootURL == "" || len(v.Artifacts) == 0 {
		return ""
	}

	best := v.Artifacts[0]
	for _, artifact := range v.Artifacts[1:] {
		if artifact.Width > best.Width {
			best = artifact
		}
	}
	if best.FileIdentifyingURLPathSegment == "" {
		return ""
	}
	return v.RootURL + best.FileIdentifyingURLPathSegment
}

// imageReference wraps a VectorImage, which LinkedIn nests inconsistently.
type imageReference struct {
	VectorImage *VectorImage `json:"vectorImage"`
}

// PictureFrame is the envelope around any image: a member photo nests it under
// displayImageReference, a company logo puts it directly under vectorImage.
type PictureFrame struct {
	DisplayImageReference *imageReference `json:"displayImageReference"`
	VectorImage           *VectorImage    `json:"vectorImage"`
}

// URL resolves whichever nesting this payload used.
func (p *PictureFrame) URL() string {
	if p == nil {
		return ""
	}
	if p.DisplayImageReference != nil {
		if url := p.DisplayImageReference.VectorImage.BestURL(); url != "" {
			return url
		}
	}
	return p.VectorImage.BestURL()
}

// Profile is the core member entity, and the root every section hangs off.
type Profile struct {
	EntityUrn        string        `json:"entityUrn"`
	FirstName        string        `json:"firstName"`
	LastName         string        `json:"lastName"`
	Headline         string        `json:"headline"`
	Summary          string        `json:"summary"`
	PublicIdentifier string        `json:"publicIdentifier"`
	LocationName     string        `json:"locationName"`
	ProfilePicture   *PictureFrame `json:"profilePicture"`
	BackgroundImage  *PictureFrame `json:"backgroundPicture"`
	VolunteerCauses  []string      `json:"volunteerCauses"`

	IndustryUrn string `json:"*industry"`

	// GeoLocation carries the displayed location, one urn further out.
	GeoLocation *struct {
		GeoUrn string `json:"*geo"`
	} `json:"geoLocation"`

	// PronounUnion is either a standardised enum or the member's own wording.
	PronounUnion *struct {
		Standardized string `json:"standardizedPronoun"`
		Custom       string `json:"customPronoun"`
	} `json:"pronounUnion"`

	PositionGroups       string `json:"*profilePositionGroups"`
	Educations           string `json:"*profileEducations"`
	Skills               string `json:"*profileSkills"`
	Certifications       string `json:"*profileCertifications"`
	Projects             string `json:"*profileProjects"`
	Publications         string `json:"*profilePublications"`
	Patents              string `json:"*profilePatents"`
	Honors               string `json:"*profileHonors"`
	Courses              string `json:"*profileCourses"`
	Languages            string `json:"*profileLanguages"`
	Organizations        string `json:"*profileOrganizations"`
	VolunteerExperiences string `json:"*profileVolunteerExperiences"`
	TestScores           string `json:"*profileTestScores"`
}

// Geo is a resolved place name. dash gives positions a name inline but the
// profile's own location only as a urn.
type Geo struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"defaultLocalizedName"`
}

// Industry is the profile's industry, reachable only through its urn.
type Industry struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"name"`
}

// EmploymentType turns a position's urn into "Internship", "Full-time", ….
type EmploymentType struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"name"`
}

// Company is the organisation behind a position, certification or volunteer role.
type Company struct {
	EntityUrn string        `json:"entityUrn"`
	Name      string        `json:"name"`
	Logo      *PictureFrame `json:"logo"`
	Universal string        `json:"universalName"`
}

// School is the institution behind an education entry.
type School struct {
	EntityUrn string        `json:"entityUrn"`
	Name      string        `json:"name"`
	Logo      *PictureFrame `json:"logo"`
}

// PositionGroup is one company a member worked at; the roles held there hang off
// it. This grouping is why experience needs three hops, not two.
type PositionGroup struct {
	EntityUrn   string     `json:"entityUrn"`
	CompanyName string     `json:"companyName"`
	CompanyUrn  string     `json:"*company"`
	Positions   string     `json:"*profilePositionInPositionGroup"`
	DateRange   *DateRange `json:"dateRange"`

	Company *Company `json:"-"`
}

// Position is one role. Only some carry a company urn of their own, so the
// assembler falls back to the group's company.
type Position struct {
	EntityUrn         string     `json:"entityUrn"`
	Title             string     `json:"title"`
	CompanyName       string     `json:"companyName"`
	CompanyUrn        string     `json:"*company"`
	Description       string     `json:"description"`
	LocationName      string     `json:"locationName"`
	GeoLocationName   string     `json:"geoLocationName"`
	EmploymentTypeUrn string     `json:"*employmentType"`
	DateRange         *DateRange `json:"dateRange"`

	Company        *Company `json:"-"`
	EmploymentType string   `json:"-"`
}

type Education struct {
	EntityUrn    string     `json:"entityUrn"`
	SchoolName   string     `json:"schoolName"`
	SchoolUrn    string     `json:"*school"`
	DegreeName   string     `json:"degreeName"`
	FieldOfStudy string     `json:"fieldOfStudy"`
	Grade        string     `json:"grade"`
	Activities   string     `json:"activities"`
	Description  string     `json:"description"`
	DateRange    *DateRange `json:"dateRange"`

	School *School `json:"-"`
}

type Skill struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"name"`
}

type Certification struct {
	EntityUrn     string     `json:"entityUrn"`
	Name          string     `json:"name"`
	Authority     string     `json:"authority"`
	LicenseNumber string     `json:"licenseNumber"`
	URL           string     `json:"url"`
	CompanyUrn    string     `json:"*company"`
	DateRange     *DateRange `json:"dateRange"`

	Company *Company `json:"-"`
}

// contributor is how dash credits a person on a project or publication: either a
// urn to their profile, or free text when they are not on LinkedIn.
type contributor struct {
	Standardized *struct {
		ProfileUrn string `json:"*profile"`
	} `json:"standardizedContributor"`
	Custom *struct {
		Name string `json:"name"`
	} `json:"customContributor"`
}

type Project struct {
	EntityUrn    string        `json:"entityUrn"`
	Title        string        `json:"title"`
	Description  string        `json:"description"`
	URL          string        `json:"url"`
	DateRange    *DateRange    `json:"dateRange"`
	Contributors []contributor `json:"contributors"`

	Members []string `json:"-"`
}

type Publication struct {
	EntityUrn   string        `json:"entityUrn"`
	Name        string        `json:"name"`
	Publisher   string        `json:"publisher"`
	Description string        `json:"description"`
	URL         string        `json:"url"`
	PublishedOn *Date         `json:"publishedOn"`
	Authors     []contributor `json:"authors"`

	AuthorNames []string `json:"-"`
}

type Honor struct {
	EntityUrn   string `json:"entityUrn"`
	Title       string `json:"title"`
	Issuer      string `json:"issuer"`
	Description string `json:"description"`
	IssuedOn    *Date  `json:"issuedOn"`
}

type Course struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"name"`
	Number    string `json:"number"`
}

type Language struct {
	EntityUrn   string `json:"entityUrn"`
	Name        string `json:"name"`
	Proficiency string `json:"proficiency"`
}

type VolunteerExperience struct {
	EntityUrn   string     `json:"entityUrn"`
	Role        string     `json:"role"`
	CompanyName string     `json:"companyName"`
	CompanyUrn  string     `json:"*company"`
	Cause       string     `json:"cause"`
	Description string     `json:"description"`
	DateRange   *DateRange `json:"dateRange"`

	Company *Company `json:"-"`
}

// Patents, organizations and test scores were empty on every profile probed so
// far, so these three follow dash's naming convention but are the only structs
// here not confirmed against a real payload. The collection pointers do exist on
// the profile, which is why the sections stay wired up rather than dropped.

type Patent struct {
	EntityUrn   string `json:"entityUrn"`
	Title       string `json:"title"`
	Issuer      string `json:"issuer"`
	Number      string `json:"applicationNumber"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Pending     bool   `json:"pending"`
	IssuedOn    *Date  `json:"issuedOn"`
	FiledOn     *Date  `json:"filedOn"`
}

type Organization struct {
	EntityUrn   string     `json:"entityUrn"`
	Name        string     `json:"name"`
	Position    string     `json:"position"`
	Description string     `json:"description"`
	DateRange   *DateRange `json:"dateRange"`
}

type TestScore struct {
	EntityUrn   string `json:"entityUrn"`
	Name        string `json:"name"`
	Score       string `json:"score"`
	Description string `json:"description"`
	DateOn      *Date  `json:"dateOn"`
}

// MiniProfile and Recommendation come from the recommendations endpoint, which is
// still on the legacy fs_ format — hence the different shape from Profile above.
type MiniProfile struct {
	EntityUrn        string        `json:"entityUrn"`
	FirstName        string        `json:"firstName"`
	LastName         string        `json:"lastName"`
	Occupation       string        `json:"occupation"`
	PublicIdentifier string        `json:"publicIdentifier"`
	Picture          *PictureFrame `json:"picture"`
}

type Recommendation struct {
	EntityUrn          string       `json:"entityUrn"`
	RecommendationText string       `json:"recommendationText"`
	Relationship       string       `json:"relationship"`
	Recommender        *MiniProfile `json:"recommenderEntity"`
	RecommenderUrn     string       `json:"*recommender"`
}
