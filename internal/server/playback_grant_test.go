package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlaybackGrantResolvesMediaServerSideAndKeepsAPIKeyPrivate(t *testing.T) {
	s, movies, _ := newAvailabilityTestServer(t)
	path := filepath.Join(movies, "Example (2026).mkv")
	if err := os.WriteFile(path, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	var received struct {
		APIKey  string `json:"apiKey"`
		MediaID string `json:"mediaId"`
	}
	plarr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/playback-grants" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"playbackToken":"temporary-token"}`))
	}))
	t.Cleanup(plarr.Close)
	s.Config.PlarrAPIKey = "shared-secret"
	s.Config.PlarrInternalHost = plarr.URL
	s.HTTP = plarr.Client()

	body := `{"kind":"movie","tmdbId":10,"title":"Example","originalTitle":"Example","originalLanguage":"en","year":2026}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/playback/grant", strings.NewReader(body))
	response := httptest.NewRecorder()
	s.playbackGrant(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if received.APIKey != "shared-secret" {
		t.Fatalf("Plarr received API key %q", received.APIKey)
	}
	wantID := base64.RawURLEncoding.EncodeToString([]byte("Movies/Example (2026).mkv"))
	if received.MediaID != wantID {
		t.Fatalf("media id = %q, want %q", received.MediaID, wantID)
	}
	if strings.Contains(response.Body.String(), "shared-secret") {
		t.Fatal("API key leaked to browser response")
	}
	var got playbackGrantOutput
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.PlaybackToken != "temporary-token" {
		t.Fatalf("token = %q", got.PlaybackToken)
	}
	download, err := url.Parse(got.OriginalDownloadURL)
	if err != nil || download.Query().Get("playback_token") != "temporary-token" {
		t.Fatalf("download URL = %q, %v", got.OriginalDownloadURL, err)
	}
}

func TestPlaybackGrantPreservesUnauthenticatedPlarrCompatibility(t *testing.T) {
	s, movies, _ := newAvailabilityTestServer(t)
	if err := os.WriteFile(filepath.Join(movies, "Example (2026).mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/playback/grant", strings.NewReader(`{"kind":"movie","tmdbId":10,"originalTitle":"Example","originalLanguage":"en","year":2026}`))
	response := httptest.NewRecorder()
	s.playbackGrant(response, request)
	if response.Code != http.StatusCreated || strings.Contains(response.Body.String(), "playbackToken") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
