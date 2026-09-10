package keywords

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

func export(t *testing.T, entries []Entry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	encoder := json.NewEncoder(writer)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func fixtureEntries() []Entry {
	entries := []Entry{
		{ID: 9715, Name: "superhero"},
		{ID: 155030, Name: "superhero team"},
		{ID: 180734, Name: "masked superhero"},
		{ID: 123, Name: "hero's journey"},
	}
	for index := 0; index < 1_001; index++ {
		entries = append(entries, Entry{ID: 500_000 + index, Name: "fixture keyword " + strconv.Itoa(index)})
	}
	return entries
}

func TestSearchesPrefixBeforeInternalMatchesAndResolvesIDs(t *testing.T) {
	entries, err := decodeExport(export(t, fixtureEntries()))
	if err != nil {
		t.Fatal(err)
	}
	database, err := store.Open(filepath.Join(t.TempDir(), "keywords.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	values := make([]store.Keyword, len(entries))
	for index, entry := range entries {
		values[index] = store.Keyword{ID: entry.ID, Name: entry.Name}
	}
	if err := database.ReplaceKeywordCatalog(context.Background(), values, time.Now()); err != nil {
		t.Fatal(err)
	}
	catalog := New(database, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))

	prefix, err := catalog.Prefix(context.Background(), "super", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefix) != 2 || prefix[0].Name != "superhero" || prefix[1].Name != "superhero team" {
		t.Fatalf("unexpected prefix results: %#v", prefix)
	}
	contains, err := catalog.Contains(context.Background(), "super", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(contains) != 1 || contains[0].Name != "masked superhero" {
		t.Fatalf("unexpected contains results: %#v", contains)
	}
	resolved, err := catalog.ByIDs(context.Background(), []int{180734, 9715, 180734})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 2 || resolved[0].ID != 180734 || resolved[1].ID != 9715 {
		t.Fatalf("unexpected resolved results: %#v", resolved)
	}
}

func TestRefreshUsesLatestAvailableExportAndPersistsIt(t *testing.T) {
	now := time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)
	payload := export(t, fixtureEntries())
	requests := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/keyword_ids_09_01_2026.json.gz" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path != "/keyword_ids_08_31_2026.json.gz" {
			t.Fatalf("unexpected export path %q", r.URL.Path)
		}
		_, _ = w.Write(payload)
	}))
	defer upstream.Close()

	database, err := store.Open(filepath.Join(t.TempDir(), "keywords.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	catalog := New(database, upstream.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	catalog.exportBase = upstream.URL
	catalog.now = func() time.Time { return now }
	if err := catalog.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !catalog.Ready(context.Background()) {
		t.Fatalf("refresh requests=%d ready=%v", requests, catalog.Ready(context.Background()))
	}
	stored, err := database.SearchKeywords(context.Background(), "super", "prefix", 10)
	if err != nil || len(stored) != 2 {
		t.Fatalf("database export mismatch: %#v %v", stored, err)
	}
}
