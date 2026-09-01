package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/victor141516/Nestrr/internal/ratings"
	"github.com/victor141516/Nestrr/internal/store"
)

type ratingDocument struct {
	MediaType string
	Payload   *json.RawMessage
}

func catalogRatingMaps(path string, document map[string]any) (string, []map[string]any) {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return "", nil
	}
	mediaType := ""
	container := ""
	switch parts[0] {
	case "search", "discover":
		if len(parts) >= 2 {
			mediaType, container = parts[1], "results"
		}
	case "movie", "tv":
		mediaType = parts[0]
		if len(parts) == 2 {
			return mediaType, []map[string]any{document}
		}
		if len(parts) == 3 && parts[2] == "recommendations" {
			container = "results"
		}
	case "collection":
		mediaType = "movie"
		if len(parts) == 2 {
			container = "parts"
		} else if len(parts) == 3 && parts[2] == "recommendations" {
			container = "results"
		}
	}
	if mediaType != "movie" && mediaType != "tv" || container == "" {
		return "", nil
	}
	rawItems, ok := document[container].([]any)
	if !ok {
		return mediaType, nil
	}
	items := make([]map[string]any, 0, len(rawItems))
	for _, raw := range rawItems {
		if item, ok := raw.(map[string]any); ok {
			items = append(items, item)
		}
	}
	return mediaType, items
}

func referenceFromMap(mediaType string, item map[string]any) (ratings.Reference, bool) {
	id := 0
	switch value := item["id"].(type) {
	case float64:
		id = int(value)
	case json.Number:
		id, _ = strconv.Atoi(value.String())
	}
	imdbID, _ := item["imdb_id"].(string)
	if imdbID == "" {
		if externalIDs, ok := item["external_ids"].(map[string]any); ok {
			imdbID, _ = externalIDs["imdb_id"].(string)
		}
	}
	return ratings.Reference{MediaType: mediaType, TMDBID: id, IMDbID: imdbID}, id > 0
}

func (s *Server) enrichCatalogRatings(ctx context.Context, path string, payload []byte, force bool) []byte {
	if s.Ratings == nil || !s.Ratings.Enabled() || len(payload) == 0 {
		return payload
	}
	var document map[string]any
	if json.Unmarshal(payload, &document) != nil {
		return payload
	}
	mediaType, items := catalogRatingMaps(path, document)
	if len(items) == 0 {
		return payload
	}
	references := make([]ratings.Reference, 0, len(items))
	for _, item := range items {
		if reference, ok := referenceFromMap(mediaType, item); ok {
			if reference.MediaType == "tv" && reference.IMDbID == "" && s.Ratings.OMDbEnabled() && s.TMDB != nil && strings.Count(path, "/") == 1 {
				if response, err := s.TMDB.Get(ctx, fmt.Sprintf("/tv/%d/external_ids", reference.TMDBID), nil, force); err == nil {
					var externalIDs struct {
						IMDbID string `json:"imdb_id"`
					}
					if json.Unmarshal(response.Body, &externalIDs) == nil && externalIDs.IMDbID != "" {
						reference.IMDbID = externalIDs.IMDbID
						item["imdb_id"] = externalIDs.IMDbID
					}
				}
			}
			references = append(references, reference)
		}
	}
	resolved := s.Ratings.Resolve(ctx, references, force)
	for _, item := range items {
		reference, ok := referenceFromMap(mediaType, item)
		if !ok {
			continue
		}
		if rating, found := resolved[mediaType+":"+strconv.Itoa(reference.TMDBID)]; found && rating.Found {
			item["rotten_tomatoes_score"] = rating.RottenTomatoesScore
		}
	}
	enriched, err := json.Marshal(document)
	if err != nil {
		return payload
	}
	return enriched
}

func (s *Server) enrichRatingDocuments(ctx context.Context, documents []ratingDocument, force bool) {
	if s.Ratings == nil || !s.Ratings.Enabled() {
		return
	}
	type parsedDocument struct {
		mediaType string
		payload   *json.RawMessage
		item      map[string]any
	}
	parsed := make([]parsedDocument, 0, len(documents))
	references := make([]ratings.Reference, 0, len(documents))
	for _, document := range documents {
		if document.Payload == nil || len(*document.Payload) == 0 {
			continue
		}
		var item map[string]any
		if json.Unmarshal(*document.Payload, &item) != nil {
			continue
		}
		reference, ok := referenceFromMap(document.MediaType, item)
		if !ok {
			continue
		}
		parsed = append(parsed, parsedDocument{mediaType: document.MediaType, payload: document.Payload, item: item})
		references = append(references, reference)
	}
	resolved := s.Ratings.Resolve(ctx, references, force)
	for _, document := range parsed {
		reference, _ := referenceFromMap(document.mediaType, document.item)
		if rating, ok := resolved[document.mediaType+":"+strconv.Itoa(reference.TMDBID)]; ok && rating.Found {
			document.item["rotten_tomatoes_score"] = rating.RottenTomatoesScore
		}
		if encoded, err := json.Marshal(document.item); err == nil {
			*document.payload = encoded
		}
	}
}

func (s *Server) enrichMediaListRatings(ctx context.Context, items []store.MediaListItem) {
	if s.Ratings == nil || !s.Ratings.Enabled() || len(items) == 0 {
		return
	}
	references := make([]ratings.Reference, 0, len(items))
	for _, item := range items {
		references = append(references, ratings.Reference{MediaType: item.MediaType, TMDBID: item.TMDBID})
	}
	resolved := s.Ratings.Resolve(ctx, references, false)
	for index := range items {
		lookupKey := items[index].MediaType + ":" + fmt.Sprint(items[index].TMDBID)
		if rating, ok := resolved[lookupKey]; ok && rating.Found {
			score := rating.RottenTomatoesScore
			items[index].RottenTomatoesScore = &score
		}
	}
}
