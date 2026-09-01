package ratings

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

	"github.com/victor141516/Nestrr/internal/store"
)

func TestMDBListBatchUsesOMDbFallbackAndPermanentCache(t *testing.T) {
	var mdbCalls, omdbCalls atomic.Int32
	mdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mdbCalls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/rating/movie/tomatoes" || r.URL.Query().Get("apikey") != "mdb-key" {
			t.Fatalf("unexpected MDBList request: %s %s", r.Method, r.URL.String())
		}
		var body struct {
			IDs      []string `json:"ids"`
			Provider string   `json:"provider"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || len(body.IDs) != 2 || body.Provider != "tmdb" {
			t.Fatalf("unexpected batch body: %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{
			"provider_id":"tmdb",
			"provider_rating":"tomatoes",
			"mediatype":"movie",
			"ratings":[{"id":1,"rating":91}]
		}`)
	}))
	defer mdb.Close()
	omdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		omdbCalls.Add(1)
		if r.URL.Query().Get("i") != "tt0000002" || r.URL.Query().Get("apikey") != "omdb-key" {
			t.Fatalf("unexpected OMDb request: %s", r.URL.String())
		}
		io.WriteString(w, `{"Response":"True","Ratings":[{"Source":"Rotten Tomatoes","Value":"73%"}]}`)
	}))
	defer omdb.Close()

	database, err := store.Open(filepath.Join(t.TempDir(), "ratings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := &Service{
		MDBListBaseURL: mdb.URL,
		MDBListAPIKey:  "mdb-key",
		OMDbBaseURL:    omdb.URL,
		OMDbAPIKey:     "omdb-key",
		HTTP:           mdb.Client(),
		Store:          database,
		Log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	references := []Reference{
		{MediaType: "movie", TMDBID: 1, IMDbID: "tt0000001"},
		{MediaType: "movie", TMDBID: 2, IMDbID: "tt0000002"},
	}
	resolved := service.Resolve(context.Background(), references, false)
	if resolved["movie:1"].RottenTomatoesScore != 91 || resolved["movie:1"].Provider != "mdblist" {
		t.Fatalf("MDBList rating was not selected: %#v", resolved["movie:1"])
	}
	if resolved["movie:2"].RottenTomatoesScore != 73 || resolved["movie:2"].Provider != "omdb" {
		t.Fatalf("OMDb fallback was not selected: %#v", resolved["movie:2"])
	}
	if mdbCalls.Load() != 1 || omdbCalls.Load() != 1 {
		t.Fatalf("unexpected initial calls: MDBList=%d OMDb=%d", mdbCalls.Load(), omdbCalls.Load())
	}

	resolved = service.Resolve(context.Background(), references, false)
	if resolved["movie:1"].RottenTomatoesScore != 91 || resolved["movie:2"].RottenTomatoesScore != 73 {
		t.Fatalf("cached ratings changed: %#v", resolved)
	}
	if mdbCalls.Load() != 1 || omdbCalls.Load() != 1 {
		t.Fatalf("permanent cache was bypassed: MDBList=%d OMDb=%d", mdbCalls.Load(), omdbCalls.Load())
	}

	service.Resolve(context.Background(), references, true)
	if mdbCalls.Load() != 2 || omdbCalls.Load() != 2 {
		t.Fatalf("force refresh did not query providers: MDBList=%d OMDb=%d", mdbCalls.Load(), omdbCalls.Load())
	}
}

func TestNegativeResultIsCachedButMissingIMDbIsNotInvented(t *testing.T) {
	var calls atomic.Int32
	omdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		io.WriteString(w, `{"Response":"True","Ratings":[]}`)
	}))
	defer omdb.Close()
	database, err := store.Open(filepath.Join(t.TempDir(), "ratings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := &Service{OMDbBaseURL: omdb.URL, OMDbAPIKey: "key", HTTP: omdb.Client(), Store: database}

	if got := service.Resolve(context.Background(), []Reference{{MediaType: "tv", TMDBID: 10}}, false); len(got) != 0 {
		t.Fatalf("rating without an IMDb ID was invented: %#v", got)
	}
	if calls.Load() != 0 {
		t.Fatal("OMDb was called without an IMDb ID")
	}

	reference := Reference{MediaType: "tv", TMDBID: 10, IMDbID: "tt0000010"}
	first := service.Resolve(context.Background(), []Reference{reference}, false)
	if first["tv:10"].Found || first["tv:10"].Provider != "omdb" {
		t.Fatalf("negative lookup was not represented: %#v", first)
	}
	service.Resolve(context.Background(), []Reference{reference}, false)
	if calls.Load() != 1 {
		t.Fatalf("negative cache was bypassed: %d", calls.Load())
	}
}

func TestMDBListBatchUsesFreeTierChunkSize(t *testing.T) {
	var calls atomic.Int32
	mdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/rating/show/tomatoes" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			IDs []string `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.IDs) == 0 || len(body.IDs) > maxMDBListBatch {
			t.Fatalf("invalid free-tier batch size: %d", len(body.IDs))
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"provider_id":"tmdb","provider_rating":"tomatoes","mediatype":"show","ratings":[]}`)
	}))
	defer mdb.Close()
	database, err := store.Open(filepath.Join(t.TempDir(), "ratings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := &Service{MDBListBaseURL: mdb.URL, MDBListAPIKey: "key", HTTP: mdb.Client(), Store: database}
	references := make([]Reference, 11)
	for index := range references {
		references[index] = Reference{MediaType: "tv", TMDBID: index + 1}
	}
	service.Resolve(context.Background(), references, false)
	if calls.Load() != 2 {
		t.Fatalf("expected two batches, got %d", calls.Load())
	}
}
