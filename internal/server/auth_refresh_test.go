package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
)

func TestRefreshInfrastructureFailureDoesNotClearValidCookie(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "refresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	hash, _ := auth.HashPassword("a secure password")
	user, _ := database.CreateFirstAdmin(ctx, "admin", hash, "en-US")
	manager := &auth.Manager{Store: database, Secret: []byte("12345678901234567890123456789012"), AccessTTL: time.Hour, RefreshTTL: time.Hour}
	tokens, err := manager.Issue(ctx, user, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: "nestrr_refresh", Value: tokens.RefreshToken})
	response := httptest.NewRecorder()
	(&Server{Auth: manager}).refresh(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("infrastructure failure returned %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Header().Get("Set-Cookie"), "nestrr_refresh=") {
		t.Fatalf("valid refresh cookie was cleared: %q", response.Header().Get("Set-Cookie"))
	}
}

func TestInvalidRefreshClearsCookie(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "refresh.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	manager := &auth.Manager{Store: database, Secret: []byte("12345678901234567890123456789012"), AccessTTL: time.Hour, RefreshTTL: time.Hour}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	request.AddCookie(&http.Cookie{Name: "nestrr_refresh", Value: "not-a-valid-session"})
	response := httptest.NewRecorder()
	(&Server{Auth: manager}).refresh(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("invalid refresh returned %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Header().Get("Set-Cookie"), "nestrr_refresh=") {
		t.Fatalf("invalid refresh cookie was not cleared: %q", response.Header().Get("Set-Cookie"))
	}
}
