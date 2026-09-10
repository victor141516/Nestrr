package recommendations

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"
	"time"

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
	phaseStarted := time.Now()
	markPhase := func(phase string) {
		s.Log.Debug("recommendation ranking phase", "phase", phase, "durationMs", time.Since(phaseStarted).Milliseconds())
		phaseStarted = time.Now()
	}
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
	markPhase("seeds")
	seedKeys := make([]store.CatalogKey, 0, len(seeds))
	for _, seed := range seeds {
		seedKeys = append(seedKeys, seed.CatalogKey)
	}
	seedFeatures, err := s.Store.CatalogFeaturesForKeys(ctx, seedKeys)
	if err != nil {
		return nil, err
	}
	markPhase("seed_features")
	preferences := weightedFeatures{Genres: map[int]float64{}, Keywords: map[int]float64{}, Companies: map[int]float64{}, Cast: map[int]float64{}, Directors: map[int]float64{}}
	for _, seed := range seeds {
		features, ok := seedFeatures[seed.CatalogKey]
		if !ok {
			continue
		}
		addWeighted(preferences.Genres, features.Genres, seed.Weight)
		addWeighted(preferences.Keywords, features.Keywords, seed.Weight)
		addWeighted(preferences.Companies, features.Companies, seed.Weight)
		addWeighted(preferences.Cast, features.Cast, seed.Weight)
		addWeighted(preferences.Directors, features.Directors, seed.Weight)
	}

	preferenceRows := flattenPreferences(preferences)
	frequencies, documents, err := s.Store.FeatureDocumentFrequencies(ctx, mediaType, preferenceRows)
	if err != nil {
		return nil, err
	}
	markPhase("document_frequencies")
	contributions := make([]store.FeaturePreference, 0, len(preferenceRows))
	for _, preference := range preferenceRows {
		idf := math.Log(1 + (float64(documents)+1)/(float64(frequencies[preference.FeatureKey])+1))
		preference.Weight *= idf * featureFamilyWeight(preference.Family)
		contributions = append(contributions, preference)
	}
	contributions = strongestContributions(contributions)
	ranked, err := s.Store.RankedRecommendationCandidates(ctx, userID, mediaType, contributions, 600)
	if err != nil {
		return nil, err
	}
	markPhase("candidate_ranking")
	candidateKeys := make([]store.CatalogKey, 0, len(ranked))
	for _, candidate := range ranked {
		candidateKeys = append(candidateKeys, candidate.Media.CatalogKey)
	}
	candidateFeatures, err := s.Store.CatalogFeaturesForKeys(ctx, candidateKeys)
	if err != nil {
		return nil, err
	}
	markPhase("candidate_features")
	candidates := make([]rankedMedia, 0, len(ranked))
	for _, candidate := range ranked {
		features := candidateFeatures[candidate.Media.CatalogKey]
		quality := math.Max(0, math.Min(1, candidate.Media.VoteAverage/10)) * math.Log10(float64(candidate.Media.VoteCount)+1)
		score := candidate.Score + quality*0.35
		candidates = append(candidates, rankedMedia{media: candidate.Media, features: features, tokens: featureTokens(features), score: score})
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
		if shared, sharedErr := s.Store.GetMetadata(ctx, candidate.media.MediaType, candidate.media.TMDBID, language); sharedErr == nil {
			localizedMedia.Payload = shared
		} else if primaryLanguage(language) != "en" {
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

func strongestContributions(values []store.FeaturePreference) []store.FeaturePreference {
	limits := map[string]int{"genre": 64, "keyword": 384, "company": 192, "cast": 384, "director": 192}
	sort.SliceStable(values, func(left, right int) bool {
		if values[left].Weight == values[right].Weight {
			if values[left].Family == values[right].Family {
				return values[left].ID < values[right].ID
			}
			return values[left].Family < values[right].Family
		}
		return values[left].Weight > values[right].Weight
	})
	counts := map[string]int{}
	result := make([]store.FeaturePreference, 0, min(len(values), 1216))
	for _, value := range values {
		if counts[value.Family] >= limits[value.Family] {
			continue
		}
		counts[value.Family]++
		result = append(result, value)
	}
	return result
}

func flattenPreferences(preferences weightedFeatures) []store.FeaturePreference {
	result := []store.FeaturePreference{}
	appendFamily := func(family string, values map[int]float64) {
		for id, weight := range values {
			if id > 0 && weight > 0 {
				result = append(result, store.FeaturePreference{FeatureKey: store.FeatureKey{Family: family, ID: id}, Weight: weight})
			}
		}
	}
	appendFamily("genre", preferences.Genres)
	appendFamily("keyword", preferences.Keywords)
	appendFamily("company", preferences.Companies)
	appendFamily("cast", preferences.Cast)
	appendFamily("director", preferences.Directors)
	return result
}

func featureFamilyWeight(family string) float64 {
	switch family {
	case "keyword":
		return 1.3
	case "company":
		return 0.8
	case "cast":
		return 0.7
	case "director":
		return 1.4
	default:
		return 1
	}
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
		Title        string `json:"title"`
		Name         string `json:"name"`
		Overview     string `json:"overview"`
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
		return title, overview
	}
	if translatedTitle := firstNonEmpty(payload.Title, payload.Name); translatedTitle != "" {
		title = translatedTitle
		if payload.Overview != "" {
			overview = payload.Overview
		}
	}
	return title, overview
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func primaryLanguage(language string) string {
	return strings.ToLower(strings.Split(strings.ReplaceAll(language, "_", "-"), "-")[0])
}
