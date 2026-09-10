package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/victor141516/Nestrr/internal/store"
)

const maxBulkImportTitles = 500

type bulkImportStatus string

const (
	bulkDatabaseSearch bulkImportStatus = "database_search"
	bulkDatabaseMatch  bulkImportStatus = "database_match"
	bulkDatabaseMiss   bulkImportStatus = "database_miss"
	bulkTMDBSearch     bulkImportStatus = "tmdb_search"
	bulkTMDBMatch      bulkImportStatus = "tmdb_match"
	bulkSaved          bulkImportStatus = "saved"
	bulkNotFound       bulkImportStatus = "not_found"
	bulkFailed         bulkImportStatus = "error"
	bulkComplete       bulkImportStatus = "complete"
)

type bulkImportEvent struct {
	Index  int                  `json:"index,omitempty"`
	Input  string               `json:"input,omitempty"`
	Status bulkImportStatus     `json:"status"`
	Source string               `json:"source,omitempty"`
	Added  bool                 `json:"added,omitempty"`
	Item   *store.MediaListItem `json:"item,omitempty"`
	Error  string               `json:"error,omitempty"`
}

type bulkMediaDocument struct {
	ID             int     `json:"id"`
	MediaType      string  `json:"media_type"`
	Title          string  `json:"title"`
	Name           string  `json:"name"`
	OriginalTitle  string  `json:"original_title"`
	OriginalName   string  `json:"original_name"`
	Overview       string  `json:"overview"`
	PosterPath     string  `json:"poster_path"`
	ReleaseDate    string  `json:"release_date"`
	FirstAirDate   string  `json:"first_air_date"`
	VoteAverage    float64 `json:"vote_average"`
	VoteCount      int     `json:"vote_count"`
	Popularity     float64 `json:"popularity"`
	Runtime        int     `json:"runtime"`
	EpisodeRuntime []int   `json:"episode_run_time"`
}

type bulkStoredCandidate struct {
	document bulkMediaDocument
	language string
}

type bulkStoredIndex map[string][]bulkStoredCandidate

var parentheticalSegment = regexp.MustCompile(`\s*\([^()]*\)`)

func normalizeBulkTitle(value string) string {
	return strings.ToLower(strings.Join(strings.FieldsFunc(strings.TrimSpace(value), unicode.IsSpace), " "))
}

func bulkTitleCandidates(value string) []string {
	candidates := []string{strings.TrimSpace(value)}
	current := candidates[0]
	for {
		matches := parentheticalSegment.FindAllStringIndex(current, -1)
		if len(matches) == 0 {
			break
		}
		match := matches[len(matches)-1]
		current = strings.TrimSpace(current[:match[0]] + " " + current[match[1]:])
		current = strings.Join(strings.Fields(current), " ")
		if current != "" {
			candidates = append(candidates, current)
		}
	}
	return candidates
}

func documentTitles(document bulkMediaDocument) []string {
	return []string{document.Title, document.Name, document.OriginalTitle, document.OriginalName}
}

func buildStoredBulkIndex(documents []store.MetadataDocument) bulkStoredIndex {
	index := bulkStoredIndex{}
	for _, record := range documents {
		var document bulkMediaDocument
		if json.Unmarshal(record.Payload, &document) != nil || document.ID <= 0 {
			continue
		}
		if document.MediaType == "" {
			document.MediaType = record.MediaType
		}
		candidate := bulkStoredCandidate{document: document, language: record.Language}
		for _, title := range documentTitles(document) {
			if normalized := normalizeBulkTitle(title); normalized != "" {
				index[normalized] = append(index[normalized], candidate)
			}
		}
	}
	return index
}

func (index bulkStoredIndex) find(input, preferredLanguage string) (bulkMediaDocument, bool) {
	for _, title := range bulkTitleCandidates(input) {
		normalized := normalizeBulkTitle(title)
		matches := append([]bulkStoredCandidate(nil), index[normalized]...)
		if len(matches) == 0 {
			continue
		}
		sort.SliceStable(matches, func(i, j int) bool {
			iPreferred := strings.EqualFold(matches[i].language, preferredLanguage)
			jPreferred := strings.EqualFold(matches[j].language, preferredLanguage)
			if iPreferred != jPreferred {
				return iPreferred
			}
			return matches[i].document.Popularity > matches[j].document.Popularity
		})
		return matches[0].document, true
	}
	return bulkMediaDocument{}, false
}

