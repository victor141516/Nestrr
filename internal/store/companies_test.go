package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestCompanySearchAndMediaRelations(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	companies := []Company{
		{TMDBID: 1, Name: "Appian Way", OriginCountry: "US"},
		{TMDBID: 2, Name: "Sikelia Productions", OriginCountry: "US"},
	}
	if err := database.ReplaceMediaCompanies(ctx, "movie", 42, companies); err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, "movie", 42, "es-ES", []byte(`{"id":42,"title":"Película"}`)); err != nil {
		t.Fatal(err)
	}
	matches, err := database.SearchCompanies(ctx, "appian", 12)
	if err != nil || len(matches) != 1 || matches[0].TMDBID != 1 {
		t.Fatalf("unexpected company search: %#v %v", matches, err)
	}
	media, err := database.LocalMediaForCompanies(ctx, "es-ES", "movie", []int{1, 2})
	if err != nil || len(media) != 1 {
		t.Fatalf("unexpected related media: %#v %v", media, err)
	}
	var document map[string]any
	if json.Unmarshal(media[0], &document) != nil || document["title"] != "Película" {
		t.Fatalf("unexpected metadata: %s", media[0])
	}
}

func TestCompanySearchCacheKeepsOrderAndEmptyResults(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	companies := []Company{{TMDBID: 2, Name: "Second"}, {TMDBID: 1, Name: "First"}}
	if err := database.PutCompanySearch(ctx, "studio", "en-US", companies); err != nil {
		t.Fatal(err)
	}
	cached, fresh, err := database.CachedCompanySearch(ctx, " STUDIO ", "en-US", 30*24*time.Hour)
	if err != nil || !fresh || len(cached) != 2 || cached[0].TMDBID != 2 {
		t.Fatalf("cached company order was lost: %#v %v %v", cached, fresh, err)
	}
	if err := database.PutCompanySearch(ctx, "missing", "en-US", nil); err != nil {
		t.Fatal(err)
	}
	empty, fresh, err := database.CachedCompanySearch(ctx, "missing", "en-US", 30*24*time.Hour)
	if err != nil || !fresh || len(empty) != 0 {
		t.Fatalf("empty company search was not cached: %#v %v %v", empty, fresh, err)
	}
}
