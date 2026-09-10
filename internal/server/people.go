package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

const personSearchTTL = 7 * 24 * time.Hour

type tmdbPerson struct {
	ID                 int     `json:"id"`
	Name               string  `json:"name"`
	ProfilePath        string  `json:"profile_path"`
	KnownForDepartment string  `json:"known_for_department"`
	Popularity         float64 `json:"popularity"`
	Adult              bool    `json:"adult"`
	CreditID           string  `json:"credit_id"`
	Department         string  `json:"department"`
	Job                string  `json:"job"`
	Character          string  `json:"character"`
	Order              int     `json:"order"`
}

type tmdbCredits struct {
	Cast []tmdbPerson `json:"cast"`
	Crew []tmdbPerson `json:"crew"`
}

func storedPerson(person tmdbPerson) store.Person {
	return store.Person{
		TMDBID:             person.ID,
		Name:               strings.TrimSpace(person.Name),
		ProfilePath:        strings.TrimSpace(person.ProfilePath),
		KnownForDepartment: strings.TrimSpace(person.KnownForDepartment),
		Popularity:         person.Popularity,
		Adult:              person.Adult,
	}
}

func creditKey(kind string, person tmdbPerson) string {
	if strings.TrimSpace(person.CreditID) != "" {
		return person.CreditID
	}
	return fmt.Sprintf("%s:%d:%s:%s:%d", kind, person.ID, person.Job, person.Character, person.Order)
}

func parseMediaCredits(mediaType string, mediaTMDBID int, payload []byte) ([]store.Person, []store.MediaCredit, error) {
	var document tmdbCredits
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, nil, err
	}
	peopleByID := map[int]store.Person{}
	credits := make([]store.MediaCredit, 0, len(document.Cast)+len(document.Crew))
	appendCredit := func(kind string, person tmdbPerson) {
		if person.ID <= 0 || strings.TrimSpace(person.Name) == "" {
			return
		}
		knownFor := person.KnownForDepartment
		if knownFor == "" {
			if kind == "cast" {
				knownFor = "Acting"
			} else {
				knownFor = person.Department
			}
		}
		person.KnownForDepartment = knownFor
		peopleByID[person.ID] = storedPerson(person)
		credits = append(credits, store.MediaCredit{
			MediaType:    mediaType,
			MediaTMDBID:  mediaTMDBID,
			CreditKey:    creditKey(kind, person),
			PersonTMDBID: person.ID,
			CreditKind:   kind,
			Department:   person.Department,
			Job:          person.Job,
			Character:    person.Character,
			Order:        person.Order,
		})
	}
	for _, person := range document.Cast {
		appendCredit("cast", person)
	}
	for _, person := range document.Crew {
		appendCredit("crew", person)
	}
	people := make([]store.Person, 0, len(peopleByID))
	for _, person := range peopleByID {
		people = append(people, person)
	}
	return people, credits, nil
}

func (s *Server) storeMediaCredits(ctx context.Context, mediaType string, mediaTMDBID int, payload []byte) error {
	people, credits, err := parseMediaCredits(mediaType, mediaTMDBID, payload)
	if err != nil {
		return err
	}
	return s.Store.ReplaceMediaCredits(ctx, mediaType, mediaTMDBID, people, credits)
}

func parsePositiveIDs(raw string, maximum int) ([]int, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	seen := map[int]bool{}
	ids := []int{}
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("identifiers must be positive integers")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) > maximum {
		return nil, fmt.Errorf("at most %d identifiers are allowed", maximum)
	}
	return ids, nil
}