func findStoredBulkMatch(input string, documents []store.MetadataDocument, preferredLanguage string) (bulkMediaDocument, bool) {
	return buildStoredBulkIndex(documents).find(input, preferredLanguage)
}

func chooseTMDBBulkMatch(input string, body []byte) (bulkMediaDocument, bool) {
	var response struct {
		Results []bulkMediaDocument `json:"results"`
	}
	if json.Unmarshal(body, &response) != nil {
		return bulkMediaDocument{}, false
	}
	results := response.Results[:0]
	for _, result := range response.Results {
		if result.ID > 0 && (result.MediaType == "movie" || result.MediaType == "tv") {
			results = append(results, result)
		}
	}
	if len(results) == 0 {
		return bulkMediaDocument{}, false
	}
	normalized := normalizeBulkTitle(input)
	for _, result := range results {
		for _, title := range documentTitles(result) {
			if normalizeBulkTitle(title) == normalized {
				return result, true
			}
		}
	}
	return results[0], true
}

func bulkDocumentItem(document bulkMediaDocument) store.MediaListItem {
	title := document.Title
	if title == "" {
		title = document.Name
	}
	originalTitle := document.OriginalTitle
	if originalTitle == "" {
		originalTitle = document.OriginalName
	}
	releaseDate := document.ReleaseDate
	if releaseDate == "" {
		releaseDate = document.FirstAirDate
	}
	runtime := document.Runtime
	if runtime == 0 && len(document.EpisodeRuntime) > 0 {
		runtime = document.EpisodeRuntime[0]
	}
	return store.MediaListItem{
		MediaType:     document.MediaType,
		TMDBID:        document.ID,
		Title:         title,
		OriginalTitle: originalTitle,
		Overview:      document.Overview,
		PosterPath:    document.PosterPath,
		ReleaseDate:   releaseDate,
		VoteAverage:   document.VoteAverage,
		VoteCount:     document.VoteCount,
		Popularity:    document.Popularity,
		Runtime:       runtime,
	}
}

func (s *Server) resolveTMDBBulkItem(ctx context.Context, user store.User, input string) (store.MediaListItem, bool, error) {
	if s.TMDB == nil {
		return store.MediaListItem{}, false, errors.New("TMDB is unavailable")
	}
	response, err := s.TMDB.Get(ctx, "/search/multi", url.Values{
		"query":    {input},
		"language": {user.Language},
	}, false)
	if err != nil {
		return store.MediaListItem{}, false, err
	}
	match, found := chooseTMDBBulkMatch(input, response.Body)
	if !found {
		return store.MediaListItem{}, false, nil
	}

	languages := []string{user.Language}
	if !strings.EqualFold(user.Language, "en-US") {
		languages = append(languages, "en-US")
	}
	var localized bulkMediaDocument
	for _, language := range languages {
		detail, detailErr := s.TMDB.Get(ctx, fmt.Sprintf("/%s/%d", match.MediaType, match.ID), url.Values{"language": {language}}, false)
		if detailErr != nil {
			if strings.EqualFold(language, user.Language) {
				return store.MediaListItem{}, false, detailErr
			}
			continue
		}
		if err := s.Store.PutMetadata(ctx, match.MediaType, match.ID, language, detail.Body); err != nil {
			return store.MediaListItem{}, false, err
		}
		var document bulkMediaDocument
		if json.Unmarshal(detail.Body, &document) == nil {
			document.MediaType = match.MediaType
			if strings.EqualFold(language, user.Language) {
				localized = document
			}
		}
	}
	if localized.ID <= 0 {
		localized = match
	}
	return bulkDocumentItem(localized), true, nil
}

