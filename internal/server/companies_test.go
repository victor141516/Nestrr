package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/victor141516/Nestrr/internal/store"
)

func TestParseMovieCompanies(t *testing.T) {
	companies, err := parseMovieCompanies([]byte(`{"production_companies":[{"id":1,"name":"Appian Way","logo_path":"/logo.png","origin_country":"US"}]}`))
	if err != nil || len(companies) != 1 || companies[0].Name != "Appian Way" || companies[0].OriginCountry != "US" {
		t.Fatalf("unexpected companies: %#v %v", companies, err)
	}
}

func TestCompanySearchCachesTMDBResults(t *testing.T) {
	var calls atomic.Int32
	service, _, user := peopleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/search/company" || r.URL.Query().Get("query") != "appian" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"results":[{"id":562,"name":"Appian Way","origin_country":"US"}]}`)
	}))
	for range 2 {
		response := httptest.NewRecorder()
		service.companySearch(response, requestWithUser(http.MethodGet, "/api/v1/companies/search?q=appian&phase=remote", user.ID))
		if response.Code != http.StatusOK {
			t.Fatalf("search returned %d: %s", response.Code, response.Body.String())
		}
		var payload struct {
			Results []store.Company `json:"results"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Results) != 1 || payload.Results[0].TMDBID != 562 {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("TMDB calls = %d, want 1", calls.Load())
	}
}
