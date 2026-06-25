package platform

import (
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
