package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func peopleTestServer(t *testing.T, upstream http.Handler) (*Server, *store.Store, store.User) {
	t.Helper()
	database, err := store.Open(filepath.Join(t.TempDir(), "people.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	user, err := database.CreateFirstAdmin(context.Background(), "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(upstream)
	t.Cleanup(remote.Close)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return &Server{
		Store: database,
		TMDB: &tmdb.Client{
			BaseURL: remote.URL,
			Token:   "token",
			TTL:     time.Hour,
			HTTP:    remote.Client(),
			Store:   database,
			Log:     logger,
		},
		Log: logger,
	}, database, user
}

func requestWithUser(method, target string, userID int64) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	return request.WithContext(context.WithValue(request.Context(), claimsKey, auth.Claims{UserID: userID}))
}

func TestParseMediaCreditsRetainsCastAndExactCrewJobs(t *testing.T) {
	people, credits, err := parseMediaCredits("movie", 42, []byte(`{
		"cast":[{"id":1,"name":"Actor","character":"Lead","order":0,"credit_id":"cast-1"}],
		"crew":[
			{"id":2,"name":"Director","department":"Directing","job":"Director","credit_id":"crew-2"},
			{"id":3,"name":"Writer","department":"Writing","job":"Screenplay","credit_id":"crew-3"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(people) != 3 || len(credits) != 3 {
		t.Fatalf("people=%d credits=%d", len(people), len(credits))
	}
	if credits[1].CreditKind != "crew" || credits[1].Job != "Director" || credits[2].Job != "Screenplay" {
		t.Fatalf("crew jobs were not preserved: %#v", credits)
	}
}

func TestPeopleSearchCachesRemoteResultsForTheSpecializedQuery(t *testing.T) {
	var calls atomic.Int32
	service, _, user := peopleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/search/person" || r.URL.Query().Get("query") != "margot" || r.URL.Query().Get("language") != "es-ES" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = io.WriteString(w, `{"results":[{"id":234352,"name":"Margot Robbie","known_for_department":"Acting","popularity":75},{"id":9,"name":"Hidden","adult":true}]}`)
	}))

	for range 2 {
		response := httptest.NewRecorder()
		service.peopleSearch(response, requestWithUser(http.MethodGet, "/api/v1/people/search?q=margot&phase=remote", user.ID))
		if response.Code != http.StatusOK {
			t.Fatalf("search returned %d: %s", response.Code, response.Body.String())
		}
		var payload struct {
			Results []store.Person `json:"results"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Results) != 1 || payload.Results[0].TMDBID != 234352 {
			t.Fatalf("unexpected response: %s", response.Body.String())
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("TMDB calls = %d, want 1", calls.Load())
	}
}

func TestDiscoverMoviesByPeopleOnlyReturnsExactDirectorCredits(t *testing.T) {
	service, _, user := peopleTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/person/10/movie_credits":
			_, _ = io.WriteString(w, `{"crew":[{"id":100,"job":"Director"},{"id":200,"job":"Producer"}]}`)
		case "/discover/movie":
			if r.URL.Query().Get("with_crew") != "10" || r.URL.Query().Get("with_cast") != "20" {
				t.Fatalf("people filters were not forwarded: %s", r.URL.String())
			}
			_, _ = io.WriteString(w, `{"page":1,"total_pages":1,"total_results":2,"results":[{"id":100,"title":"Directed"},{"id":200,"title":"Only produced"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	response := httptest.NewRecorder()
	service.discoverMoviesByPeople(response, requestWithUser(http.MethodGet, "/api/v1/people/discover/movie?directors=10&with_cast=20&page=1", user.ID))
	if response.Code != http.StatusOK {
		t.Fatalf("discover returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || len(payload.Results) != 1 || payload.Results[0].ID != 100 {
		t.Fatalf("director filtering was not exact: %s", response.Body.String())
	}
}
