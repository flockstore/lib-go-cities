package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/flockstore/lib-go-cities/platform"
)

type cliResponse struct {
	Matched     bool            `json:"matched"`
	Message     string          `json:"message,omitempty"`
	Reason      string          `json:"reason,omitempty"`
	City        string          `json:"city,omitempty"`
	Department  string          `json:"department,omitempty"`
	Code        string          `json:"code,omitempty"`
	Delivery    int             `json:"delivery,omitempty"`
	Extras      bool            `json:"extras,omitempty"`
	Confidence  float64         `json:"confidence,omitempty"`
	Suggestions []cliSuggestion `json:"suggestions,omitempty"`
	Threshold   float64         `json:"threshold"`
	Source      string          `json:"source"`
	Version     string          `json:"version"`
}

type cliSuggestion struct {
	City       string  `json:"city"`
	Department string  `json:"department"`
	Code       string  `json:"code"`
	Delivery   int     `json:"delivery,omitempty"`
	Extras     bool    `json:"extras,omitempty"`
	Confidence float64 `json:"confidence"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, writer io.Writer) int {
	flags := flag.NewFlagSet("cities", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	source := flags.String("source", "cities.json", "JSON source path")
	city := flags.String("city", "", "city text to match")
	department := flags.String("department", "", "optional department text")
	threshold := flags.Float64("threshold", platform.DefaultThreshold, "minimum confidence from 0 to 1")

	if err := flags.Parse(args); err != nil {
		writeJSON(writer, cliResponse{
			Matched:   false,
			Message:   err.Error(),
			Threshold: *threshold,
			Source:    *source,
			Version:   platform.Version,
		})
		return 2
	}
	if *city == "" {
		writeJSON(writer, cliResponse{
			Matched:   false,
			Message:   "city is required",
			Threshold: *threshold,
			Source:    *source,
			Version:   platform.Version,
		})
		return 2
	}

	matcher, err := platform.LoadFile(*source)
	if err != nil {
		writeJSON(writer, cliResponse{
			Matched:   false,
			Message:   err.Error(),
			Threshold: *threshold,
			Source:    *source,
			Version:   platform.Version,
		})
		return 1
	}

	match, found, err := matcher.Match(context.Background(), platform.SearchRequest{
		City:       *city,
		Department: *department,
		Threshold:  *threshold,
	})
	if err != nil {
		writeJSON(writer, cliResponse{
			Matched:   false,
			Message:   err.Error(),
			Threshold: *threshold,
			Source:    matcher.Source(),
			Version:   platform.Version,
		})
		return 1
	}
	if !found {
		writeJSON(writer, cliResponse{
			Matched:     false,
			Message:     "no coincidences",
			Reason:      string(match.Reason),
			Suggestions: suggestions(match.Suggestions),
			Threshold:   *threshold,
			Source:      matcher.Source(),
			Version:     platform.Version,
		})
		return 0
	}

	writeJSON(writer, cliResponse{
		Matched:    true,
		City:       match.City.Name,
		Department: match.City.Department,
		Code:       string(match.City.Code),
		Delivery:   match.City.Delivery,
		Extras:     match.City.Extras,
		Confidence: match.Confidence,
		Threshold:  *threshold,
		Source:     match.Source,
		Version:    platform.Version,
	})
	return 0
}

func writeJSON(writer io.Writer, response cliResponse) {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		_, _ = fmt.Fprintf(writer, `{"matched":false,"message":%q}`+"\n", err.Error())
	}
}

func suggestions(candidates []platform.MatchCandidate) []cliSuggestion {
	out := make([]cliSuggestion, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, cliSuggestion{
			City:       candidate.City.Name,
			Department: candidate.City.Department,
			Code:       string(candidate.City.Code),
			Delivery:   candidate.City.Delivery,
			Extras:     candidate.City.Extras,
			Confidence: candidate.Confidence,
		})
	}
	return out
}
