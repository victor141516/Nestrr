package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func catalogDetailRequest(user store.User, target string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	route := chi.NewRouteContext()
	route.URLParams.Add("*", "movie/42")
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, route)
	ctx = context.WithValue(ctx, claimsKey, auth.Claims{UserID: user.ID, IsAdmin: user.IsAdmin})
	return request.WithContext(ctx)
}

func TestCatalogMetadataIsSharedAndUsesOneNormalizedEffectiveLanguage(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "shared.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	admin, err := database.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	member, err := database.CreateUser(ctx, admin.ID, "member", "hash", "es_es")
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	seenLanguages := []string{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		seenLanguages = append(seenLanguages, r.URL.Query().Get("language"))
		_, _ = fmt.Fprintf(w, `{"id":42,"title":"%s"}`, r.URL.Query().Get("language"))
	}))
	defer upstream.Close()
	tmdbClient := &tmdb.Client{BaseURL: upstream.URL, TTL: time.Hour, HTTP: upstream.Client(), Store: database, Log: slog.Default()}
	server := &Server{Store: database, TMDB: tmdbClient, Log: slog.Default()}

	for _, user := range []store.User{admin, member} {
		response := httptest.NewRecorder()
		server.catalog(response, catalogDetailRequest(user, "/api/v1/catalog/movie/42"))
		if response.Code != http.StatusOK {
			t.Fatalf("user %d status=%d body=%s", user.ID, response.Code, response.Body.String())
		}
	}
	if requests != 1 || len(seenLanguages) != 1 || seenLanguages[0] != "es-ES" {
		t.Fatalf("shared locale fetched %d times with languages %#v", requests, seenLanguages)
	}

	request := catalogDetailRequest(member, "/api/v1/catalog/movie/42?language=en_us")
	request.URL.RawQuery = url.Values{"language": {"en_us"}}.Encode()
	response := httptest.NewRecorder()
	server.catalog(response, request)
	if response.Code != http.StatusOK || requests != 2 || seenLanguages[1] != "en-US" {
		t.Fatalf("explicit language was not normalized consistently: status=%d requests=%d languages=%#v", response.Code, requests, seenLanguages)
	}
	response = httptest.NewRecorder()
	request = catalogDetailRequest(admin, "/api/v1/catalog/movie/42?language=EN-us")
	request.URL.RawQuery = url.Values{"language": {"EN-us"}}.Encode()
	server.catalog(response, request)
	if response.Code != http.StatusOK || requests != 2 {
		t.Fatalf("normalized explicit language was not shared: status=%d requests=%d", response.Code, requests)
	}
}

func TestOnlyAdministratorCanClearSharedMetadata(t *testing.T) {
	database, err := store.Open(filepath.Join(t.TempDir(), "shared.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	admin, _ := database.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	member, _ := database.CreateUser(ctx, admin.ID, "member", "hash", "en-US")
	if err := database.PutMetadata(ctx, "movie", 1, "en-US", []byte(`{"id":1}`)); err != nil {
		t.Fatal(err)
	}
	manager := &auth.Manager{Store: database, Secret: []byte("12345678901234567890123456789012"), AccessTTL: time.Hour, RefreshTTL: time.Hour}
	server := &Server{Store: database, Auth: manager, Log: slog.Default()}
	handler := server.Handler()
	tokenFor := func(user store.User) string {
		tokens, issueErr := manager.Issue(ctx, user, 0)
		if issueErr != nil {
			t.Fatal(issueErr)
		}
		return tokens.AccessToken
	}
	request := func(method, path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}
	if response := request(http.MethodDelete, "/api/v1/me/metadata", tokenFor(admin)); response.Code != http.StatusNotFound {
		t.Fatalf("legacy user metadata route still exists: %d", response.Code)
	}
	if response := request(http.MethodDelete, "/api/v1/admin/metadata", tokenFor(member)); response.Code != http.StatusForbidden {
		t.Fatalf("member cleared shared metadata: %d", response.Code)
	}
	if response := request(http.MethodDelete, "/api/v1/admin/metadata", tokenFor(admin)); response.Code != http.StatusNoContent {
		t.Fatalf("administrator could not clear shared metadata: %d %s", response.Code, response.Body.String())
	}
	if _, err := database.GetMetadata(ctx, "movie", 1, "en-US"); err != sql.ErrNoRows {
		t.Fatalf("shared metadata was not cleared: %v", err)
	}
}
