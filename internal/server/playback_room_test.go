package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWatchPartyCreationKeepsAPIKeyOnServer(t *testing.T) {
	s, movies, _ := newAvailabilityTestServer(t)
	if err := os.WriteFile(filepath.Join(movies, "Example (2026).mkv"), []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	var received struct {
		APIKey  string `json:"apiKey"`
		MediaID string `json:"mediaId"`
	}
	plarr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/watch-parties" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"roomId":"room-123","playerUrl":"https://plarr.example/#/play/id?room=room-123&playbackToken=short-lived"}`))
	}))
	defer plarr.Close()
	s.Config.PlarrAPIKey = "server-secret"
	s.Config.PlarrInternalHost = plarr.URL
	s.HTTP = plarr.Client()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/playback/rooms", strings.NewReader(`{"kind":"movie","tmdbId":10,"title":"Example","originalTitle":"Example","originalLanguage":"en","year":2026}`))
	response := httptest.NewRecorder()
	s.createPlaybackRoom(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if received.APIKey != "server-secret" || received.MediaID != base64.RawURLEncoding.EncodeToString([]byte("Movies/Example (2026).mkv")) {
		t.Fatalf("Plarr request: %+v", received)
	}
	if strings.Contains(response.Body.String(), "server-secret") {
		t.Fatal("server API key leaked")
	}
}
