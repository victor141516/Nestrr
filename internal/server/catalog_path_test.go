package server

import "testing"

func TestCatalogPathAllowsMovieAndShowKeywords(t *testing.T) {
	for _, path := range []string{"movie/550/keywords", "tv/1399/keywords"} {
		if !catalogPath.MatchString(path) {
			t.Fatalf("expected catalog path %q to be allowed", path)
		}
	}
}
