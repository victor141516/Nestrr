package server

import "testing"

func TestCatalogPathAllowsSupportedMovieAndShowMetadata(t *testing.T) {
	for _, path := range []string{
		"movie/550/keywords",
		"tv/1399/keywords",
		"movie/550/credits",
		"tv/1399/credits",
	} {
		if !catalogPath.MatchString(path) {
			t.Fatalf("expected catalog path %q to be allowed", path)
		}
	}
}
