package platform

import "testing"

func TestNormalizeFoldsAccentsAndSpacing(t *testing.T) {
	got := Normalize("  Puerto  Nariño, D.C. ")
	want := "puerto narino d c"
	if got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestTextScoreMatchesContainedCity(t *testing.T) {
	got := textScore("bogota", "bogota d c")
	if got < 0.9 {
		t.Fatalf("textScore() = %f, want at least 0.9", got)
	}
}
