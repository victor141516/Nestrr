package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
)

func settingsTestServer(t *testing.T) (*Server, store.User) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	user, err := db.CreateFirstAdmin(context.Background(), "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Store: db}, user
}

func patchSettings(t *testing.T, s *Server, userID int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("PATCH", "/api/v1/me/settings", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), claimsKey, auth.Claims{UserID: userID}))
	response := httptest.NewRecorder()
	s.settings(response, req)
	return response
}

func TestSettingsUpdatesAccentWithoutChangingLanguage(t *testing.T) {
	s, user := settingsTestServer(t)
	response := patchSettings(t, s, user.ID, `{"accentColor":"#38bdf8"}`)
	if response.Code != 200 {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
	var updated store.User
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Language != "es-ES" || updated.AccentColor != "#38bdf8" {
		t.Fatalf("unexpected settings: %#v", updated)
	}
}

func TestSettingsRejectsUnsupportedAccent(t *testing.T) {
	s, user := settingsTestServer(t)
	response := patchSettings(t, s, user.ID, `{"accentColor":"#000000"}`)
	if response.Code != 400 {
		t.Fatalf("unsupported accent returned %d: %s", response.Code, response.Body.String())
	}
	stored, err := s.Store.UserByID(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AccentColor != "#ff624f" {
		t.Fatalf("unsupported accent was persisted: %q", stored.AccentColor)
	}
}
