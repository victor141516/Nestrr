package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func TestBulkTitleCandidatesRemoveParenthesesFromTheRight(t *testing.T) {
	got := bulkTitleCandidates("La Odisea (2026) (The Asylum)")
	want := []string{
		"La Odisea (2026) (The Asylum)",
		"La Odisea (2026)",
		"La Odisea",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("bulk title candidates = %#v, want %#v", got, want)
	}
}

func TestFindStoredBulkMatchUsesLocalizedAndEnglishTitles(t *testing.T) {
	documents := []store.MetadataDocument{
		{MediaType: "movie", TMDBID: 550, Language: "es-ES", Payload: []byte(`{"id":550,"title":"El club de la lucha","original_title":"Fight Club","popularity":50}`)},
		{MediaType: "movie", TMDBID: 550, Language: "en-US", Payload: []byte(`{"id":550,"title":"Fight Club","original_title":"Fight Club","popularity":50}`)},
	}
	for _, input := range []string{"El club de la lucha", "Fight Club", "Fight Club (1999) (4K)"} {
		match, found := findStoredBulkMatch(input, documents, "es-ES")
		if !found || match.ID != 550 || match.MediaType != "movie" {
			t.Fatalf("%q did not resolve to the cached movie: found=%v match=%#v", input, found, match)
		}
	}
}

func TestBulkImportStreamsDatabaseTMDBAndMissingResults(t *testing.T) {
	ctx := context.Background()
	database, err := store.Open(filepath.Join(t.TempDir(), "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	user, err := database.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	list, err := database.CreateMediaList(ctx, user.ID, "Cine")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, user.ID, "movie", 550, "es-ES", []byte(`{"id":550,"title":"El club de la lucha","original_title":"Fight Club","poster_path":"/fight.jpg"}`)); err != nil {
		t.Fatal(err)
	}

	var searchQueries []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/multi":
			query := r.URL.Query().Get("query")
			searchQueries = append(searchQueries, query)
			if query == "Brand New Movie (2026)" {
				_, _ = io.WriteString(w, `{"results":[{"id":900,"media_type":"movie","title":"Brand New Movie","popularity":80}]}`)
				return
			}
			_, _ = io.WriteString(w, `{"results":[]}`)
		case "/movie/900":
			if r.URL.Query().Get("language") == "es-ES" {
				_, _ = io.WriteString(w, `{"id":900,"title":"Película nueva","original_title":"Brand New Movie","poster_path":"/new.jpg","runtime":101}`)
				return
			}
			_, _ = io.WriteString(w, `{"id":900,"title":"Brand New Movie","original_title":"Brand New Movie","poster_path":"/new.jpg","runtime":101}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(upstream.Close)

	service := &Server{
		Store: database,
		TMDB: &tmdb.Client{
			BaseURL: upstream.URL,
			Token:   "token",
			TTL:     time.Hour,
			HTTP:    upstream.Client(),
			Store:   database,
			Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	}
	body := bytes.NewBufferString(`{"titles":"El club de la lucha (1999) (Blu-ray)\nBrand New Movie (2026)\nBrand New Movie (2026)\nUnknown Work"}`)
	request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/lists/%d/bulk-import", list.ID), body)
	route := chi.NewRouteContext()
	route.URLParams.Add("listID", fmt.Sprint(list.ID))
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
	request = request.WithContext(context.WithValue(request.Context(), claimsKey, auth.Claims{UserID: user.ID}))
	response := httptest.NewRecorder()

	service.bulkImportMediaList(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("bulk import returned %d: %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "application/x-ndjson; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	events := []bulkImportEvent{}
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		var event bulkImportEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode event %q: %v", scanner.Text(), err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	assertEvent := func(index int, status bulkImportStatus, source string) bulkImportEvent {
		t.Helper()
		for _, event := range events {
			if event.Index == index && event.Status == status && event.Source == source {
				return event
			}
		}
		t.Fatalf("missing event index=%d status=%s source=%s in %#v", index, status, source, events)
		return bulkImportEvent{}
	}
	if saved := assertEvent(0, bulkSaved, "database"); !saved.Added || saved.Item == nil || saved.Item.TMDBID != 550 {
		t.Fatalf("unexpected database saved event: %#v", saved)
	}
	if saved := assertEvent(1, bulkSaved, "tmdb"); !saved.Added || saved.Item == nil || saved.Item.Title != "Película nueva" {
		t.Fatalf("unexpected TMDB saved event: %#v", saved)
	}
	if saved := assertEvent(2, bulkSaved, "database"); saved.Added || saved.Item == nil || saved.Item.AddedAt == "" {
		t.Fatalf("unexpected repeated saved event: %#v", saved)
	}
	assertEvent(3, bulkNotFound, "tmdb")
	if events[len(events)-1].Status != bulkComplete {
		t.Fatalf("last event = %#v", events[len(events)-1])
	}
	if !slices.Equal(searchQueries, []string{"Brand New Movie (2026)", "Unknown Work"}) {
		t.Fatalf("TMDB received queries %#v", searchQueries)
	}
	items, err := database.MediaListItems(ctx, user.ID, list.ID, "", "addedAt", "desc")
	if err != nil || len(items) != 2 {
		t.Fatalf("saved items = %#v, err=%v", items, err)
	}
	if _, err := database.GetMetadata(ctx, user.ID, "movie", 900, "en-US"); err != nil {
		t.Fatalf("English TMDB metadata was not cached: %v", err)
	}
	if _, err := database.GetMetadata(ctx, user.ID, "movie", 900, "es-ES"); err != nil {
		t.Fatalf("localized TMDB metadata was not cached: %v", err)
	}

}
