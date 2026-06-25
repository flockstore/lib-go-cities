// Package platform validates and matches Colombian city text against a JSON source.
package platform

import "encoding/json"

// Version identifies the public package and CLI version.
const Version = "1.0.0"

// DefaultThreshold is the recommended minimum confidence for callers and the CLI.
const DefaultThreshold = 0.75

// RejectionReason explains why a lookup did not produce an accepted match.
type RejectionReason string

const (
	// ReasonLowThreshold means the best score did not meet the requested threshold.
	ReasonLowThreshold RejectionReason = "LOW_THRESHOLD"
	// ReasonDuplicated means the city name maps to multiple departments without enough evidence.
	ReasonDuplicated RejectionReason = "DUPLICATED"
	// ReasonIncongruent means the city was recognized but the department evidence conflicts.
	ReasonIncongruent RejectionReason = "INCONGRUENT"
	// ReasonAmbiguous means multiple plausible cities match the request.
	ReasonAmbiguous RejectionReason = "AMBIGUOUS"
)

// Code is a city code as written by the JSON source.
type Code string

// UnmarshalJSON accepts city codes encoded as JSON strings or numbers.
func (c *Code) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*c = Code(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	*c = Code(number.String())
	return nil
}

// City is one city record loaded from the JSON source.
type City struct {
	// Code is the carrier or geographic city code.
	Code Code `json:"code"`
	// Name is the display city name.
	Name string `json:"name"`
	// Normalized is the source-provided normalized city name.
	Normalized string `json:"normalized"`
	// Department is the display department name.
	Department string `json:"department"`
	// Delivery is the delivery classification from the source.
	Delivery int `json:"delivery"`
	// Extras reports whether the city supports extra service metadata.
	Extras bool `json:"extras"`
}

// SearchRequest describes one city lookup.
type SearchRequest struct {
	// City is the required city text to match.
	City string
	// Department is optional department text that improves match quality.
	Department string
	// Threshold is the minimum accepted confidence from 0 to 1.
	Threshold float64
}

// MatchResult is the best accepted city match.
type MatchResult struct {
	// City is the matched source city.
	City City `json:"city"`
	// Confidence is the match score from 0 to 1.
	Confidence float64 `json:"confidence"`
	// Source is the JSON source identifier used to build the matcher.
	Source string `json:"source"`
	// Reason explains why the result was rejected when no match was found.
	Reason RejectionReason `json:"reason,omitempty"`
	// Suggestions contains plausible records when the request was rejected.
	Suggestions []MatchCandidate `json:"suggestions,omitempty"`
}

// MatchCandidate is a plausible city match returned for rejected lookups.
type MatchCandidate struct {
	// City is the suggested source city.
	City City `json:"city"`
	// Confidence is the candidate score from 0 to 1.
	Confidence float64 `json:"confidence"`
}

// AsyncResult is emitted by asynchronous match calls.
type AsyncResult struct {
	// Match is the accepted match when Found is true.
	Match MatchResult
	// Found reports whether a match met the request threshold.
	Found bool
	// Err contains validation or context errors.
	Err error
}