func (s *Server) peopleSearch(w http.ResponseWriter, r *http.Request) {
	if rawIDs := strings.TrimSpace(r.URL.Query().Get("ids")); rawIDs != "" {
		ids, err := parsePositiveIDs(rawIDs, 50)
		if err != nil {
			problem(w, http.StatusBadRequest, "invalid_person_ids", err.Error())
			return
		}
		people, err := s.Store.PeopleByIDs(r.Context(), ids)
		if err != nil {
			internal(w, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": people})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) < 2 {
		problem(w, http.StatusBadRequest, "person_query_too_short", "Enter at least two characters.")
		return
	}
	limit := 12
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed > 0 {
		limit = min(parsed, 30)
	}
	phase := r.URL.Query().Get("phase")
	if phase == "" || phase == "local" {
		people, err := s.Store.SearchPeople(r.Context(), query, limit)
		if err != nil {
			internal(w, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": people, "phase": "local"})
		return
	}
	if phase != "remote" {
		problem(w, http.StatusBadRequest, "invalid_person_phase", "Person search phase must be local or remote.")
		return
	}

	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	if cached, fresh, err := s.Store.CachedPersonSearch(r.Context(), query, user.Language, personSearchTTL); err != nil {
		internal(w, err)
		return
	} else if fresh {
		if len(cached) > limit {
			cached = cached[:limit]
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": cached, "phase": "remote", "cached": true})
		return
	}
	response, err := s.TMDB.Get(r.Context(), "/search/person", url.Values{
		"query":         {query},
		"language":      {user.Language},
		"include_adult": {"false"},
		"page":          {"1"},
	}, false)
	if err != nil {
		problem(w, http.StatusBadGateway, "person_search_failed", "People could not be searched.")
		return
	}
	var document struct {
		Results []tmdbPerson `json:"results"`
	}
	if err := json.Unmarshal(response.Body, &document); err != nil {
		internal(w, err)
		return
	}
	people := make([]store.Person, 0, len(document.Results))
	for _, result := range document.Results {
		person := storedPerson(result)
		if person.TMDBID > 0 && person.Name != "" && !person.Adult {
			people = append(people, person)
		}
	}
	if err := s.Store.PutPersonSearch(r.Context(), query, user.Language, people); err != nil {
		internal(w, err)
		return
	}
	if len(people) > limit {
		people = people[:limit]
	}
	jsonResponse(w, http.StatusOK, map[string]any{"results": people, "phase": "remote", "cached": false})
}

func (s *Server) localMediaByPeople(w http.ResponseWriter, r *http.Request) {
	castIDs, castErr := parsePositiveIDs(r.URL.Query().Get("cast"), 8)
	directorIDs, directorErr := parsePositiveIDs(r.URL.Query().Get("directors"), 8)
	if castErr != nil || directorErr != nil {
		problem(w, http.StatusBadRequest, "invalid_people_filter", "People filters must contain positive identifiers.")
		return
	}
	if len(castIDs) == 0 && len(directorIDs) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"results": []any{}})
		return
	}
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	results, err := s.Store.LocalMediaForPeople(r.Context(), user.Language, "movie", castIDs, directorIDs)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) directorMovieSets(ctx context.Context, directorIDs []int, language string) ([]map[int]bool, error) {
	sets := make([]map[int]bool, 0, len(directorIDs))
	for _, directorID := range directorIDs {
		response, err := s.TMDB.Get(ctx, fmt.Sprintf("/person/%d/movie_credits", directorID), url.Values{"language": {language}}, false)
		if err != nil {
			return nil, err
		}
		var document struct {
			Crew []struct {
				ID  int    `json:"id"`
				Job string `json:"job"`
			} `json:"crew"`
		}
		if err := json.Unmarshal(response.Body, &document); err != nil {
			return nil, err
		}
		set := map[int]bool{}
		for _, credit := range document.Crew {
			if credit.ID > 0 && credit.Job == "Director" {
				set[credit.ID] = true
			}
		}
		sets = append(sets, set)
	}
	return sets, nil
}

func movieInEverySet(movieID int, sets []map[int]bool) bool {
	for _, set := range sets {
		if !set[movieID] {
			return false
		}
	}
	return true
}

func (s *Server) discoverMoviesByPeople(w http.ResponseWriter, r *http.Request) {
	directorIDs, err := parsePositiveIDs(r.URL.Query().Get("directors"), 8)
	if err != nil || len(directorIDs) == 0 {
		problem(w, http.StatusBadRequest, "invalid_directors", "At least one valid director is required.")
		return
	}
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	requestedPage, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if requestedPage < 1 {
		requestedPage = 1
	}
	query := r.URL.Query()
	query.Del("directors")
	query.Del("page")
	if query.Get("language") == "" {
		query.Set("language", user.Language)
	}
	directorStrings := make([]string, len(directorIDs))
	for index, id := range directorIDs {
		directorStrings[index] = strconv.Itoa(id)
	}
	query.Set("with_crew", strings.Join(directorStrings, ","))
	sets, err := s.directorMovieSets(r.Context(), directorIDs, user.Language)
	if err != nil {
		problem(w, http.StatusBadGateway, "director_credits_failed", "Director credits could not be loaded.")
		return
	}

	const pageSize = 20
	targetEnd := requestedPage * pageSize
	matches := []json.RawMessage{}
	rawPage, rawTotalPages := 1, 1
	for rawPage <= rawTotalPages && rawPage <= 50 && len(matches) <= targetEnd {
		query.Set("page", strconv.Itoa(rawPage))
		response, requestErr := s.TMDB.Get(r.Context(), "/discover/movie", query, false)
		if requestErr != nil {
			problem(w, http.StatusBadGateway, "movie_discover_failed", "Movies could not be loaded.")
			return
		}
		var document struct {
			TotalPages int               `json:"total_pages"`
			Results    []json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(response.Body, &document); err != nil {
			internal(w, err)
			return
		}
		rawTotalPages = document.TotalPages
		for _, raw := range document.Results {
			var identity struct {
				ID int `json:"id"`
			}
			if json.Unmarshal(raw, &identity) == nil && movieInEverySet(identity.ID, sets) {
				matches = append(matches, raw)
			}
		}
		rawPage++
	}
	start := (requestedPage - 1) * pageSize
	if start > len(matches) {
		start = len(matches)
	}
	end := min(start+pageSize, len(matches))
	results := matches[start:end]
	hasMore := len(matches) > end || rawPage <= rawTotalPages
	totalPages := requestedPage
	if hasMore {
		totalPages++
	}
	totalResults := len(matches)
	if hasMore && totalResults <= end {
		totalResults = end + 1
	}
	payload, err := json.Marshal(map[string]any{
		"page":          requestedPage,
		"total_pages":   totalPages,
		"total_results": totalResults,
		"results":       results,
	})
	if err != nil {
		internal(w, err)
		return
	}
	jsonBytes(w, http.StatusOK, "application/json", s.enrichCatalogRatings(r.Context(), "discover/movie", payload, false))
}
