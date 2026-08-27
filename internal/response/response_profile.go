package response

// These DTOs are the public contract, kept separate from the models so a new field
// cannot reach a client by accident. Pointers are null when absent, slices are [].

// ProfileResult is the payload of POST /v1/profile.
type ProfileResult struct {
	Profile ProfileCore `json:"profile"`
	// Sections holds each enabled top-level section; a disabled one is absent.
	Sections map[string]any `json:"-"`
	// SectionOrder fixes serialisation order so output is stable.
	SectionOrder []string    `json:"-"`
	Meta         ProfileMeta `json:"meta"`
}

// ProfileCore is the profile-level object; about, contactInfo and openTo appear
// only when their sections are enabled.
type ProfileCore struct {
	Identity    Identity     `json:"identity"`
	Headline    *string      `json:"headline"`
	Location    *string      `json:"location"`
	Industry    *string      `json:"industry"`
	About       *string      `json:"about,omitempty"`
	ContactInfo *ContactInfo `json:"contactInfo,omitempty"`
	OpenTo      []string     `json:"openTo,omitempty"`
	Images      Images       `json:"images"`
}

// Identity carries the member's name. Pronouns are null unless stated.
type Identity struct {
	Name      string  `json:"name"`
	FirstName string  `json:"firstName"`
	LastName  string  `json:"lastName"`
	Pronouns  *string `json:"pronouns"`
}

// Images are null when LinkedIn gave us nothing — never a placeholder URL.
type Images struct {
	ProfilePhoto    *string `json:"profilePhoto"`
	BackgroundPhoto *string `json:"backgroundPhoto"`
}

type ContactInfo struct {
	Email        *string   `json:"email"`
	PhoneNumbers []Phone   `json:"phoneNumbers"`
	Websites     []Website `json:"websites"`
	Twitter      []string  `json:"twitter"`
	Address      *string   `json:"address"`
	Birthday     *string   `json:"birthday"`
}

type Phone struct {
	Number string `json:"number"`
	Type   string `json:"type,omitempty"`
}

type Website struct {
	URL      string `json:"url"`
	Category string `json:"category,omitempty"`
}

// ProfileMeta describes the fetch, not the person.
type ProfileMeta struct {
	PublicID   string   `json:"publicId"`
	ProfileURL string   `json:"profileUrl"`
	Cached     bool     `json:"cached"`
	FetchedAt  string   `json:"fetchedAt"`
	Sections   []string `json:"sectionsReturned"`
	// Unavailable names enabled sections LinkedIn had no data for.
	Unavailable []string `json:"sectionsUnavailable"`
}

// DateRange renders a period; Present marks an ongoing entry, so no end date is invented.
type DateRange struct {
	Start   *string `json:"start"`
	End     *string `json:"end"`
	Present bool    `json:"present"`
}

type Media struct {
	Title        string `json:"title,omitempty"`
	Description  string `json:"description,omitempty"`
	URL          string `json:"url,omitempty"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Type         string `json:"type,omitempty"`
}

type Experience struct {
	Title          string    `json:"title"`
	Company        string    `json:"company"`
	CompanyLogo    *string   `json:"companyLogo"`
	EmploymentType string    `json:"employmentType,omitempty"`
	Location       string    `json:"location,omitempty"`
	Description    string    `json:"description,omitempty"`
	Period         DateRange `json:"period"`
	Media          []Media   `json:"media"`
}

type Education struct {
	School       string    `json:"school"`
	SchoolLogo   *string   `json:"schoolLogo"`
	Degree       string    `json:"degree,omitempty"`
	FieldOfStudy string    `json:"fieldOfStudy,omitempty"`
	Grade        string    `json:"grade,omitempty"`
	Activities   string    `json:"activities,omitempty"`
	Description  string    `json:"description,omitempty"`
	Period       DateRange `json:"period"`
	Media        []Media   `json:"media"`
}

type Skill struct {
	Name         string `json:"name"`
	Endorsements int    `json:"endorsements"`
}

type Featured struct {
	Title       string  `json:"title"`
	Subtitle    string  `json:"subtitle,omitempty"`
	Description string  `json:"description,omitempty"`
	URL         string  `json:"url,omitempty"`
	Media       []Media `json:"media"`
}

type Certification struct {
	Name          string    `json:"name"`
	Authority     string    `json:"authority,omitempty"`
	AuthorityLogo *string   `json:"authorityLogo"`
	LicenseNumber string    `json:"licenseNumber,omitempty"`
	URL           string    `json:"url,omitempty"`
	Period        DateRange `json:"period"`
	Media         []Media   `json:"media"`
}

type Project struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	URL         string    `json:"url,omitempty"`
	Period      DateRange `json:"period"`
	Members     []string  `json:"members"`
	Media       []Media   `json:"media"`
}

type Course struct {
	Name   string `json:"name"`
	Number string `json:"number,omitempty"`
}

type Recommendation struct {
	Text                string  `json:"text"`
	RecommenderName     string  `json:"recommenderName"`
	RecommenderHeadline string  `json:"recommenderHeadline,omitempty"`
	RecommenderPhoto    *string `json:"recommenderPhoto"`
	Relationship        string  `json:"relationship,omitempty"`
}

type Volunteering struct {
	Role         string    `json:"role"`
	Organization string    `json:"organization,omitempty"`
	Cause        string    `json:"cause,omitempty"`
	Description  string    `json:"description,omitempty"`
	Period       DateRange `json:"period"`
	Media        []Media   `json:"media"`
}

type Publication struct {
	Name        string   `json:"name"`
	Publisher   string   `json:"publisher,omitempty"`
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url,omitempty"`
	Date        *string  `json:"date"`
	Authors     []string `json:"authors"`
	Media       []Media  `json:"media"`
}

type Patent struct {
	Title       string   `json:"title"`
	Issuer      string   `json:"issuer,omitempty"`
	Number      string   `json:"number,omitempty"`
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url,omitempty"`
	Pending     bool     `json:"pending"`
	Date        *string  `json:"date"`
	Inventors   []string `json:"inventors"`
	Media       []Media  `json:"media"`
}

type Honor struct {
	Title       string  `json:"title"`
	Issuer      string  `json:"issuer,omitempty"`
	Description string  `json:"description,omitempty"`
	Date        *string `json:"date"`
	Media       []Media `json:"media"`
}

type TestScore struct {
	Name        string  `json:"name"`
	Score       string  `json:"score,omitempty"`
	Description string  `json:"description,omitempty"`
	Date        *string `json:"date"`
}

type Language struct {
	Name        string `json:"name"`
	Proficiency string `json:"proficiency,omitempty"`
}

type Organization struct {
	Name        string    `json:"name"`
	Role        string    `json:"role,omitempty"`
	Description string    `json:"description,omitempty"`
	Period      DateRange `json:"period"`
}

type CareerBreak struct {
	Reason      string    `json:"reason"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	Period      DateRange `json:"period"`
}
