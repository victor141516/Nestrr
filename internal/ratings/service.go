package ratings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

// MDBList allows 10 IDs per rating request for free accounts (100 for
// supporters). Using the free-tier limit keeps batch enrichment portable.
const maxMDBListBatch = 10

type Reference struct {
	MediaType string
	TMDBID    int
	IMDbID    string
}

type Service struct {
	MDBListBaseURL string
	MDBListAPIKey  string
	OMDbBaseURL    string
	OMDbAPIKey     string
	HTTP           *http.Client
	Store          *store.Store
	Log            *slog.Logger
}

func (s *Service) Enabled() bool {
	return s != nil && (s.MDBListAPIKey != "" || s.OMDbAPIKey != "")
}

func (s *Service) OMDbEnabled() bool {
	return s != nil && s.OMDbAPIKey != ""
}

func key(mediaType string, tmdbID int) string {
	return mediaType + ":" + strconv.Itoa(tmdbID)
}

func canonicalReference(reference Reference) (Reference, bool) {
	reference.MediaType = strings.ToLower(strings.TrimSpace(reference.MediaType))
	if reference.MediaType == "show" {
		reference.MediaType = "tv"
	}
	reference.IMDbID = strings.TrimSpace(reference.IMDbID)
	return reference, (reference.MediaType == "movie" || reference.MediaType == "tv") && reference.TMDBID > 0
}

// Resolve returns every cached or newly resolved rating, including negative
// cache entries. A force refresh only replaces an existing value after at least
// one provider completed a valid lookup.
func (s *Service) Resolve(ctx context.Context, references []Reference, force bool) map[string]store.ExternalRating {
	resolved := map[string]store.ExternalRating{}
	if !s.Enabled() || s.Store == nil || s.HTTP == nil {
		return resolved
	}

	unique := map[string]Reference{}
	previous := map[string]store.ExternalRating{}
	for _, raw := range references {
		reference, ok := canonicalReference(raw)
		if !ok {
			continue
		}
		lookupKey := key(reference.MediaType, reference.TMDBID)
		if existing, found := unique[lookupKey]; found {
			if existing.IMDbID == "" && reference.IMDbID != "" {
				unique[lookupKey] = reference
			}
			continue
		}
		unique[lookupKey] = reference
		if cached, err := s.Store.GetExternalRating(ctx, reference.MediaType, reference.TMDBID); err == nil {
			previous[lookupKey] = cached
			if !force {
				resolved[lookupKey] = cached
			}
		}
	}

	pending := make([]Reference, 0, len(unique))
	for lookupKey, reference := range unique {
		if _, ok := resolved[lookupKey]; !ok {
			pending = append(pending, reference)
		}
	}
	if len(pending) == 0 {
		return resolved
	}

	checked := map[string]string{}
	if s.MDBListAPIKey != "" {
		for _, mediaType := range []string{"movie", "tv"} {
			group := make([]Reference, 0, len(pending))
			for _, reference := range pending {
				if reference.MediaType == mediaType {
					group = append(group, reference)
				}
			}
			for start := 0; start < len(group); start += maxMDBListBatch {
				end := min(start+maxMDBListBatch, len(group))
				batch := group[start:end]
				scores, err := s.resolveMDBList(ctx, mediaType, batch)
				if err != nil {
					s.warn("MDBList rating lookup failed", "mdblist", mediaType, err)
					continue
				}
				for _, reference := range batch {
					lookupKey := key(reference.MediaType, reference.TMDBID)
					checked[lookupKey] = "mdblist"
					if score, found := scores[reference.TMDBID]; found {
						rating := newRating(reference, score, true, "mdblist")
						s.persist(ctx, rating)
						resolved[lookupKey] = rating
					}
				}
			}
		}
	}

	if s.OMDbAPIKey != "" {
		for _, reference := range pending {
			lookupKey := key(reference.MediaType, reference.TMDBID)
			if _, ok := resolved[lookupKey]; ok || reference.IMDbID == "" {
				continue
			}
			score, found, valid, err := s.resolveOMDb(ctx, reference.IMDbID)
			if err != nil {
				s.warn("OMDb rating lookup failed", "omdb", reference.MediaType, err)
				continue
			}
			if !valid {
				continue
			}
			checked[lookupKey] = "omdb"
			if found {
				rating := newRating(reference, score, true, "omdb")
				s.persist(ctx, rating)
				resolved[lookupKey] = rating
			}
		}
	}

	for _, reference := range pending {
		lookupKey := key(reference.MediaType, reference.TMDBID)
		if _, ok := resolved[lookupKey]; ok {
			continue
		}
		if provider := checked[lookupKey]; provider != "" {
			rating := newRating(reference, 0, false, provider)
			s.persist(ctx, rating)
			resolved[lookupKey] = rating
			continue
		}
		if cached, ok := previous[lookupKey]; ok {
			resolved[lookupKey] = cached
		}
	}
	return resolved
}

