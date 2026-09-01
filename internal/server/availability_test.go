package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/config"
	"github.com/victor141516/Nestrr/internal/media"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func newAvailabilityTestServer(t *testing.T) (*Server, string, string) {
	t.Helper()
	root := t.TempDir()
	movies := filepath.Join(root, "Movies")
	shows := filepath.Join(root, "Shows")
	if err := os.MkdirAll(movies, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(shows, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(root, "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Server{
		Config: config.Config{
			MovieLibraryRoot: movies,
			ShowLibraryRoot:  shows,
			PlarrHost:        "http://plarr.test",
		},
		Store: db,
		Media: media.Resolver{MoviesRoot: movies, ShowsRoot: shows},
	}, movies, shows
}

func TestResolveAvailabilityUsesEnglishMovieNameInsteadOfLocalizedDisplayName(t *testing.T) {
	s, movies, _ := newAvailabilityTestServer(t)
	canonicalPath := filepath.Join(movies, "Dune - Part Two (2024).mkv")
	if err := os.WriteFile(canonicalPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/693134" || r.URL.Query().Get("language") != "en-US" {
			t.Errorf("unexpected TMDB request: %s?%s", r.URL.Path, r.URL.RawQuery)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":693134,"title":"Dune: Part Two"}`)
	}))
	t.Cleanup(upstream.Close)
	s.TMDB = &tmdb.Client{
		BaseURL: upstream.URL,
		TTL:     time.Hour,
		HTTP:    upstream.Client(),
		Store:   s.Store,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	got := s.resolveAvailability(context.Background(), availabilityInput{
		Kind:   "movie",
		TMDBID: 693134,
		Title:  "Dune: Parte dos",
		Year:   2024,
	})

	if !got.Available || got.Path != canonicalPath {
		t.Fatalf("availability = %#v; want canonical English path %q", got, canonicalPath)
	}
}

func TestResolveAvailabilityFetchesEnglishShowNameForNonEnglishOriginal(t *testing.T) {
	s, _, shows := newAvailabilityTestServer(t)
	episodePath := filepath.Join(shows, "Money Heist", "Season 1", "Money Heist S01E01.mkv")
	if err := os.MkdirAll(filepath.Dir(episodePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(episodePath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv/71446" || r.URL.Query().Get("language") != "en-US" {
			t.Errorf("unexpected TMDB request: %s?%s", r.URL.Path, r.URL.RawQuery)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":71446,"name":"Money Heist"}`)
	}))
	t.Cleanup(upstream.Close)
	s.TMDB = &tmdb.Client{
		BaseURL: upstream.URL,
		TTL:     time.Hour,
		HTTP:    upstream.Client(),
		Store:   s.Store,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	got := s.resolveAvailability(context.Background(), availabilityInput{
		Kind:             "episode",
		TMDBID:           123,
		Show:             "La casa de papel",
		OriginalShow:     "La casa de papel",
		OriginalLanguage: "es",
		ShowTMDBID:       71446,
		Season:           1,
		Episode:          1,
	})

	if !got.Available || got.Path != episodePath {
		t.Fatalf("availability = %#v; want canonical English path %q", got, episodePath)
	}
}
