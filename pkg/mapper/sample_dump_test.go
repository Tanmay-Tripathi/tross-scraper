package mapper

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Tanmay-Tripathi/tross-scraper/internal/config"
	"github.com/Tanmay-Tripathi/tross-scraper/internal/models"
)

// TestDumpSampleResponse writes a representative response to TROSS_SAMPLE_OUT —
// this is where the README's example JSON comes from. Skipped unless it is set.
func TestDumpSampleResponse(t *testing.T) {
	out := os.Getenv("TROSS_SAMPLE_OUT")
	if out == "" {
		t.Skip("set TROSS_SAMPLE_OUT to dump a sample response")
	}

	photo := "https://media.licdn.com/dms/image/sample/800_800/x.jpg"
	profile := &models.Profile{
		PublicID:  "priya-nair-eng",
		Identity:  models.Identity{Name: "Priya Nair", FirstName: "Priya", LastName: "Nair"},
		Headline:  "Senior Backend Engineer · Distributed Systems & Payments Infrastructure",
		Location:  "Bengaluru, Karnataka, India",
		Industry:  "Financial Services Software",
		About:     "Backend engineer focused on payment rails, event-driven systems and reliability at scale.\nI enjoy turning ambiguous problems into boring, dependable infrastructure.",
		FetchedAt: time.Now().UTC(),
		Images:    models.Images{ProfilePhoto: &photo}, // background absent on purpose
		OpenTo:    []string{"Speaking engagements", "Mentoring backend engineers"},
		Contact:   &models.ContactInfo{Email: "priya.nair@example.dev"},
		Experience: []models.Experience{
			{
				Title: "Senior Backend Engineer", Company: "Razorstack",
				EmploymentType: "Full time", Location: "Bengaluru · Hybrid",
				Description: "Own the settlement and reconciliation platform. Cut settlement lag from 40 min to under 90 s.",
				Period:      models.DateRange{Start: &models.Date{Year: 2022, Month: 4}, Present: true},
				Media:       []models.Media{},
			},
			{
				Title: "Backend Engineer", Company: "Finlytics", Location: "Remote",
				Description: "Built the fraud-scoring ingestion service handling 12k events/s at p99 under 60 ms.",
				Period:      models.DateRange{Start: &models.Date{Year: 2019}, End: &models.Date{Year: 2022, Month: 3}},
				Media:       []models.Media{},
			},
		},
		Education: []models.Education{
			{School: "BITS Pilani", Degree: "B.E. Computer Science", Period: models.DateRange{Start: &models.Date{Year: 2013}, End: &models.Date{Year: 2017}}, Media: []models.Media{}},
		},
		Skills: []models.Skill{
			{Name: "Go", Endorsements: 48}, {Name: "Distributed Systems", Endorsements: 39},
			{Name: "Kafka", Endorsements: 31}, {Name: "PostgreSQL", Endorsements: 27},
			{Name: "gRPC", Endorsements: 19}, {Name: "Kubernetes", Endorsements: 22},
		},
		Certification: []models.Certification{
			{Name: "Certified Kubernetes Administrator", Authority: "CNCF", LicenseNumber: "LF-8841-CKA", Period: models.DateRange{Start: &models.Date{Year: 2023, Month: 3}}, Media: []models.Media{}},
		},
		Recommends: []models.Recommendation{
			{Text: "Priya has an unusual talent for making complex systems feel calm.", RecommenderName: "Devang Rao", Relationship: "Engineering Manager at Razorstack"},
		},
		Languages: []models.Language{
			{Name: "English", Proficiency: "Native or bilingual"}, {Name: "Hindi", Proficiency: "Native or bilingual"},
		},
		// Deliberately empty: proves the "enabled but no data" rule emits [].
		Patents:       []models.Patent{},
		TestScores:    []models.TestScore{},
		Projects:      []models.Project{},
		Publications:  []models.Publication{},
		Honors:        []models.Honor{},
		Courses:       []models.Course{},
		Volunteering:  []models.Volunteering{},
		Organizations: []models.Organization{},
		Featured:      []models.Featured{},
		Services:      []string{},
		CareerBreaks:  []models.CareerBreak{},
		Causes:        []string{},
	}

	encoded, err := json.MarshalIndent(map[string]any{
		"code":    "00000",
		"message": "success",
		"result":  ToProfileResult(profile, config.AllEnabled(), false),
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(out, encoded, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Logf("wrote %d bytes to %s", len(encoded), out)
}
