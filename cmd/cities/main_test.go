package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunMatchesCity(t *testing.T) {
	source := writeSource(t)
	var out bytes.Buffer

	code := run([]string{"-source", source, "-city", "Cali", "-department", "Valle del Cauca"}, &out)
	if code != 0 {
		t.Fatalf("run() code = %d, out = %s", code, out.String())
	}

	response := decodeResponse(t, out.Bytes())
	if !response.Matched {
		t.Fatalf("Matched = false, message = %q", response.Message)
	}
	if response.Code != "76001" || response.Version != "1.0.1" {
		t.Fatalf("response = %+v", response)
	}
}

func TestRunRejectsDuplicate(t *testing.T) {
	source := writeSource(t)
	var out bytes.Buffer

	code := run([]string{"-source", source, "-city", "Armenia"}, &out)
	if code != 0 {
		t.Fatalf("run() code = %d", code)
	}

	response := decodeResponse(t, out.Bytes())
	if response.Matched || response.Reason != "DUPLICATED" {
		t.Fatalf("response = %+v", response)
	}
	if len(response.Suggestions) != 2 {
		t.Fatalf("len(Suggestions) = %d", len(response.Suggestions))
	}
}

func TestRunHandlesInputErrors(t *testing.T) {
	source := writeSource(t)
	tests := [][]string{
		{},
		{"-source", source, "-threshold", "2", "-city", "Cali"},
		{"-bad-flag"},
		{"-source", filepath.Join(t.TempDir(), "missing.json"), "-city", "Cali"},
	}

	for _, args := range tests {
		var out bytes.Buffer
		if code := run(args, &out); code == 0 {
			t.Fatalf("run(%v) code = 0, out = %s", args, out.String())
		}
		if !json.Valid(out.Bytes()) {
			t.Fatalf("run(%v) produced invalid JSON: %s", args, out.String())
		}
	}
}

func TestSuggestionsMapsCandidates(t *testing.T) {
	got := suggestions(nil)
	if len(got) != 0 {
		t.Fatalf("len(suggestions(nil)) = %d", len(got))
	}
}

func writeSource(t *testing.T) string {
	t.Helper()
	source := `[
		{"code":"76001","name":"Cali","normalized":"Cali","department":"VALLE DEL CAUCA","delivery":1,"extras":true},
		{"code":"05059","name":"Armenia","normalized":"Armenia","department":"ANTIOQUIA","delivery":6},
		{"code":"63001","name":"Armenia","normalized":"Armenia","department":"QUINDIO","delivery":1}
	]`
	path := filepath.Join(t.TempDir(), "cities.json")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeResponse(t *testing.T, data []byte) cliResponse {
	t.Helper()
	var response cliResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("decode response: %v; data = %s", err, string(data))
	}
	return response
}
