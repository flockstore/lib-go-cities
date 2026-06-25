package platform

import (
	"math"
	"strings"
	"unicode"
)

// Normalize returns comparable lowercase ASCII-ish text for city matching.
func Normalize(value string) string {
	var builder strings.Builder
	lastSpace := true

	for _, char := range value {
		char = foldAccent(unicode.ToLower(char))
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}

	return strings.TrimSpace(builder.String())
}

func foldAccent(char rune) rune {
	switch char {
	case 'á', 'à', 'â', 'ä', 'ã':
		return 'a'
	case 'é', 'è', 'ê', 'ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï':
		return 'i'
	case 'ó', 'ò', 'ô', 'ö', 'õ':
		return 'o'
	case 'ú', 'ù', 'û', 'ü':
		return 'u'
	case 'ñ':
		return 'n'
	default:
		return char
	}
}

func textScore(query string, candidate string) float64 {
	return textScoreTerm(searchTerm{
		text:   query,
		tokens: strings.Fields(query),
		length: len(query),
	}, indexedCity{
		name:   candidate,
		tokens: strings.Fields(candidate),
		length: len(candidate),
	})
}

func textScoreTerm(query searchTerm, candidate indexedCity) float64 {
	return textScoreParts(query, candidate.name, candidate.tokens, candidate.length)
}

func textScoreDepartment(query searchTerm, candidate indexedCity) float64 {
	return textScoreParts(query, candidate.department, candidate.departmentTokens, candidate.departmentLength)
}

func textScoreParts(query searchTerm, candidate string, tokens []string, length int) float64 {
	if query.text == candidate {
		return 1
	}
	if query.text == "" || candidate == "" {
		return 0
	}
	if strings.Contains(candidate, query.text) || strings.Contains(query.text, candidate) {
		return math.Max(0.9, lengthRatio(query.length, length))
	}

	editScore := 1 - float64(distance(query.text, candidate))/float64(max(query.length, length))
	tokenScore := tokenOverlap(query.tokens, tokens)
	return math.Max(0, editScore*0.72+tokenScore*0.28)
}

func lengthRatio(a int, b int) float64 {
	if a > b {
		a, b = b, a
	}
	return float64(a) / float64(b)
}

func tokenOverlap(queryTokens []string, candidateTokens []string) float64 {
	if len(queryTokens) == 0 {
		return 0
	}

	hits := 0
	for _, queryToken := range queryTokens {
		for _, candidateToken := range candidateTokens {
			if queryToken == candidateToken {
				hits++
				break
			}
		}
	}
	return float64(hits) / float64(len(queryTokens))
}

func distance(a string, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	var previousBuf [96]int
	var currentBuf [96]int
	previous := previousBuf[:]
	current := currentBuf[:]
	if len(b)+1 > len(previousBuf) {
		previous = make([]int, len(b)+1)
		current = make([]int, len(b)+1)
	} else {
		previous = previous[:len(b)+1]
		current = current[:len(b)+1]
	}

	for j := range previous {
		previous[j] = j
	}

	for i := range len(a) {
		current[0] = i + 1
		for j := range len(b) {
			cost := 1
			if a[i] == b[j] {
				cost = 0
			}
			current[j+1] = min(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous, current = current, previous
	}
	return previous[len(b)]
}

func roundScore(value float64) float64 {
	return math.Round(value*10000) / 10000
}
