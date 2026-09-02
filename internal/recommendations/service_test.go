package recommendations

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func recommendationTestStore(t *testing.T) *store.Store {
	t.Helper()
	database, err := store.Open(filepath.Join(t.TempDir(), "recommendations.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestParseCompleteMediaRejectsListingsAndIndexesFeatures(t *testing.T) {
	if _, err := parseCompleteMedia("movie", 1, []byte(`{"page":1,"results":[{"id":1}]}`)); err == nil {
		t.Fatal("listing payload was accepted as canonical detail")
	}
	payload := []byte(`{"id":1,"title":"Complete","genres":[{"id":12}],"keywords":{"keywords":[{"id":22}]},"production_companies":[{"id":32,"name":"Studio"}],"credits":{"cast":[{"id":42,"name":"Actor","credit_id":"cast","order":0}],"crew":[{"id":52,"name":"Director","credit_id":"crew","job":"Director","department":"Directing"}]}}`)
	parsed, err := parseCompleteMedia("movie", 1, payload)
	if err != nil {
		t.Fatal(err)
	}
	var features store.CatalogFeatures
	if json.Unmarshal(parsed.Catalog.Features, &features) != nil || len(features.Genres) != 1 || len(features.Keywords) != 1 || len(features.Cast) != 1 || len(features.Directors) != 1 || len(features.Companies) != 1 {
		t.Fatalf("incomplete feature extraction: %#v", features)
	}
}

func TestRunHydratesCandidatesAndRecommendationsStayLocal(t *testing.T) {
	database := recommendationTestStore(t)
	ctx := context.Background()
	user, _ := database.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err := database.SetSystemListMembership(ctx, user.ID, "favorites", store.MediaListItem{MediaType: "movie", TMDBID: 1, Title: "Seed"}, true); err != nil {
		t.Fatal(err)
	}

	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/movie/1":
			_, _ = w.Write([]byte(completeMovieJSON(1, "Seed", 10)))
		case r.URL.Path == "/movie/2":
			_, _ = w.Write([]byte(completeMovieJSON(2, "Candidate", 10)))
		case r.URL.Path == "/tv/3":
			_, _ = w.Write([]byte(completeShowJSON(3, "Series", 10)))
		case strings.HasPrefix(r.URL.Path, "/discover/movie"):
			_, _ = w.Write([]byte(`{"page":1,"total_pages":3,"results":[{"id":2,"title":"PARTIAL LISTING"}]}`))
		case strings.HasPrefix(r.URL.Path, "/discover/tv"):
			_, _ = w.Write([]byte(`{"page":1,"total_pages":3,"results":[{"id":3,"name":"PARTIAL LISTING"}]}`))
		case strings.HasSuffix(r.URL.Path, "/tv_credits"):
			_, _ = w.Write([]byte(`{"cast":[{"id":3}],"crew":[{"id":3,"job":"Creator"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()
	client := &tmdb.Client{BaseURL: upstream.URL, TTL: time.Hour, HTTP: upstream.Client(), Store: database, Log: slog.New(slog.NewTextHandler(os.Stderr, nil))}
	service := New(database, client, slog.Default())
	status, started, err := service.RunAsync()
	if err != nil || !started || status.State != "running" {
		t.Fatalf("run did not start: %#v %v %v", status, started, err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		status, err = service.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if status.State != "running" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status.State != "completed" {
		t.Fatalf("run did not complete: %#v", status)
	}
	if status.CatalogCount != 3 {
		t.Fatalf("expected three fully hydrated documents, got %#v", status)
	}
	media, err := database.CatalogMediaByKey(ctx, store.CatalogKey{MediaType: "movie", TMDBID: 2})
	if err != nil || media.Title != "Candidate" || strings.Contains(string(media.Payload), "PARTIAL") {
		t.Fatalf("listing payload leaked into canonical catalog: %#v %v", media, err)
	}
	beforeRecommendations := requestCount
	recommendations, err := service.Recommend(ctx, user.ID, "es-ES", "movie", 10)
	if err != nil || len(recommendations) != 1 || recommendations[0].ID != 2 {
		t.Fatalf("unexpected recommendations: %#v %v", recommendations, err)
	}
	if recommendations[0].Title != "Candidate ES" || recommendations[0].Overview != "Descripción" {
		t.Fatalf("recommendation was not localized from its complete document: %#v", recommendations[0])
	}
	if requestCount != beforeRecommendations {
		t.Fatal("live recommendation calculation called TMDB")
	}
}

func completeMovieJSON(id int, title string, genre int) string {
	return `{"id":` + jsonNumber(id) + `,"title":"` + title + `","overview":"Overview","poster_path":"/poster.jpg","release_date":"2025-01-01","vote_average":8,"vote_count":1000,"genres":[{"id":` + jsonNumber(genre) + `}],"keywords":{"keywords":[{"id":20}]},"production_companies":[{"id":30,"name":"Studio"}],"credits":{"cast":[{"id":40,"name":"Actor","credit_id":"cast","order":0}],"crew":[{"id":50,"name":"Director","credit_id":"crew","job":"Director"}]},"translations":{"translations":[{"iso_639_1":"es","data":{"title":"` + title + ` ES","overview":"Descripción"}}]}}`
}

func completeShowJSON(id int, title string, genre int) string {
	return `{"id":` + jsonNumber(id) + `,"name":"` + title + `","first_air_date":"2025-01-01","genres":[{"id":` + jsonNumber(genre) + `}],"keywords":{"results":[{"id":20}]},"production_companies":[{"id":30,"name":"Studio"}],"created_by":[{"id":50,"name":"Creator"}],"credits":{"cast":[{"id":40,"name":"Actor","credit_id":"cast","order":0}],"crew":[]}}`
}

func jsonNumber(value int) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
