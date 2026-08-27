// Package models holds the domain types services and repositories exchange.
package models

import "time"

// Profile is the whole scraped result. Nothing is invented: a field the source
// omitted stays nil or empty, which the response layer renders as null or [].
type Profile struct {
	PublicID  string
	Identity  Identity
	Headline  string
	Location  string
	Industry  string
	About     string
	Images    Images
	OpenTo    []string
	Contact   *ContactInfo
	FetchedAt time.Time

	Experience    []Experience
	Education     []Education
	Skills        []Skill
	Featured      []Featured
	Certification []Certification
	Projects      []Project
	Courses       []Course
	Recommends    []Recommendation
	Volunteering  []Volunteering
	Publications  []Publication
	Patents       []Patent
	Honors        []Honor
	TestScores    []TestScore
	Languages     []Language
	Organizations []Organization
	Services      []string
	CareerBreaks  []CareerBreak
	Causes        []string
}

// Identity is who the profile belongs to.
type Identity struct {
	Name      string
	FirstName string
	LastName  string
	Pronouns  string
}

// Images holds the profile photos; nil means LinkedIn gave none, never a placeholder.
type Images struct {
	ProfilePhoto    *string
	BackgroundPhoto *string
}

// ContactInfo is populated only when the profile shares it with our account.
type ContactInfo struct {
	Email        string
	PhoneNumbers []Phone
	Websites     []Website
	Twitter      []string
	Address      string
	Birthday     string
}

// IsEmpty reports nothing usable, so the response can carry null.
func (c *ContactInfo) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.Email == "" && c.Address == "" && c.Birthday == "" &&
		len(c.PhoneNumbers) == 0 && len(c.Websites) == 0 && len(c.Twitter) == 0
}

type Phone struct {
	Number string
	Type   string
}

type Website struct {
	URL      string
	Category string
}

// DateRange is a start and optional end; Present marks an ongoing entry.
type DateRange struct {
	Start   *Date
	End     *Date
	Present bool
}

// Date is partial: LinkedIn routinely gives a year with no month.
type Date struct {
	Day   int
	Month int
	Year  int
}

// Media is an attachment on a section entry.
type Media struct {
	Title        string
	Description  string
	URL          string
	ThumbnailURL string
	Type         string
}

type Experience struct {
	Title          string
	Company        string
	CompanyURL     string
	CompanyLogo    *string
	EmploymentType string
	Location       string
	Description    string
	Period         DateRange
	Media          []Media
}

type Education struct {
	School       string
	SchoolLogo   *string
	Degree       string
	FieldOfStudy string
	Grade        string
	Activities   string
	Description  string
	Period       DateRange
	Media        []Media
}

type Skill struct {
	Name         string
	Endorsements int
}

type Featured struct {
	Title       string
	Subtitle    string
	Description string
	URL         string
	Media       []Media
}

type Certification struct {
	Name          string
	Authority     string
	LicenseNumber string
	URL           string
	AuthorityLogo *string
	Period        DateRange
	Media         []Media
}

type Project struct {
	Title       string
	Description string
	URL         string
	Period      DateRange
	Members     []string
	Media       []Media
}

type Course struct {
	Name   string
	Number string
}

type Recommendation struct {
	Text                string
	RecommenderName     string
	RecommenderHeadline string
	RecommenderPhoto    *string
	Relationship        string
}

type Volunteering struct {
	Role         string
	Organization string
	Cause        string
	Description  string
	Period       DateRange
	Media        []Media
}

type Publication struct {
	Name        string
	Publisher   string
	Description string
	URL         string
	Date        *Date
	Authors     []string
	Media       []Media
}

type Patent struct {
	Title       string
	Issuer      string
	Number      string
	Description string
	URL         string
	Pending     bool
	Date        *Date
	Inventors   []string
	Media       []Media
}

type Honor struct {
	Title       string
	Issuer      string
	Description string
	Date        *Date
	Media       []Media
}

type TestScore struct {
	Name        string
	Score       string
	Description string
	Date        *Date
}

type Language struct {
	Name        string
	Proficiency string
}

type Organization struct {
	Name        string
	Role        string
	Description string
	Period      DateRange
}

type CareerBreak struct {
	Reason      string
	Description string
	Location    string
	Period      DateRange
}
