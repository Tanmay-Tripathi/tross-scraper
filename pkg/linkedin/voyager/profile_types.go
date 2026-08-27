package voyager

// These structs mirror LinkedIn's wire format and are intentionally permissive:
// every field is optional, and an unrecognised one must never break decoding.

// ProfileViewData is the profileView "data" object; each view lists entityUrns.
type ProfileViewData struct {
	Profile string `json:"*profile"`

	PositionView            urnList `json:"*positionView"`
	EducationView           urnList `json:"*educationView"`
	SkillView               urnList `json:"*skillView"`
	CertificationView       urnList `json:"*certificationView"`
	ProjectView             urnList `json:"*projectView"`
	PublicationView         urnList `json:"*publicationView"`
	PatentView              urnList `json:"*patentView"`
	HonorView               urnList `json:"*honorView"`
	CourseView              urnList `json:"*courseView"`
	LanguageView            urnList `json:"*languageView"`
	OrganizationView        urnList `json:"*organizationView"`
	VolunteerExperienceView urnList `json:"*volunteerExperienceView"`
	TestScoreView           urnList `json:"*testScoreView"`
}

// urnList wraps a view's element list.
type urnList struct {
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

// TimePeriod is a start and an optional end. A nil EndDate means "Present".
type TimePeriod struct {
	StartDate *Date `json:"startDate"`
	EndDate   *Date `json:"endDate"`
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

// PictureFrame is the profilePicture / backgroundImage envelope.
type PictureFrame struct {
	DisplayImageReference *imageReference `json:"displayImageReference"`
	// Older payloads put the vector image directly under the frame.
	VectorImage *VectorImage `json:"vectorImage"`
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

// Profile is the core member entity.
type Profile struct {
	EntityUrn        string        `json:"entityUrn"`
	FirstName        string        `json:"firstName"`
	LastName         string        `json:"lastName"`
	Headline         string        `json:"headline"`
	Summary          string        `json:"summary"`
	IndustryName     string        `json:"industryName"`
	LocationName     string        `json:"locationName"`
	GeoLocationName  string        `json:"geoLocationName"`
	GeoCountryName   string        `json:"geoCountryName"`
	PublicIdentifier string        `json:"publicIdentifier"`
	ProfilePicture   *PictureFrame `json:"profilePicture"`
	BackgroundImage  *PictureFrame `json:"backgroundImage"`
}

// MiniProfile is the compact member entity attached to recommendations.
type MiniProfile struct {
	EntityUrn        string        `json:"entityUrn"`
	FirstName        string        `json:"firstName"`
	LastName         string        `json:"lastName"`
	Occupation       string        `json:"occupation"`
	PublicIdentifier string        `json:"publicIdentifier"`
	Picture          *PictureFrame `json:"picture"`
}

// Company is the organisation attached to a position or certification.
type Company struct {
	EntityUrn string        `json:"entityUrn"`
	Name      string        `json:"name"`
	Logo      *PictureFrame `json:"logo"`
	Universal string        `json:"universalName"`
}

// School is the institution attached to an education entry.
type School struct {
	EntityUrn  string        `json:"entityUrn"`
	SchoolName string        `json:"schoolName"`
	Logo       *PictureFrame `json:"logo"`
}

type Position struct {
	EntityUrn       string      `json:"entityUrn"`
	Title           string      `json:"title"`
	CompanyName     string      `json:"companyName"`
	CompanyUrn      string      `json:"companyUrn"`
	Description     string      `json:"description"`
	LocationName    string      `json:"locationName"`
	GeoLocationName string      `json:"geoLocationName"`
	EmploymentType  string      `json:"employmentType"`
	TimePeriod      *TimePeriod `json:"timePeriod"`
	Company         *Company    `json:"company"`
}

type Education struct {
	EntityUrn    string      `json:"entityUrn"`
	SchoolName   string      `json:"schoolName"`
	DegreeName   string      `json:"degreeName"`
	FieldOfStudy string      `json:"fieldOfStudy"`
	Grade        string      `json:"grade"`
	Activities   string      `json:"activities"`
	Description  string      `json:"description"`
	TimePeriod   *TimePeriod `json:"timePeriod"`
	School       *School     `json:"school"`
}

type Skill struct {
	EntityUrn string `json:"entityUrn"`
	Name      string `json:"name"`
}

type Certification struct {
	EntityUrn     string      `json:"entityUrn"`
	Name          string      `json:"name"`
	Authority     string      `json:"authority"`
	LicenseNumber string      `json:"licenseNumber"`
	URL           string      `json:"url"`
	TimePeriod    *TimePeriod `json:"timePeriod"`
	Company       *Company    `json:"company"`
}

type Project struct {
	EntityUrn   string      `json:"entityUrn"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	TimePeriod  *TimePeriod `json:"timePeriod"`
	Members     []struct {
		Member *MiniProfile `json:"member"`
		Name   string       `json:"name"`
	} `json:"members"`
}

type Publication struct {
	EntityUrn   string `json:"entityUrn"`
	Name        string `json:"name"`
	Publisher   string `json:"publisher"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Date        *Date  `json:"date"`
	Authors     []struct {
		Member *MiniProfile `json:"member"`
		Name   string       `json:"name"`
	} `json:"authors"`
}

type Patent struct {
	EntityUrn   string `json:"entityUrn"`
	Title       string `json:"title"`
	Issuer      string `json:"issuer"`
	Number      string `json:"number"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Pending     bool   `json:"pending"`
	IssueDate   *Date  `json:"issueDate"`
	FilingDate  *Date  `json:"filingDate"`
	Inventors   []struct {
		Member *MiniProfile `json:"member"`
		Name   string       `json:"name"`
	} `json:"inventors"`
}

type Honor struct {
	EntityUrn   string `json:"entityUrn"`
	Title       string `json:"title"`
	Issuer      string `json:"issuer"`
	Description string `json:"description"`
	IssueDate   *Date  `json:"issueDate"`
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

type Organization struct {
	EntityUrn   string      `json:"entityUrn"`
	Name        string      `json:"name"`
	Position    string      `json:"position"`
	Description string      `json:"description"`
	TimePeriod  *TimePeriod `json:"timePeriod"`
}

type VolunteerExperience struct {
	EntityUrn   string      `json:"entityUrn"`
	Role        string      `json:"role"`
	CompanyName string      `json:"companyName"`
	Cause       string      `json:"cause"`
	Description string      `json:"description"`
	TimePeriod  *TimePeriod `json:"timePeriod"`
}

type TestScore struct {
	EntityUrn   string `json:"entityUrn"`
	Name        string `json:"name"`
	Score       string `json:"score"`
	Description string `json:"description"`
	Date        *Date  `json:"date"`
}

type Recommendation struct {
	EntityUrn          string       `json:"entityUrn"`
	RecommendationText string       `json:"recommendationText"`
	Relationship       string       `json:"relationship"`
	Recommender        *MiniProfile `json:"recommender"`
	RecommenderUrn     string       `json:"*recommender"`
}

// ContactInfo is the profileContactInfo response.
type ContactInfo struct {
	EmailAddress string `json:"emailAddress"`
	Address      string `json:"address"`
	BirthDateOn  *Date  `json:"birthDateOn"`
	PhoneNumbers []struct {
		Number string `json:"number"`
		Type   string `json:"type"`
	} `json:"phoneNumbers"`
	Websites []struct {
		URL  string `json:"url"`
		Type struct {
			Standard *struct {
				Category string `json:"category"`
			} `json:"com.linkedin.voyager.identity.profile.StandardWebsite"`
			Custom *struct {
				Label string `json:"label"`
			} `json:"com.linkedin.voyager.identity.profile.CustomWebsite"`
		} `json:"type"`
	} `json:"websites"`
	TwitterHandles []struct {
		Name string `json:"name"`
	} `json:"twitterHandles"`
}