func (s *Server) addBulkItem(ctx context.Context, userID, listID int64, item store.MediaListItem) (bool, store.MediaListItem, error) {
	existing, err := s.Store.MediaListContains(ctx, userID, listID, item.MediaType, item.TMDBID)
	if err != nil {
		return false, item, err
	}
	if existing {
		item.AddedAt, err = s.Store.MediaListItemAddedAt(ctx, userID, listID, item.MediaType, item.TMDBID)
		if err != nil {
			return false, item, err
		}
	} else {
		item.AddedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if err := s.Store.AddMediaListItem(ctx, userID, listID, item); err != nil {
		return false, item, err
	}
	return !existing, item, nil
}

func writeBulkImportEvent(w http.ResponseWriter, flusher http.Flusher, event bulkImportEvent) error {
	if err := json.NewEncoder(w).Encode(event); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func (s *Server) bulkImportMediaList(w http.ResponseWriter, r *http.Request) {
	listID, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	if _, err := s.Store.MediaListByID(r.Context(), claims(r).UserID, listID); errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_not_found", "The list does not exist.")
		return
	} else if err != nil {
		internal(w, err)
		return
	}
	var input struct {
		Titles string `json:"titles"`
	}
	if !decode(w, r, &input) {
		return
	}
	titles := []string{}
	for _, line := range strings.Split(strings.ReplaceAll(input.Titles, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			titles = append(titles, line)
		}
	}
	if len(titles) == 0 || len(titles) > maxBulkImportTitles {
		problem(w, 400, "invalid_bulk_import", "Provide between 1 and 500 titles.")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "streaming_unavailable", "Streaming is unavailable.")
		return
	}
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	documents, err := s.Store.MediaMetadata(r.Context(), user.Language, "en-US")
	if err != nil {
		internal(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	metadataIndex := buildStoredBulkIndex(documents)
	for index, title := range titles {
		if r.Context().Err() != nil {
			return
		}
		base := bulkImportEvent{Index: index, Input: title}
		base.Status = bulkDatabaseSearch
		if writeBulkImportEvent(w, flusher, base) != nil {
			return
		}
		document, found := metadataIndex.find(title, user.Language)
		var item store.MediaListItem
		source := "database"
		if found {
			base.Status, base.Source = bulkDatabaseMatch, source
			if writeBulkImportEvent(w, flusher, base) != nil {
				return
			}
			item = bulkDocumentItem(document)
		} else {
			base.Status = bulkDatabaseMiss
			if writeBulkImportEvent(w, flusher, base) != nil {
				return
			}
			base.Status = bulkTMDBSearch
			if writeBulkImportEvent(w, flusher, base) != nil {
				return
			}
			source = "tmdb"
			var resolveErr error
			item, found, resolveErr = s.resolveTMDBBulkItem(r.Context(), user, title)
			if resolveErr != nil {
				base.Status, base.Source, base.Error = bulkFailed, source, resolveErr.Error()
				if writeBulkImportEvent(w, flusher, base) != nil {
					return
				}
				continue
			}
			if !found {
				base.Status, base.Source = bulkNotFound, source
				if writeBulkImportEvent(w, flusher, base) != nil {
					return
				}
				continue
			}
			base.Status, base.Source, base.Item = bulkTMDBMatch, source, &item
			if writeBulkImportEvent(w, flusher, base) != nil {
				return
			}
		}
		added, saved, saveErr := s.addBulkItem(r.Context(), user.ID, listID, item)
		if saveErr != nil {
			base.Status, base.Source, base.Error, base.Item = bulkFailed, source, saveErr.Error(), nil
			if writeBulkImportEvent(w, flusher, base) != nil {
				return
			}
			continue
		}
		base.Status, base.Source, base.Added, base.Item = bulkSaved, source, added, &saved
		if writeBulkImportEvent(w, flusher, base) != nil {
			return
		}
		if source == "tmdb" {
			if refreshed, refreshErr := s.Store.MediaMetadata(r.Context(), user.Language, "en-US"); refreshErr == nil {
				documents = refreshed
				metadataIndex = buildStoredBulkIndex(documents)
			}
		}
	}
	_ = writeBulkImportEvent(w, flusher, bulkImportEvent{Status: bulkComplete})
}
