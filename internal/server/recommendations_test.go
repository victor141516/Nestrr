package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/victor141516/Nestrr/internal/auth"
	recommendationservice "github.com/victor141516/Nestrr/internal/recommendations"
	"github.com/victor141516/Nestrr/internal/store"
)

func recommendationServer(t *testing.T) (*Server, store.User) {
	t.Helper()
	database, err := store.Open(filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	user, err := database.CreateFirstAdmin(context.Background(), "admin", "hash", "en-US")
	if err != nil {
		t.Fatal(err)
	}
	return &Server{Store: database, Recommendations: recommendationservice.New(database, nil, slog.Default())}, user
}

func recommendationRequest(method, target string, user store.User) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	claims := auth.Claims{UserID: user.ID, IsAdmin: user.IsAdmin}
	return request.WithContext(context.WithValue(request.Context(), claimsKey, claims))
}

func TestRecommendationEndpointReadsOnlyLocalCatalog(t *testing.T) {
	server, user := recommendationServer(t)
	features, _ := json.Marshal(store.CatalogFeatures{Genres: []int{12}})
	for _, media := range []store.CatalogMedia{
		{CatalogKey: store.CatalogKey{MediaType: "movie", TMDBID: 1}, Payload: []byte(`{"id":1,"title":"Seed"}`), Features: features, Title: "Seed"},
		{CatalogKey: store.CatalogKey{MediaType: "movie", TMDBID: 2}, Payload: []byte(`{"id":2,"title":"Candidate"}`), Features: features, Title: "Candidate"},
	} {
		if err := server.Store.PutCatalogMedia(context.Background(), media); err != nil {
			t.Fatal(err)
		}
	}
	if err := server.Store.SetSystemListMembership(context.Background(), user.ID, "favorites", store.MediaListItem{MediaType: "movie", TMDBID: 1, Title: "Seed"}, true); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.recommendations(response, recommendationRequest(http.MethodGet, "/api/v1/recommendations?kind=movie", user))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}
	if json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body.Results) != 1 || body.Results[0].ID != 2 {
		t.Fatalf("unexpected body: %s", response.Body.String())
	}
}

func TestAdminCanStartAndObserveRecommendationCatalogRun(t *testing.T) {
	server, user := recommendationServer(t)
	response := httptest.NewRecorder()
	server.runRecommendationCatalog(response, recommendationRequest(http.MethodPost, "/api/v1/admin/recommendations/catalog/run", user))
	if response.Code != http.StatusAccepted {
		t.Fatalf("unexpected run status %d: %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	server.recommendationCatalogStatus(response, recommendationRequest(http.MethodGet, "/api/v1/admin/recommendations/catalog", user))
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status response %d: %s", response.Code, response.Body.String())
	}
}
