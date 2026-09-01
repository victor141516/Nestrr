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
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
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
	catalog := New(filepath.Join(t.TempDir(), "keywords.json.gz"), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	catalog.replace(entries, time.Now())

	prefix := catalog.Prefix("super", 10)
	if len(prefix) != 2 || prefix[0].Name != "superhero" || prefix[1].Name != "superhero team" {
		t.Fatalf("unexpected prefix results: %#v", prefix)
	}
	contains := catalog.Contains("super", 10)
	if len(contains) != 1 || contains[0].Name != "masked superhero" {
		t.Fatalf("unexpected contains results: %#v", contains)
	}
	resolved := catalog.ByIDs([]int{180734, 9715, 180734})
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

	path := filepath.Join(t.TempDir(), "tmdb-keywords.json.gz")
	catalog := New(path, upstream.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	catalog.exportBase = upstream.URL
	catalog.now = func() time.Time { return now }
	if err := catalog.refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || !catalog.Ready() {
		t.Fatalf("refresh requests=%d ready=%v", requests, catalog.Ready())
	}
	stored, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(stored, payload) {
		t.Fatalf("persisted export mismatch: %v", err)
	}
}
