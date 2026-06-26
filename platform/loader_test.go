package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReaderBuildsMatcher(t *testing.T) {
	source := `[
		{"code":11001,"name":"Bogota, D.C.","normalized":"Bogota DC","department":"CUNDINAMARCA","delivery":1,"extras":true},
		{"code":"05120","name":"Caceres","normalized":"Caceres","department":"ANTIOQUIA","delivery":4,"extras":true}
	]`

	matcher, err := LoadReader("test-source", strings.NewReader(source))
	if err != nil {
		t.Fatalf("LoadReader() error = %v", err)
	}

	if matcher.Source() != "test-source" {
		t.Fatalf("Source() = %q", matcher.Source())
	}
	cities := matcher.Cities()
	if got := len(cities); got != 2 {
		t.Fatalf("len(Cities()) = %d", got)
	}
	if cities[0].Code != "11001" || cities[1].Code != "05120" {
		t.Fatalf("Codes = %q, %q", cities[0].Code, cities[1].Code)
	}
}

func TestLoadReaderRejectsEmptySource(t *testing.T) {
	_, err := LoadReader("empty", strings.NewReader(`[]`))
	if err == nil {
		t.Fatal("LoadReader() error = nil")
	}
}

func TestLoadFileReadsAndClosesSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cities.json")
	data := `[{"code":11001,"name":"Bogota","department":"CUNDINAMARCA","delivery":1}]`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	matcher, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if matcher.Source() != path {
		t.Fatalf("Source() = %q", matcher.Source())
	}
	if err := os.Rename(path, path+".moved"); err != nil {
		t.Fatalf("source file still appears held open: %v", err)
	}
}

func TestLoadFileRejectsMissingSource(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("LoadFile() error = nil")
	}
}

func TestLoadBytesRejectsBadJSON(t *testing.T) {
	_, err := LoadBytes("bad", []byte(`{"bad":`))
	if err == nil {
		t.Fatal("LoadBytes() error = nil")
	}
}

func TestCodeRejectsInvalidJSON(t *testing.T) {
	var code Code
	if err := json.Unmarshal([]byte(`true`), &code); err == nil {
		t.Fatal("Code.UnmarshalJSON() error = nil")
	}
}
