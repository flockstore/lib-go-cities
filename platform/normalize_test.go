package platform

import "testing"

func TestNormalizeFoldsAccentsAndSpacing(t *testing.T) {
	got := Normalize("  ÁÀÂÄÃ ÉÈÊË ÍÌÎÏ ÓÒÔÖÕ ÚÙÛÜ Ñ Puerto  Nariño, D.C. ")
	want := "aaaaa eeee iiii ooooo uuuu n puerto narino d c"
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

func TestTextScoreHandlesEmptyAndFuzzyValues(t *testing.T) {
	if got := textScore("", "bogota"); got != 0 {
		t.Fatalf("empty textScore() = %f", got)
	}
	if got := textScore("bogta", "bogota"); got <= 0.55 {
		t.Fatalf("fuzzy textScore() = %f", got)
	}
}

func TestTextHelpersHandleEdgeCases(t *testing.T) {
	if got := tokenOverlap(nil, []string{"bogota"}); got != 0 {
		t.Fatalf("tokenOverlap(nil) = %f", got)
	}
	if got := distance("", "cali"); got != 4 {
		t.Fatalf("distance(empty, cali) = %d", got)
	}
	if got := distance("cali", ""); got != 4 {
		t.Fatalf("distance(cali, empty) = %d", got)
	}
}
