package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/victor141516/Nestrr/internal/store"
)

func TestLocalCatalogSearchUsesSQLiteAndLocalizesCanonicalDocuments(t *testing.T) {
	service, database, user := peopleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("local catalog search unexpectedly called TMDB: %s", r.URL.String())
	}))
	features, _ := json.Marshal(store.CatalogFeatures{Genres: []int{18}})
	if err := database.PutCatalogMedia(context.Background(), store.CatalogMedia{
		CatalogKey: store.CatalogKey{MediaType: "movie", TMDBID: 101},
		Payload:    []byte(`{"id":101,"title":"Inglourious Basterds","original_title":"Inglourious Basterds","overview":"English","poster_path":"/poster.jpg","genres":[{"id":18}],"translations":{"translations":[{"iso_639_1":"es","data":{"title":"Malditos bastardos","overview":"Descripción"}}]}}`),
		Features:   features,
		Title:      "Inglourious Basterds",
	}); err != nil {
		t.Fatal(err)
	}
	request := requestWithUser(http.MethodGet, "/api/v1/catalog/local/search/movie?query=malditos", user.ID)
	route := chi.NewRouteContext()
	route.URLParams.Add("kind", "movie")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, route))
	response := httptest.NewRecorder()
	service.localCatalogSearch(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Results []localCatalogCard `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Results) != 1 {
		t.Fatalf("unexpected payload: %s err=%v", response.Body.String(), err)
	}
	if payload.Results[0].Title != "Malditos bastardos" || payload.Results[0].Overview != "Descripción" || len(payload.Results[0].GenreIDs) != 1 {
		t.Fatalf("localized card was not produced: %#v", payload.Results[0])
	}
}
