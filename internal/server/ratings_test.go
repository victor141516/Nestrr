package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/ratings"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func TestCatalogRatingsAreEnrichedAndCached(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.WriteString(w, `[{"tmdb_id":42,"ratings":[{"source":"tomatoes","score":88}]}]`)
	}))
	defer upstream.Close()
	database, err := store.Open(filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	server := &Server{Ratings: &ratings.Service{
		MDBListBaseURL: upstream.URL,
		MDBListAPIKey:  "key",
		HTTP:           upstream.Client(),
		Store:          database,
	}}
	payload := []byte(`{"page":1,"results":[{"id":42,"title":"Film"}]}`)

	enriched := server.enrichCatalogRatings(context.Background(), "discover/movie", payload, false)
	var document struct {
		Results []struct {
			Score int `json:"rotten_tomatoes_score"`
		} `json:"results"`
	}
	if json.Unmarshal(enriched, &document) != nil || document.Results[0].Score != 88 {
		t.Fatalf("catalog response was not enriched: %s", enriched)
	}
	server.enrichCatalogRatings(context.Background(), "discover/movie", payload, false)
	if calls.Load() != 1 {
		t.Fatalf("catalog enrichment bypassed the permanent cache: %d", calls.Load())
	}
	server.enrichCatalogRatings(context.Background(), "discover/movie", payload, true)
	if calls.Load() != 2 {
		t.Fatalf("forced enrichment did not refresh the provider: %d", calls.Load())
	}
}

func TestSetupPublishesRatingCapability(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "setup.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/setup", nil)

	response := httptest.NewRecorder()
	(&Server{Store: database}).setup(response, request)
	var disabled map[string]any
	json.Unmarshal(response.Body.Bytes(), &disabled)
	if disabled["rottenTomatoesEnabled"] != false {
		t.Fatalf("unconfigured ratings were advertised: %s", response.Body.String())
	}

	response = httptest.NewRecorder()
	(&Server{Store: database, Ratings: &ratings.Service{MDBListAPIKey: "key"}}).setup(response, request)
	var enabled map[string]any
	json.Unmarshal(response.Body.Bytes(), &enabled)
	if enabled["rottenTomatoesEnabled"] != true {
		t.Fatalf("configured ratings were hidden: %s", response.Body.String())
	}
}

func TestSeriesDetailResolvesIMDbIDForOMDbFallback(t *testing.T) {
	tmdbUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv/7/external_ids" {
			t.Fatalf("unexpected TMDB path: %s", r.URL.Path)
		}
		io.WriteString(w, `{"imdb_id":"tt0000007"}`)
	}))
	defer tmdbUpstream.Close()
	omdbUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("i") != "tt0000007" {
			t.Fatalf("unexpected IMDb ID: %s", r.URL.String())
		}
		io.WriteString(w, `{"Response":"True","Ratings":[{"Source":"Rotten Tomatoes","Value":"84%"}]}`)
	}))
	defer omdbUpstream.Close()
	database, err := store.Open(filepath.Join(t.TempDir(), "series.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := &Server{
		TMDB:    &tmdb.Client{BaseURL: tmdbUpstream.URL, Token: "token", TTL: time.Hour, HTTP: tmdbUpstream.Client(), Store: database, Log: logger},
		Ratings: &ratings.Service{OMDbBaseURL: omdbUpstream.URL, OMDbAPIKey: "key", HTTP: omdbUpstream.Client(), Store: database, Log: logger},
	}
	enriched := server.enrichCatalogRatings(context.Background(), "tv/7", []byte(`{"id":7,"name":"Series"}`), false)
	var document map[string]any
	if json.Unmarshal(enriched, &document) != nil || int(document["rotten_tomatoes_score"].(float64)) != 84 || document["imdb_id"] != "tt0000007" {
		t.Fatalf("series detail did not use its external IMDb ID: %s", enriched)
	}
}