func newRating(reference Reference, score int, found bool, provider string) store.ExternalRating {
	return store.ExternalRating{
		MediaType:           reference.MediaType,
		TMDBID:              reference.TMDBID,
		RottenTomatoesScore: score,
		Found:               found,
		Provider:            provider,
		FetchedAt:           time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (s *Service) persist(ctx context.Context, rating store.ExternalRating) {
	if err := s.Store.PutExternalRating(ctx, rating); err != nil {
		s.warn("rating cache write failed", rating.Provider, rating.MediaType, err)
	}
}

func (s *Service) warn(message, provider, mediaType string, err error) {
	if s.Log != nil {
		s.Log.Warn(message, "operation", "ratings.resolve", "provider", provider, "mediaType", mediaType, "error", err)
	}
}

func (s *Service) resolveMDBList(ctx context.Context, mediaType string, references []Reference) (map[int]int, error) {
	providerType := mediaType
	if providerType == "tv" {
		providerType = "show"
	}
	endpoint, err := url.Parse(s.MDBListBaseURL + "/rating/" + providerType + "/tomatoes")
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("apikey", s.MDBListAPIKey)
	endpoint.RawQuery = query.Encode()
	ids := make([]int, 0, len(references))
	providerIDs := make([]string, 0, len(references))
	for _, reference := range references {
		ids = append(ids, reference.TMDBID)
		providerIDs = append(providerIDs, strconv.Itoa(reference.TMDBID))
	}
	payload, _ := json.Marshal(map[string]any{"ids": providerIDs, "provider": "tmdb"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	res, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", res.StatusCode)
	}
	var document any
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, err
	}
	scores := map[int]int{}
	switch response := document.(type) {
	case map[string]any:
		// The ratings endpoint returns one envelope describing the requested
		// provider and an ID/rating pair for every resolved title.
		ratings, ok := response["ratings"].([]any)
		if !ok && response["ratings"] != nil {
			return nil, errors.New("invalid MDBList ratings response")
		}
		for _, raw := range ratings {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			tmdbID := integerField(item, "id", "provider_id", "tmdb_id", "tmdb")
			if score, found := percentage(item["rating"]); tmdbID > 0 && found {
				scores[tmdbID] = score
			}
		}
	case []any:
		// Retain compatibility with the full media batch response.
		for index, raw := range response {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			tmdbID := integerField(item, "provider_id", "tmdb_id", "tmdbid", "tmdb")
			if tmdbID == 0 {
				if idsMap, ok := item["ids"].(map[string]any); ok {
					tmdbID = integerField(idsMap, "tmdb", "tmdb_id")
				}
			}
			if tmdbID == 0 && len(response) == len(ids) && index < len(ids) {
				tmdbID = ids[index]
			}
			if tmdbID == 0 {
				continue
			}
			if score, found := percentage(item["provider_rating"]); found {
				scores[tmdbID] = score
				continue
			}
			if score, found := tomatoesScore(item["ratings"]); found {
				scores[tmdbID] = score
			}
		}
	default:
		return nil, errors.New("invalid MDBList response")
	}
	return scores, nil
}

func integerField(item map[string]any, names ...string) int {
	for _, name := range names {
		switch value := item[name].(type) {
		case float64:
			return int(value)
		case json.Number:
			parsed, _ := strconv.Atoi(value.String())
			return parsed
		case string:
			parsed, _ := strconv.Atoi(value)
			return parsed
		}
	}
	return 0
}

func tomatoesScore(value any) (int, bool) {
	ratings, ok := value.([]any)
	if !ok {
		return 0, false
	}
	for _, raw := range ratings {
		rating, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		source := normalizedSource(fmt.Sprint(rating["source"]))
		if strings.Contains(source, "audience") || strings.Contains(source, "popcorn") {
			continue
		}
		if source != "tomatoes" && source != "rottentomatoes" && source != "rtomatoes" && source != "tomatometer" {
			continue
		}
		for _, field := range []string{"score", "value", "rating"} {
			if score, valid := percentage(rating[field]); valid {
				return score, true
			}
		}
	}
	return 0, false
}

func normalizedSource(source string) string {
	return strings.NewReplacer(" ", "", "_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(source)))
}

func percentage(value any) (int, bool) {
	var number float64
	switch typed := value.(type) {
	case float64:
		number = typed
	case json.Number:
		parsed, err := strconv.ParseFloat(typed.String(), 64)
		if err != nil {
			return 0, false
		}
		number = parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(typed, "%")), 64)
		if err != nil {
			return 0, false
		}
		number = parsed
	default:
		return 0, false
	}
	if number < 0 || number > 100 {
		return 0, false
	}
	return int(number + 0.5), true
}

func (s *Service) resolveOMDb(ctx context.Context, imdbID string) (score int, found bool, valid bool, err error) {
	endpoint, err := url.Parse(s.OMDbBaseURL + "/")
	if err != nil {
		return 0, false, false, err
	}
	query := endpoint.Query()
	query.Set("apikey", s.OMDbAPIKey)
	query.Set("i", imdbID)
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, false, false, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := s.HTTP.Do(req)
	if err != nil {
		return 0, false, false, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return 0, false, false, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return 0, false, false, fmt.Errorf("status %d", res.StatusCode)
	}
	var doc struct {
		Response string `json:"Response"`
		Error    string `json:"Error"`
		Ratings  []struct {
			Source string `json:"Source"`
			Value  string `json:"Value"`
		} `json:"Ratings"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return 0, false, false, err
	}
	if strings.EqualFold(doc.Response, "False") {
		if strings.Contains(strings.ToLower(doc.Error), "not found") {
			return 0, false, true, nil
		}
		return 0, false, false, errors.New(doc.Error)
	}
	for _, rating := range doc.Ratings {
		if normalizedSource(rating.Source) != "rottentomatoes" {
			continue
		}
		if score, ok := percentage(rating.Value); ok {
			return score, true, true, nil
		}
	}
	return 0, false, true, nil
}
