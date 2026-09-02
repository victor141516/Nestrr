package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestPeopleSearchAndCreditRelations(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	user, _ := database.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	people := []Person{
		{TMDBID: 10, Name: "Margot Robbie", ProfilePath: "/margot.jpg", KnownForDepartment: "Acting", Popularity: 80},
		{TMDBID: 20, Name: "Martin Scorsese", KnownForDepartment: "Directing", Popularity: 40},
	}
	credits := []MediaCredit{
		{MediaType: "movie", MediaTMDBID: 1, CreditKey: "cast-10", PersonTMDBID: 10, CreditKind: "cast", Character: "Lead", Order: 0},
		{MediaType: "movie", MediaTMDBID: 1, CreditKey: "crew-20", PersonTMDBID: 20, CreditKind: "crew", Department: "Directing", Job: "Director"},
	}
	if err := database.ReplaceMediaCredits(ctx, "movie", 1, people, credits); err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, user.ID, "movie", 1, "es-ES", []byte(`{"id":1,"title":"Una película"}`)); err != nil {
		t.Fatal(err)
	}

	matches, err := database.SearchPeople(ctx, "rob", 12)
	if err != nil || len(matches) != 1 || matches[0].TMDBID != 10 {
		t.Fatalf("unexpected local people search: %#v %v", matches, err)
	}
	media, err := database.LocalMediaForPeople(ctx, user.ID, "es-ES", "movie", []int{10}, []int{20})
	if err != nil || len(media) != 1 {
		t.Fatalf("unexpected local credit media: %#v %v", media, err)
	}
	var document map[string]any
	if json.Unmarshal(media[0], &document) != nil || document["title"] != "Una película" {
		t.Fatalf("unexpected local metadata: %s", media[0])
	}
}

func TestPersonSearchCacheKeepsRankAndEmptyResults(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	people := []Person{
		{TMDBID: 2, Name: "Second", Popularity: 2},
		{TMDBID: 1, Name: "First", Popularity: 100},
	}
	if err := database.PutPersonSearch(ctx, "test", "en-US", people); err != nil {
		t.Fatal(err)
	}
	cached, fresh, err := database.CachedPersonSearch(ctx, " TEST ", "en-US", 7*24*time.Hour)
	if err != nil || !fresh || len(cached) != 2 || cached[0].TMDBID != 2 {
		t.Fatalf("cached person order was lost: %#v %v %v", cached, fresh, err)
	}
	if err := database.PutPersonSearch(ctx, "nobody", "en-US", nil); err != nil {
		t.Fatal(err)
	}
	empty, fresh, err := database.CachedPersonSearch(ctx, "nobody", "en-US", 7*24*time.Hour)
	if err != nil || !fresh || len(empty) != 0 {
		t.Fatalf("empty person result was not cached: %#v %v %v", empty, fresh, err)
	}
}
