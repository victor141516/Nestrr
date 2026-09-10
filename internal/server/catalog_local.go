package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/victor141516/Nestrr/internal/store"
)

type localCatalogCard struct {
	ID               int     `json:"id"`
	MediaType        string  `json:"media_type"`
	Title            string  `json:"title,omitempty"`
	Name             string  `json:"name,omitempty"`
	OriginalTitle    string  `json:"original_title,omitempty"`
	OriginalName     string  `json:"original_name,omitempty"`
	OriginalLanguage string  `json:"original_language,omitempty"`
	Overview         string  `json:"overview,omitempty"`
	PosterPath       string  `json:"poster_path,omitempty"`
	BackdropPath     string  `json:"backdrop_path,omitempty"`
	ReleaseDate      string  `json:"release_date,omitempty"`
	FirstAirDate     string  `json:"first_air_date,omitempty"`
	VoteAverage      float64 `json:"vote_average,omitempty"`
	VoteCount        int     `json:"vote_count,omitempty"`
	Popularity       float64 `json:"popularity,omitempty"`
	GenreIDs         []int   `json:"genre_ids,omitempty"`
}

type localCatalogPayload struct {
	localCatalogCard
	Genres []struct {
		ID int `json:"id"`
	} `json:"genres"`
	Translations struct {
		Results []struct {
			Language string `json:"iso_639_1"`
			Data     struct {
				Title    string `json:"title"`
				Name     string `json:"name"`
				Overview string `json:"overview"`
			} `json:"data"`
		} `json:"translations"`
	} `json:"translations"`
}

func localCatalogCardFromDocument(document store.LocalCatalogDocument, language string) (localCatalogCard, error) {
	var payload localCatalogPayload
	if err := json.Unmarshal(document.Payload, &payload); err != nil {
		return localCatalogCard{}, err
	}
	payload.ID = document.TMDBID
	payload.MediaType = document.MediaType
	payload.GenreIDs = make([]int, 0, len(payload.Genres))
	for _, genre := range payload.Genres {
		if genre.ID > 0 {
			payload.GenreIDs = append(payload.GenreIDs, genre.ID)
		}
	}
	primaryLanguage := strings.ToLower(strings.Split(store.NormalizeLanguage(language), "-")[0])
	for _, translation := range payload.Translations.Results {
		if strings.ToLower(translation.Language) != primaryLanguage {
			continue
		}
		if document.MediaType == "tv" && strings.TrimSpace(translation.Data.Name) != "" {
			payload.Name = translation.Data.Name
		} else if document.MediaType == "movie" && strings.TrimSpace(translation.Data.Title) != "" {
			payload.Title = translation.Data.Title
		}
		if strings.TrimSpace(translation.Data.Overview) != "" {
			payload.Overview = translation.Data.Overview
		}
		break
	}
	return payload.localCatalogCard, nil
}

func (s *Server) localCatalogSearch(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if kind != "movie" && kind != "tv" {
		problem(w, http.StatusBadRequest, "invalid_media_type", "Media type must be movie or tv.")
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		jsonResponse(w, http.StatusOK, map[string]any{"page": 1, "total_pages": 1, "total_results": 0, "results": []localCatalogCard{}})
		return
	}
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	documents, total, err := s.Store.SearchLocalCatalog(r.Context(), kind, query, user.Language, limit)
	if err != nil {
		internal(w, err)
		return
	}
	results := make([]localCatalogCard, 0, len(documents))
	for _, document := range documents {
		card, parseErr := localCatalogCardFromDocument(document, user.Language)
		if parseErr != nil {
			s.Log.Warn("local catalog document could not be parsed", "operation", "catalog.local_search", "mediaType", kind, "tmdbId", document.TMDBID, "error", parseErr)
			continue
		}
		results = append(results, card)
	}
	jsonResponse(w, http.StatusOK, map[string]any{"page": 1, "total_pages": 1, "total_results": total, "results": results})
}
