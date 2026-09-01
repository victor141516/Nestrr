package tmdb

import (
	"context"
	"github.com/victor141516/Nestrr/internal/store"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNormalizedRequestCache(t *testing.T) {
	var calls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer up.Close()
	s, err := store.Open(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	c := Client{BaseURL: up.URL, Token: "token", TTL: time.Hour, HTTP: up.Client(), Store: s, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	q1 := url.Values{"language": {"es-ES"}, "page": {"1"}}
	q2 := url.Values{"page": {"1"}, "language": {"es-ES"}}
	for _, q := range []url.Values{q1, q2} {
		if _, err = c.Get(context.Background(), "/movie/1", q, false); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("upstream called %d times", calls.Load())
	}
}
