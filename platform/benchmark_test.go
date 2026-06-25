package platform

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type realCase struct {
	name       string
	city       string
	department string
	code       Code
	found      bool
	reason     RejectionReason
}

func TestRealSourceExamples(t *testing.T) {
	matcher := loadRealMatcher(t)
	cases := []realCase{
		{name: "duplicate armenia", city: "Armenia", found: false, reason: ReasonDuplicated},
		{name: "incongruent armenia", city: "Armenia", department: "Santander", found: false, reason: ReasonIncongruent},
		{name: "embedded armenia", city: "Armenia Quindio", code: "63001", found: true},
		{name: "santiago alias", city: "Santiago de Cali", code: "76001", found: true},
		{name: "embedded cali", city: "Cali valle del cauca", code: "76001", found: true},
		{name: "parenthesized department", city: "Bucaramanga (Santander)", code: "68001", found: true},
		{name: "low threshold miss", city: "zzzz-no-city", found: false, reason: ReasonLowThreshold},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			match, found, err := matcher.Match(context.Background(), SearchRequest{
				City:       test.city,
				Department: test.department,
				Threshold:  0.8,
			})
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if found != test.found {
				t.Fatalf("Match() found = %t, want %t, reason = %q", found, test.found, match.Reason)
			}
			if test.reason != "" && match.Reason != test.reason {
				t.Fatalf("Match() reason = %q, want %q", match.Reason, test.reason)
			}
			if test.code != "" && match.City.Code != test.code {
				t.Fatalf("Match() code = %s, want %s", match.City.Code, test.code)
			}
		})
	}
}

func TestRealSourceRandomMutations(t *testing.T) {
	matcher := loadRealMatcher(t)
	random := rand.New(rand.NewSource(42))
	cities := matcher.Cities()

	for i := 0; i < 200; i++ {
		city := cities[random.Intn(len(cities))]
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assertRealMatch(t, matcher, city, city.Name, city.Department)
			assertRealMatch(t, matcher, city, city.Name+" "+city.Department, "")
			assertRealMatch(t, matcher, city, city.Name+" ("+city.Department+")", "")
		})
	}
}

func BenchmarkLoadRealSource(b *testing.B) {
	source := realSourcePath(b)
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := LoadFile(source); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMatchRealExactWithDepartment(b *testing.B) {
	benchmarkRealMatches(b, func(city City) SearchRequest {
		return SearchRequest{City: city.Name, Department: city.Department, Threshold: 0.8}
	})
}

func BenchmarkMatchRealEmbeddedDepartment(b *testing.B) {
	benchmarkRealMatches(b, func(city City) SearchRequest {
		return SearchRequest{City: city.Name + " " + city.Department, Threshold: 0.8}
	})
}

func BenchmarkMatchRealKnownEdges(b *testing.B) {
	matcher := loadRealMatcher(b)
	requests := []SearchRequest{
		{City: "Armenia", Threshold: 0.8},
		{City: "Armenia Quindio", Threshold: 0.8},
		{City: "Santiago de Cali", Threshold: 0.8},
		{City: "Cali valle del cauca", Threshold: 0.8},
		{City: "Bucaramanga (Santander)", Threshold: 0.8},
		{City: "Armenia", Department: "Santander", Threshold: 0.8},
	}
	benchmarkRequests(b, matcher, requests)
}

func BenchmarkMatchRealMisses(b *testing.B) {
	matcher := loadRealMatcher(b)
	requests := []SearchRequest{
		{City: "zzzz-no-city", Threshold: 0.8},
		{City: "not a colombian place", Threshold: 0.8},
		{City: "california usa", Threshold: 0.8},
	}
	benchmarkRequests(b, matcher, requests)
}

func BenchmarkMatchRealParallel(b *testing.B) {
	matcher := loadRealMatcher(b)
	requests := realRequests(matcher.Cities(), 512, func(city City) SearchRequest {
		return SearchRequest{City: city.Name, Department: city.Department, Threshold: 0.8}
	})
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			request := requests[i%len(requests)]
			if _, _, err := matcher.Match(context.Background(), request); err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

func assertRealMatch(t *testing.T, matcher *Matcher, city City, text string, department string) {
	t.Helper()
	match, found, err := matcher.Match(context.Background(), SearchRequest{
		City:       text,
		Department: department,
		Threshold:  0.8,
	})
	if err != nil {
		t.Fatalf("Match() error = %v", err)
	}
	if !found {
		t.Fatalf("Match(%q, %q) found = false, reason = %q", text, department, match.Reason)
	}
	if match.City.Code != city.Code {
		t.Fatalf("Match(%q, %q) code = %s, want %s", text, department, match.City.Code, city.Code)
	}
}

func benchmarkRealMatches(b *testing.B, build func(City) SearchRequest) {
	matcher := loadRealMatcher(b)
	requests := realRequests(matcher.Cities(), 512, build)
	benchmarkRequests(b, matcher, requests)
}

func benchmarkRequests(b *testing.B, matcher *Matcher, requests []SearchRequest) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		request := requests[i%len(requests)]
		if _, _, err := matcher.Match(context.Background(), request); err != nil {
			b.Fatal(err)
		}
	}
}

func realRequests(cities []City, count int, build func(City) SearchRequest) []SearchRequest {
	random := rand.New(rand.NewSource(99))
	requests := make([]SearchRequest, 0, count)
	for len(requests) < count {
		requests = append(requests, build(cities[random.Intn(len(cities))]))
	}
	return requests
}

func loadRealMatcher(tb testing.TB) *Matcher {
	tb.Helper()
	matcher, err := LoadFile(realSourcePath(tb))
	if err != nil {
		tb.Fatal(err)
	}
	return matcher
}

func realSourcePath(tb testing.TB) string {
	tb.Helper()
	source := filepath.Join("..", "cities.json")
	if _, err := os.Stat(source); err == nil {
		return source
	} else if errors.Is(err, os.ErrNotExist) {
		tb.Skip("cities.json is not available")
	} else {
		tb.Fatal(err)
	}
	return source
}
