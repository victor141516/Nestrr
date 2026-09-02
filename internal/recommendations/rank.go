package recommendations

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/victor141516/Nestrr/internal/store"
)

type Recommendation struct {
	ID            int     `json:"id"`
	MediaType     string  `json:"media_type"`
	Title         string  `json:"title,omitempty"`
	Name          string  `json:"name,omitempty"`
	OriginalTitle string  `json:"original_title,omitempty"`
	OriginalName  string  `json:"original_name,omitempty"`
	Overview      string  `json:"overview"`
	PosterPath    string  `json:"poster_path"`
	BackdropPath  string  `json:"backdrop_path"`
	ReleaseDate   string  `json:"release_date,omitempty"`
	FirstAirDate  string  `json:"first_air_date,omitempty"`
	VoteAverage   float64 `json:"vote_average"`
	VoteCount     int     `json:"vote_count"`
	Popularity    float64 `json:"popularity"`
	GenreIDs      []int   `json:"genre_ids"`
	Score         float64 `json:"recommendation_score"`
}

type weightedFeatures struct {
	Genres, Keywords, Companies, Cast, Directors map[int]float64
}

type rankedMedia struct {
	media    store.CatalogMedia
	features store.CatalogFeatures
	tokens   map[uint64]struct{}
	score    float64
}

func (s *Service) Recommend(ctx context.Context, userID int64, language, mediaType string, limit int) ([]Recommendation, error) {
	if mediaType != "movie" && mediaType != "tv" {
		mediaType = "movie"
	}
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	seeds, err := s.Store.RecommendationSeeds(ctx, userID)
	if err != nil {
		return nil, err
	}
	excluded, err := s.Store.RecommendationExclusions(ctx, userID)
	if err != nil {
		return nil, err
	}
	catalog, err := s.Store.CatalogMediaRankingEntries(ctx)
	if err != nil {
		return nil, err
	}
	byKey := make(map[store.CatalogKey]store.CatalogMedia, len(catalog))
	for _, media := range catalog {
		byKey[media.CatalogKey] = media
	}
	preferences := weightedFeatures{Genres: map[int]float64{}, Keywords: map[int]float64{}, Companies: map[int]float64{}, Cast: map[int]float64{}, Directors: map[int]float64{}}
	for _, seed := range seeds {
		media, ok := byKey[seed.CatalogKey]
		if !ok {
			continue
		}
		var features store.CatalogFeatures
		if json.Unmarshal(media.Features, &features) != nil {
			continue
		}
		addWeighted(preferences.Genres, features.Genres, seed.Weight)
		addWeighted(preferences.Keywords, features.Keywords, seed.Weight)
		addWeighted(preferences.Companies, features.Companies, seed.Weight)
		addWeighted(preferences.Cast, features.Cast, seed.Weight)
		addWeighted(preferences.Directors, features.Directors, seed.Weight)
	}

	documents := 0
	documentFrequency := weightedFeatures{Genres: map[int]float64{}, Keywords: map[int]float64{}, Companies: map[int]float64{}, Cast: map[int]float64{}, Directors: map[int]float64{}}
	parsed := make(map[store.CatalogKey]store.CatalogFeatures, len(catalog))
	for _, media := range catalog {
		if media.MediaType != mediaType {
			continue
		}
		var features store.CatalogFeatures
		if json.Unmarshal(media.Features, &features) != nil {
			continue
		}
		parsed[media.CatalogKey] = features
		documents++
		addDocumentFrequency(documentFrequency.Genres, features.Genres)
		addDocumentFrequency(documentFrequency.Keywords, features.Keywords)
		addDocumentFrequency(documentFrequency.Companies, features.Companies)
		addDocumentFrequency(documentFrequency.Cast, features.Cast)
		addDocumentFrequency(documentFrequency.Directors, features.Directors)
	}

	candidates := []rankedMedia{}
	for _, media := range catalog {
		if media.MediaType != mediaType || excluded[media.CatalogKey] {
			continue
		}
		features, ok := parsed[media.CatalogKey]
		if !ok {
			continue
		}
		score := featureScore(preferences.Genres, documentFrequency.Genres, features.Genres, documents, 1.0)
		score += featureScore(preferences.Keywords, documentFrequency.Keywords, features.Keywords, documents, 1.3)
		score += featureScore(preferences.Companies, documentFrequency.Companies, features.Companies, documents, 0.8)
		score += featureScore(preferences.Cast, documentFrequency.Cast, features.Cast, documents, 0.7)
		score += featureScore(preferences.Directors, documentFrequency.Directors, features.Directors, documents, 1.4)
		if score <= 0 {
			continue
		}
		quality := math.Max(0, math.Min(1, media.VoteAverage/10)) * math.Log10(float64(media.VoteCount)+1)
		score += quality * 0.35
		candidates = append(candidates, rankedMedia{media: media, features: features, tokens: featureTokens(features), score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].media.Popularity > candidates[j].media.Popularity
		}
		return candidates[i].score > candidates[j].score
	})
	selected := diversify(candidates, limit)
	result := make([]Recommendation, 0, len(selected))
	for _, candidate := range selected {
		localizedMedia := candidate.media
		if primaryLanguage(language) != "en" {
			if complete, completeErr := s.Store.CatalogMediaByKey(ctx, candidate.media.CatalogKey); completeErr == nil {
				localizedMedia.Payload = complete.Payload
			}
		}
		title, overview := localizedText(localizedMedia, language)
		item := Recommendation{ID: candidate.media.TMDBID, MediaType: mediaType, Overview: overview, PosterPath: candidate.media.PosterPath, BackdropPath: candidate.media.BackdropPath, VoteAverage: candidate.media.VoteAverage, VoteCount: candidate.media.VoteCount, Popularity: candidate.media.Popularity, GenreIDs: candidate.features.Genres, Score: candidate.score}
		if mediaType == "movie" {
			item.Title, item.OriginalTitle, item.ReleaseDate = title, candidate.media.OriginalTitle, candidate.media.ReleaseDate
		} else {
			item.Name, item.OriginalName, item.FirstAirDate = title, candidate.media.OriginalTitle, candidate.media.ReleaseDate
		}
		result = append(result, item)
	}
	return result, nil
}

