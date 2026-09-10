package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSearchLocalCatalogIndexesCanonicalTranslationsAndLocalizedMetadata(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	features, _ := json.Marshal(CatalogFeatures{Genres: []int{18}})
	if err := database.PutCatalogMedia(ctx, CatalogMedia{
		CatalogKey: CatalogKey{MediaType: "movie", TMDBID: 101},
		Payload: []byte(`{
			"id":101,"title":"Inglourious Basterds","original_title":"Inglourious Basterds","popularity":40,
			"translations":{"translations":[{"iso_639_1":"es","iso_3166_1":"ES","data":{"title":"Malditos bastardos","overview":"Descripción"}}]}
		}`),
		Features:      features,
		Title:         "Inglourious Basterds",
		OriginalTitle: "Inglourious Basterds",
		Popularity:    40,
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, "movie", 202, "es-ES", []byte(`{"id":202,"title":"Malditos de nuevo"}`)); err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, "tv", 303, "es-ES", []byte(`{"id":303,"name":"Malditos: la serie"}`)); err != nil {
		t.Fatal(err)
	}

	documents, total, err := database.SearchLocalCatalog(ctx, "movie", "malditos", "es-ES", 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(documents) != 2 || documents[0].TMDBID != 101 || documents[1].TMDBID != 202 {
		t.Fatalf("unexpected localized movie results: total=%d documents=%#v", total, documents)
	}
	tv, tvTotal, err := database.SearchLocalCatalog(ctx, "tv", "la serie", "es-ES", 20)
	if err != nil || tvTotal != 1 || len(tv) != 1 || tv[0].TMDBID != 303 {
		t.Fatalf("unexpected media-type scoped result: total=%d documents=%#v err=%v", tvTotal, tv, err)
	}
}

func TestSearchLocalCatalogReplacesStaleTitleAliases(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	features, _ := json.Marshal(CatalogFeatures{})
	media := CatalogMedia{CatalogKey: CatalogKey{MediaType: "movie", TMDBID: 9}, Features: features, Title: "Old title", Payload: []byte(`{"id":9,"title":"Old title"}`)}
	if err := database.PutCatalogMedia(ctx, media); err != nil {
		t.Fatal(err)
	}
	media.Title = "New title"
	media.Payload = []byte(`{"id":9,"title":"New title"}`)
	if err := database.PutCatalogMedia(ctx, media); err != nil {
		t.Fatal(err)
	}
	if _, total, err := database.SearchLocalCatalog(ctx, "movie", "old title", "en-US", 20); err != nil || total != 0 {
		t.Fatalf("stale title remained searchable: total=%d err=%v", total, err)
	}
}
