package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeTitleMatchesChiprrContract(t *testing.T) {
	cases := map[string]string{
		`Star Wars: A/B <Test> "Cut"?`: "Star Wars - A - B Test Cut",
		"  Movie...   ":                "Movie",
		"CON":                          "CON_",
		"El laberinto del fauno":       "El laberinto del fauno",
	}
	for in, want := range cases {
		got, err := SanitizeTitle(in)
		if err != nil || got != want {
			t.Fatalf("SanitizeTitle(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestResolverSelectsAlphabeticallyAndEnumerates(t *testing.T) {
	root := t.TempDir()
	movies := filepath.Join(root, "Movies")
	shows := filepath.Join(root, "Shows")
	os.MkdirAll(filepath.Join(shows, "Breaking Bad", "Season 1"), 0o755)
	os.MkdirAll(movies, 0o755)
	for _, p := range []string{filepath.Join(movies, "Arrival (2016).mkv"), filepath.Join(movies, "Arrival (2016).mp4"), filepath.Join(shows, "Breaking Bad", "Season 1", "Breaking Bad S01E01.mkv")} {
		if err := os.WriteFile(p, []byte("mock"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := Resolver{MoviesRoot: movies, ShowsRoot: shows}
	path, ok, err := r.Movie("Arrival", 2016)
	if err != nil || !ok || filepath.Ext(path) != ".mkv" {
		t.Fatalf("deterministic movie resolution failed: %s %v %v", path, ok, err)
	}
	showPath, showOK, err := r.Show("Breaking Bad")
	if err != nil || !showOK || showPath != filepath.Join(shows, "Breaking Bad") {
		t.Fatalf("show availability failed: %s %v %v", showPath, showOK, err)
	}
	entries, err := r.Enumerate()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries", len(entries))
	}
}
