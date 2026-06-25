package platform

import (
	"context"
	"testing"
)

func TestMatchUsesDepartmentToImproveResult(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "1", Name: "La Union", Normalized: "La Union", Department: "ANTIOQUIA", Delivery: 2, Extras: true},
		{Code: "2", Name: "La Union", Normalized: "La Union", Department: "NARINO", Delivery: 5, Extras: false},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:       "la unión",
		Department: "Nariño",
		Threshold:  0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !found {
		t.Fatal("Match() found = false")
	}
	if match.City.Code != "2" {
		t.Fatalf("Match() code = %s", match.City.Code)
	}
}

func TestMatchRejectsBelowThreshold(t *testing.T) {
	matcher := NewMatcher("test", []City{{Code: "1", Name: "Bogota", Department: "CUNDINAMARCA"}})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Cartagena",
		Threshold: 0.95,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if found {
		t.Fatal("Match() found = true")
	}
	if match.Reason != ReasonLowThreshold {
		t.Fatalf("Match() reason = %q", match.Reason)
	}
}

func TestMatchAsyncReturnsResult(t *testing.T) {
	matcher := NewMatcher("test", []City{{Code: "1", Name: "Cali", Department: "VALLE DEL CAUCA"}})

	result := <-matcher.MatchAsync(context.Background(), SearchRequest{City: "Cali"})
	if result.Err != nil {
		t.Fatalf("MatchAsync() error = %v", result.Err)
	}
	if !result.Found {
		t.Fatal("MatchAsync() found = false")
	}
}

func TestMatchRejectsInvalidInput(t *testing.T) {
	matcher := NewMatcher("test", []City{{Code: "1", Name: "Cali", Department: "VALLE DEL CAUCA"}})

	if _, _, err := matcher.Match(context.Background(), SearchRequest{}); err == nil {
		t.Fatal("Match() empty city error = nil")
	}
	if _, _, err := matcher.Match(context.Background(), SearchRequest{City: "Cali", Threshold: 2}); err == nil {
		t.Fatal("Match() threshold error = nil")
	}
}

func TestMatchHonorsContextCancellation(t *testing.T) {
	matcher := NewMatcher("test", []City{{Code: "1", Name: "Cali", Department: "VALLE DEL CAUCA"}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := matcher.Match(ctx, SearchRequest{City: "unknown", Threshold: 0.9})
	if err == nil {
		t.Fatal("Match() context error = nil")
	}
}

func TestCitiesReturnsCopy(t *testing.T) {
	matcher := NewMatcher("test", []City{{Code: "1", Name: "Cali", Department: "VALLE DEL CAUCA"}})
	cities := matcher.Cities()
	cities[0].Code = "changed"

	if matcher.Cities()[0].Code != "1" {
		t.Fatal("Cities() returned mutable backing data")
	}
}

func TestMatchRejectsDuplicatedCityWithoutDepartment(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "05059", Name: "Armenia", Normalized: "Armenia", Department: "ANTIOQUIA"},
		{Code: "63001", Name: "Armenia", Normalized: "Armenia", Department: "QUINDIO"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Armenia",
		Threshold: 0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if found {
		t.Fatal("Match() found = true")
	}
	if match.Reason != ReasonDuplicated {
		t.Fatalf("Match() reason = %q", match.Reason)
	}
	if len(match.Suggestions) != 2 {
		t.Fatalf("len(Suggestions) = %d", len(match.Suggestions))
	}
}

func TestMatchRejectsIncongruentDepartment(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "05059", Name: "Armenia", Normalized: "Armenia", Department: "ANTIOQUIA"},
		{Code: "63001", Name: "Armenia", Normalized: "Armenia", Department: "QUINDIO"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:       "Armenia",
		Department: "Santander",
		Threshold:  0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if found {
		t.Fatal("Match() found = true")
	}
	if match.Reason != ReasonIncongruent {
		t.Fatalf("Match() reason = %q", match.Reason)
	}
}

func TestMatchRejectsAmbiguousDifferentCities(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "1", Name: "Santa Rosa", Normalized: "Santa Rosa", Department: "BOLIVAR"},
		{Code: "2", Name: "Santa Maria", Normalized: "Santa Maria", Department: "BOYACA"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Santa",
		Threshold: 0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if found {
		t.Fatal("Match() found = true")
	}
	if match.Reason != ReasonAmbiguous {
		t.Fatalf("Match() reason = %q", match.Reason)
	}
}

func TestMatchResolvesEmbeddedDepartment(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "76001", Name: "Cali", Normalized: "Cali", Department: "VALLE DEL CAUCA"},
		{Code: "54680", Name: "Santiago", Normalized: "Santiago", Department: "NORTE DE SANTANDER"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Cali valle del cauca",
		Threshold: 0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !found {
		t.Fatalf("Match() found = false, reason = %q", match.Reason)
	}
	if match.City.Code != "76001" {
		t.Fatalf("Match() code = %s", match.City.Code)
	}
}

func TestMatchResolvesKnownLongCityAlias(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "76001", Name: "Cali", Normalized: "Cali", Department: "VALLE DEL CAUCA"},
		{Code: "54680", Name: "Santiago", Normalized: "Santiago", Department: "NORTE DE SANTANDER"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Santiago de Cali",
		Threshold: 0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !found {
		t.Fatalf("Match() found = false, reason = %q", match.Reason)
	}
	if match.City.Code != "76001" {
		t.Fatalf("Match() code = %s", match.City.Code)
	}
}

func TestMatchResolvesParenthesizedDepartment(t *testing.T) {
	matcher := NewMatcher("test", []City{
		{Code: "68001", Name: "Bucaramanga", Normalized: "Bucaramanga", Department: "SANTANDER"},
	})

	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:      "Bucaramanga (Santander)",
		Threshold: 0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !found {
		t.Fatalf("Match() found = false, reason = %q", match.Reason)
	}
	if match.City.Code != "68001" {
		t.Fatalf("Match() code = %s", match.City.Code)
	}
}