func addWeighted(target map[int]float64, values []int, weight float64) {
	seen := map[int]bool{}
	for _, value := range values {
		if value > 0 && !seen[value] {
			target[value] += weight
			seen[value] = true
		}
	}
}

func addDocumentFrequency(target map[int]float64, values []int) {
	seen := map[int]bool{}
	for _, value := range values {
		if value > 0 && !seen[value] {
			target[value]++
			seen[value] = true
		}
	}
}

func featureScore(preferences, frequency map[int]float64, values []int, documents int, familyWeight float64) float64 {
	score := 0.0
	for _, value := range values {
		preference := preferences[value]
		if preference <= 0 {
			continue
		}
		idf := math.Log(1 + (float64(documents)+1)/(frequency[value]+1))
		score += preference * idf * familyWeight
	}
	return score
}

func diversify(candidates []rankedMedia, limit int) []rankedMedia {
	selected := make([]rankedMedia, 0, min(limit, len(candidates)))
	remaining := append([]rankedMedia(nil), candidates...)
	for len(selected) < limit && len(remaining) > 0 {
		bestIndex, bestScore := 0, math.Inf(-1)
		for index, candidate := range remaining {
			// Candidates remain sorted by their unadjusted score. The diversity
			// penalty can only lower that score, so nothing after this point can
			// beat the best candidate already found.
			if candidate.score <= bestScore {
				break
			}
			penalty := 0.0
			for _, prior := range selected {
				penalty = math.Max(penalty, featureOverlap(candidate.tokens, prior.tokens))
			}
			adjusted := candidate.score * (1 - 0.18*penalty)
			if adjusted > bestScore {
				bestIndex, bestScore = index, adjusted
			}
		}
		selected = append(selected, remaining[bestIndex])
		remaining = append(remaining[:bestIndex], remaining[bestIndex+1:]...)
	}
	return selected
}

func featureTokens(features store.CatalogFeatures) map[uint64]struct{} {
	tokens := make(map[uint64]struct{}, len(features.Genres)+len(features.Keywords)+len(features.Companies)+len(features.Cast)+len(features.Directors))
	addFamily := func(family uint64, values []int) {
		for _, value := range values {
			if value > 0 {
				tokens[family<<56|uint64(uint32(value))] = struct{}{}
			}
		}
	}
	addFamily(1, features.Genres)
	addFamily(2, features.Keywords)
	addFamily(3, features.Companies)
	addFamily(4, features.Cast)
	addFamily(5, features.Directors)
	return tokens
}

func featureOverlap(leftIDs, rightIDs map[uint64]struct{}) float64 {
	if len(leftIDs) > len(rightIDs) {
		leftIDs, rightIDs = rightIDs, leftIDs
	}
	intersection := 0
	for value := range leftIDs {
		if _, exists := rightIDs[value]; exists {
			intersection++
		}
	}
	union := len(leftIDs) + len(rightIDs) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func localizedText(media store.CatalogMedia, language string) (string, string) {
	title, overview := media.Title, media.Overview
	primary := primaryLanguage(language)
	if primary == "" || primary == "en" {
		return title, overview
	}
	var payload struct {
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
	if json.Unmarshal(media.Payload, &payload) != nil {
		return title, overview
	}
	for _, translation := range payload.Translations.Results {
		if strings.ToLower(translation.Language) != primary {
			continue
		}
		translatedTitle := translation.Data.Title
		if translatedTitle == "" {
			translatedTitle = translation.Data.Name
		}
		if translatedTitle != "" {
			title = translatedTitle
		}
		if translation.Data.Overview != "" {
			overview = translation.Data.Overview
		}
		break
	}
	return title, overview
}

func primaryLanguage(language string) string {
	return strings.ToLower(strings.Split(strings.ReplaceAll(language, "_", "-"), "-")[0])
}
