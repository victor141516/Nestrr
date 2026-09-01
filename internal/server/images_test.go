package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/victor141516/Nestrr/internal/config"
	"github.com/victor141516/Nestrr/internal/store"
)

func newImageTestServer(t *testing.T, upstream http.Handler) (*Server, *httptest.Server) {
	t.Helper()
	remote := httptest.NewServer(upstream)
	t.Cleanup(remote.Close)
	db, err := store.Open(filepath.Join(t.TempDir(), "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Server{
		Config: config.Config{
			ImageCacheDir:    t.TempDir(),
			TMDBImageBaseURL: remote.URL + "/t/p/original",
		},
		Store: db,
		HTTP:  remote.Client(),
	}, remote
}

func TestImageProxyCachesExactImmutableVariant(t *testing.T) {
	var mu sync.Mutex
	requests := make([]string, 0, 2)
	s, _ := newImageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = io.WriteString(w, "image:"+r.URL.Path)
	}))

	request := func(size string, etag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/images/cache?path=%2Fposter.jpg&size="+size, nil)
		if etag != "" {
			r.Header.Set("If-None-Match", etag)
		}
		w := httptest.NewRecorder()
		s.imageProxy(w, r)
		return w
	}

	first := request("w500", "")
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d; want 200; body=%s", first.Code, first.Body.String())
	}
	if got := first.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if got := first.Header().Get("ETag"); got == "" {
		t.Fatal("ETag is empty")
	}
	if got := first.Body.String(); got != "image:/t/p/w500/poster.jpg" {
		t.Fatalf("body = %q", got)
	}

	second := request("w500", "")
	if second.Code != http.StatusOK || second.Body.String() != first.Body.String() {
		t.Fatalf("cached response = (%d, %q)", second.Code, second.Body.String())
	}
	third := request("w780", "")
	if third.Code != http.StatusOK || third.Body.String() != "image:/t/p/w780/poster.jpg" {
		t.Fatalf("second variant = (%d, %q)", third.Code, third.Body.String())
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 2 || requests[0] != "/t/p/w500/poster.jpg" || requests[1] != "/t/p/w780/poster.jpg" {
		t.Fatalf("upstream requests = %#v", requests)
	}
}

func TestImageProxyHonorsETagAndRejectsUnknownSize(t *testing.T) {
	requests := 0
	s, _ := newImageTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = io.WriteString(w, "image")
	}))

	firstRequest := httptest.NewRequest(http.MethodGet, "/api/v1/images/cache?path=%2Fposter.jpg&size=w500", nil)
	first := httptest.NewRecorder()
	s.imageProxy(first, firstRequest)

	conditionalRequest := httptest.NewRequest(http.MethodGet, "/api/v1/images/cache?path=%2Fposter.jpg&size=w500", nil)
	conditionalRequest.Header.Set("If-None-Match", first.Header().Get("ETag"))
	conditional := httptest.NewRecorder()
	s.imageProxy(conditional, conditionalRequest)
	if conditional.Code != http.StatusNotModified {
		t.Fatalf("conditional status = %d; want 304", conditional.Code)
	}

	invalidRequest := httptest.NewRequest(http.MethodGet, "/api/v1/images/cache?path=%2Fposter.jpg&size=w999", nil)
	invalid := httptest.NewRecorder()
	s.imageProxy(invalid, invalidRequest)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid size status = %d; want 400", invalid.Code)
	}
	if requests != 1 {
		t.Fatalf("upstream requests = %d; want 1", requests)
	}
}
