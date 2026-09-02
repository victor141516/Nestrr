package store

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCatalogMediaHydrationReplacesStagedCandidate(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	key := CatalogKey{MediaType: "movie", TMDBID: 42}
	added, err := database.AddRecommendationCandidates(ctx, []CatalogKey{key, key})
	if err != nil || added != 1 {
		t.Fatalf("stage candidate: added=%d err=%v", added, err)
	}
	features, _ := json.Marshal(CatalogFeatures{Genres: []int{12}})
	if err := database.PutCatalogMedia(ctx, CatalogMedia{CatalogKey: key, Payload: []byte(`{"id":42,"title":"Complete"}`), Features: features, Title: "Complete"}); err != nil {
		t.Fatal(err)
	}
	stored, err := database.CatalogMediaByKey(ctx, key)
	if err != nil || stored.Title != "Complete" {
		t.Fatalf("unexpected canonical media: %#v %v", stored, err)
	}
	counts, err := database.RecommendationCandidateCounts(ctx)
	if err != nil || counts["pending"] != 0 {
		t.Fatalf("hydrated candidate remained staged: %#v %v", counts, err)
	}
}

func TestRecommendationSeedsPreferFavoritesAndExcludeKnownMedia(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	user, _ := database.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	if err := database.SetSystemListMembership(ctx, user.ID, "favorites", MediaListItem{MediaType: "movie", TMDBID: 10, Title: "Favorite"}, true); err != nil {
		t.Fatal(err)
	}
	if err := database.SetSystemListMembership(ctx, user.ID, "watched", MediaListItem{MediaType: "tv", TMDBID: 20, Title: "Watched"}, true); err != nil {
		t.Fatal(err)
	}
	seeds, err := database.RecommendationSeeds(ctx, user.ID)
	if err != nil || len(seeds) != 2 {
		t.Fatalf("unexpected seeds: %#v %v", seeds, err)
	}
	weights := map[CatalogKey]float64{}
	for _, seed := range seeds {
		weights[seed.CatalogKey] = seed.Weight
	}
	if weights[(CatalogKey{MediaType: "movie", TMDBID: 10})] != 5 || weights[(CatalogKey{MediaType: "tv", TMDBID: 20})] != 2 {
		t.Fatalf("unexpected seed weights: %#v", weights)
	}
	excluded, err := database.RecommendationExclusions(ctx, user.ID)
	if err != nil || !excluded[CatalogKey{MediaType: "movie", TMDBID: 10}] || !excluded[CatalogKey{MediaType: "tv", TMDBID: 20}] {
		t.Fatalf("known media was not excluded: %#v %v", excluded, err)
	}
}

func TestRecommendationSeedsTreatCompletedEpisodesAsOneShowSignal(t *testing.T) {
	ctx := context.Background()
	database := testStore(t)
	user, err := database.CreateFirstAdmin(ctx, "episode-viewer", "hash", "en-US")
	if err != nil {
		t.Fatal(err)
	}
	for _, episodeID := range []int{101, 102} {
		if _, err = database.SetEpisodeWatched(ctx, user.ID, Progress{MediaType: "episode", TMDBID: episodeID, ShowTMDBID: 88, SeasonNumber: 1, EpisodeNumber: episodeID - 100, DurationSeconds: 1200}, true); err != nil {
			t.Fatal(err)
		}
	}
	seeds, err := database.RecommendationSeeds(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 1 || seeds[0].MediaType != "tv" || seeds[0].TMDBID != 88 || seeds[0].Weight != 2 {
		t.Fatalf("expected one show seed with stable weight, got %#v", seeds)
	}
}

func TestRecommendationCursorAndRunStatusPersist(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	if page, err := database.RecommendationDiscoveryPage(ctx, "movie", "genre", 12, "popularity.desc"); err != nil || page != 1 {
		t.Fatalf("unexpected initial cursor: %d %v", page, err)
	}
	if err := database.SetRecommendationDiscoveryPage(ctx, "movie", "genre", 12, "popularity.desc", 8); err != nil {
		t.Fatal(err)
	}
	if page, err := database.RecommendationDiscoveryPage(ctx, "movie", "genre", 12, "popularity.desc"); err != nil || page != 8 {
		t.Fatalf("unexpected stored cursor: %d %v", page, err)
	}
	id, err := database.CreateRecommendationRun(ctx, "running", []byte(`{"state":"running"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpdateRecommendationRun(ctx, id, "completed", []byte(`{"state":"completed"}`), true); err != nil {
		t.Fatal(err)
	}
	run, err := database.LatestRecommendationRun(ctx)
	if err != nil || run.State != "completed" || run.FinishedAt == "" {
		t.Fatalf("unexpected persisted run: %#v %v", run, err)
	}
}
