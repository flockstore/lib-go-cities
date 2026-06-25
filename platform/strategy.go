package platform

import (
	"math"
	"sort"
	"strings"
)

type candidateScore struct {
	item  indexedCity
	score float64
}

type searchText struct {
	cityTerms      []searchTerm
	department     string
	departmentTerm searchTerm
}

type searchTerm struct {
	text   string
	tokens []string
	length int
}

func (m *Matcher) searchText(query string, department string) searchText {
	if department == "" {
		if city, found := m.embeddedCityDepartment(query); found {
			return searchText{
				cityTerms:      uniqueTerms(query, city.name),
				department:     city.department,
				departmentTerm: newSearchTerm(city.department),
			}
		}
	}
	if department == "" {
		department = m.embeddedDepartment(query)
	}

	cityOnly := query
	if department != "" {
		cityOnly = Normalize(strings.ReplaceAll(" "+query+" ", " "+department+" ", " "))
	}
	terms := []string{query}
	if cityOnly != "" && cityOnly != query {
		terms = append(terms, cityOnly)
	}
	if strings.HasPrefix(cityOnly, "santiago de ") {
		terms = append(terms, strings.TrimSpace(strings.TrimPrefix(cityOnly, "santiago de ")))
	}

	return searchText{
		cityTerms:      uniqueTerms(terms...),
		department:     department,
		departmentTerm: newSearchTerm(department),
	}
}

func (m *Matcher) embeddedCityDepartment(query string) (indexedCity, bool) {
	item, ok := m.byPair[query]
	if ok {
		return item, true
	}
	return indexedCity{}, false
}

func (m *Matcher) embeddedDepartment(query string) string {
	wrapped := " " + query + " "
	for _, department := range m.departments {
		if strings.Contains(wrapped, " "+department+" ") {
			return department
		}
	}
	return ""
}

func (m *Matcher) resolveExactMatches(terms searchText, threshold float64) (MatchResult, bool, bool) {
	var matches []candidateScore
	for _, term := range terms.cityTerms {
		for _, item := range m.byName[term.text] {
			matches = append(matches, candidateScore{item: item, score: exactScore(terms, item)})
		}
	}
	if len(matches) == 0 {
		return MatchResult{}, false, false
	}
	sort.Slice(matches, func(i int, j int) bool {
		return matches[i].score > matches[j].score
	})

	if terms.department == "" {
		if len(matches) > 1 {
			return rejected(m.source, ReasonDuplicated, matches), false, true
		}
		if matches[0].score >= threshold {
			return accepted(m.source, matches[0]), true, true
		}
		return MatchResult{}, false, false
	}

	filtered := departmentMatches(matches, terms.department)
	if len(filtered) == 1 {
		return accepted(m.source, filtered[0]), true, true
	}
	if len(filtered) > 1 {
		return rejected(m.source, ReasonDuplicated, filtered), false, true
	}
	return rejected(m.source, ReasonIncongruent, matches), false, true
}

func scoreCity(terms searchText, item indexedCity) float64 {
	nameScore := 0.0
	for _, term := range terms.cityTerms {
		nameScore = math.Max(nameScore, textScoreTerm(term, item))
	}
	if terms.department == "" {
		return nameScore
	}

	departmentScore := textScoreDepartment(terms.departmentTerm, item)
	score := nameScore*0.82 + departmentScore*0.18
	if departmentScore < 0.55 {
		score *= 0.9
	}
	return score
}

func exactScore(terms searchText, item indexedCity) float64 {
	if terms.department == "" {
		return 1
	}
	return scoreCity(terms, item)
}

func accepted(source string, candidate candidateScore) MatchResult {
	return MatchResult{
		City:       candidate.item.city,
		Confidence: roundScore(candidate.score),
		Source:     source,
	}
}

func rejected(source string, reason RejectionReason, scores []candidateScore) MatchResult {
	return MatchResult{
		Source:      source,
		Reason:      reason,
		Suggestions: suggestions(scores, 5),
	}
}

func departmentMatches(scores []candidateScore, department string) []candidateScore {
	exact := make([]candidateScore, 0, len(scores))
	for _, score := range scores {
		if department == score.item.department {
			exact = append(exact, score)
		}
	}
	if len(exact) > 0 {
		return exact
	}

	matches := make([]candidateScore, 0, len(scores))
	for _, score := range scores {
		if textScore(department, score.item.department) >= 0.85 {
			matches = append(matches, score)
		}
	}
	return matches
}

func closeCandidates(scores []candidateScore, best float64) []candidateScore {
	tied := make([]candidateScore, 0, 3)
	for _, score := range scores {
		if score.score < best-0.02 {
			break
		}
		tied = append(tied, score)
	}
	return tied
}

func ambiguousReason(scores []candidateScore) RejectionReason {
	first := scores[0].item.name
	for _, score := range scores[1:] {
		if score.item.name != first {
			return ReasonAmbiguous
		}
	}
	return ReasonDuplicated
}

func suggestions(scores []candidateScore, limit int) []MatchCandidate {
	count := min(len(scores), limit)
	out := make([]MatchCandidate, 0, count)
	for i := 0; i < count; i++ {
		if scores[i].score <= 0 {
			continue
		}
		out = append(out, MatchCandidate{
			City:       scores[i].item.city,
			Confidence: roundScore(scores[i].score),
		})
	}
	return out
}

func uniqueTerms(values ...string) []searchTerm {
	out := make([]searchTerm, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if hasTerm(out, value) {
			continue
		}
		out = append(out, newSearchTerm(value))
	}
	return out
}

func newSearchTerm(value string) searchTerm {
	return searchTerm{text: value, tokens: strings.Fields(value), length: len(value)}
}

func hasTerm(terms []searchTerm, value string) bool {
	for _, term := range terms {
		if term.text == value {
			return true
		}
	}
	return false
}
