package platform

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Matcher holds precomputed city data for fast repeated lookups.
type Matcher struct {
	source      string
	cities      []City
	index       []indexedCity
	byName      map[string][]indexedCity
	byPair      map[string]indexedCity
	departments []string
}

type indexedCity struct {
	city             City
	name             string
	department       string
	tokens           []string
	departmentTokens []string
	length           int
	departmentLength int
}

type topCandidates struct {
	values [5]candidateScore
	count  int
}

// NewMatcher creates a matcher from already loaded city records.
func NewMatcher(source string, records []City) *Matcher {
	cities := append([]City(nil), records...)
	index := make([]indexedCity, 0, len(cities))
	byName := make(map[string][]indexedCity, len(cities))
	byPair := make(map[string]indexedCity, len(cities)*2)
	departmentSet := make(map[string]struct{}, 32)

	for _, city := range cities {
		name := city.Normalized
		if strings.TrimSpace(name) == "" {
			name = city.Name
		}
		item := indexedCity{
			city:       city,
			name:       Normalize(name),
			department: Normalize(city.Department),
		}
		item.tokens = strings.Fields(item.name)
		item.departmentTokens = strings.Fields(item.department)
		item.length = len(item.name)
		item.departmentLength = len(item.department)
		index = append(index, item)
		byName[item.name] = append(byName[item.name], item)
		byPair[item.name+" "+item.department] = item
		byPair[item.department+" "+item.name] = item
		departmentSet[item.department] = struct{}{}
	}

	departments := make([]string, 0, len(departmentSet))
	for department := range departmentSet {
		if department != "" {
			departments = append(departments, department)
		}
	}
	sort.Slice(departments, func(i int, j int) bool {
		return len(departments[i]) > len(departments[j])
	})

	return &Matcher{
		source:      source,
		cities:      cities,
		index:       index,
		byName:      byName,
		byPair:      byPair,
		departments: departments,
	}
}

// Source returns the JSON source identifier used to build the matcher.
func (m *Matcher) Source() string {
	return m.source
}

// Cities returns a copy of the loaded city records.
func (m *Matcher) Cities() []City {
	return append([]City(nil), m.cities...)
}

// Match returns the best accepted city match for the request.
func (m *Matcher) Match(ctx context.Context, request SearchRequest) (MatchResult, bool, error) {
	query := Normalize(request.City)
	if query == "" {
		return MatchResult{}, false, fmt.Errorf("city is required")
	}
	threshold, err := threshold(request.Threshold)
	if err != nil {
		return MatchResult{}, false, err
	}

	terms := m.searchText(query, Normalize(request.Department))
	if result, found, handled := m.resolveExactMatches(terms, threshold); handled {
		return result, found, nil
	}

	var top topCandidates

	for i, item := range m.index {
		if i%128 == 0 {
			if err := ctx.Err(); err != nil {
				return MatchResult{}, false, err
			}
		}

		top.add(candidateScore{item: item, score: scoreCity(terms, item)})
	}

	scores := top.list()
	if len(scores) == 0 || scores[0].score < threshold {
		return MatchResult{
			Source:      m.source,
			Reason:      ReasonLowThreshold,
			Suggestions: suggestions(scores, 3),
		}, false, nil
	}

	best := MatchResult{
		City:       scores[0].item.city,
		Confidence: roundScore(scores[0].score),
		Source:     m.source,
	}
	if tied := closeCandidates(scores, scores[0].score); len(tied) > 1 {
		best.Reason = ambiguousReason(tied)
		best.Suggestions = suggestions(tied, 5)
		return best, false, nil
	}

	return best, true, nil
}

// MatchAsync runs Match in a goroutine and returns a one-value result channel.
func (m *Matcher) MatchAsync(ctx context.Context, request SearchRequest) <-chan AsyncResult {
	out := make(chan AsyncResult, 1)
	go func() {
		match, found, err := m.Match(ctx, request)
		out <- AsyncResult{Match: match, Found: found, Err: err}
		close(out)
	}()
	return out
}

func (t *topCandidates) add(candidate candidateScore) {
	if candidate.score <= 0 {
		return
	}
	if t.count == len(t.values) && candidate.score <= t.values[t.count-1].score {
		return
	}

	pos := t.count
	if pos == len(t.values) {
		pos--
	} else {
		t.count++
	}
	for pos > 0 && candidate.score > t.values[pos-1].score {
		t.values[pos] = t.values[pos-1]
		pos--
	}
	t.values[pos] = candidate
}

func (t *topCandidates) list() []candidateScore {
	return t.values[:t.count]
}

func threshold(value float64) (float64, error) {
	if value < 0 || value > 1 {
		return 0, fmt.Errorf("threshold must be between 0 and 1")
	}
	return value, nil
}
