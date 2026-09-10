package store

import (
	"context"
	"fmt"
	"strings"
)

type FeatureKey struct {
	Family string
	ID     int
}

type FeaturePreference struct {
	FeatureKey
	Weight float64
}

type RankedCatalogMedia struct {
	Media CatalogMedia
	Score float64
}

func (s *Store) CatalogFeaturesForKeys(ctx context.Context, keys []CatalogKey) (map[CatalogKey]CatalogFeatures, error) {
	result := make(map[CatalogKey]CatalogFeatures, len(keys))
	grouped := map[string][]int{"movie": {}, "tv": {}}
	for _, key := range keys {
		if !validCatalogKey(key) {
			continue
		}
		result[key] = CatalogFeatures{}
		grouped[key.MediaType] = append(grouped[key.MediaType], key.TMDBID)
	}
	for mediaType, ids := range grouped {
		ids = uniquePositiveIDs(ids)
		if len(ids) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		args := make([]any, 0, len(ids)+1)
		args = append(args, mediaType)
		for _, id := range ids {
			args = append(args, id)
		}
		queries := []struct {
			family, query string
		}{
			{"genre", "SELECT media_tmdb_id,genre_tmdb_id FROM media_genres WHERE media_type=? AND media_tmdb_id IN (" + placeholders + ")"},
			{"keyword", "SELECT media_tmdb_id,keyword_tmdb_id FROM media_keywords WHERE media_type=? AND media_tmdb_id IN (" + placeholders + ")"},
			{"company", "SELECT media_tmdb_id,company_tmdb_id FROM media_companies WHERE media_type=? AND media_tmdb_id IN (" + placeholders + ")"},
			{"cast", "SELECT media_tmdb_id,person_tmdb_id FROM media_credits WHERE media_type=? AND media_tmdb_id IN (" + placeholders + ") AND credit_kind='cast' AND billing_order<20 GROUP BY media_tmdb_id,person_tmdb_id"},
			{"director", "SELECT media_tmdb_id,person_tmdb_id FROM media_credits WHERE media_type=? AND media_tmdb_id IN (" + placeholders + ") AND credit_kind='crew' AND job IN ('Director','Creator') GROUP BY media_tmdb_id,person_tmdb_id"},
		}
		for _, definition := range queries {
			rows, err := s.DB.QueryContext(ctx, definition.query, args...)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var mediaID, featureID int
				if err := rows.Scan(&mediaID, &featureID); err != nil {
					rows.Close()
					return nil, err
				}
				key := CatalogKey{MediaType: mediaType, TMDBID: mediaID}
				features := result[key]
				switch definition.family {
				case "genre":
					features.Genres = append(features.Genres, featureID)
				case "keyword":
					features.Keywords = append(features.Keywords, featureID)
				case "company":
					features.Companies = append(features.Companies, featureID)
				case "cast":
					features.Cast = append(features.Cast, featureID)
				case "director":
					features.Directors = append(features.Directors, featureID)
				}
				result[key] = features
			}
			if err := rows.Close(); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (s *Store) FeatureDocumentFrequencies(ctx context.Context, mediaType string, preferences []FeaturePreference) (map[FeatureKey]int, int, error) {
	frequencies := map[FeatureKey]int{}
	var documents int
	if err := s.DB.QueryRowContext(ctx, "SELECT count(*) FROM catalog_media WHERE media_type=?", mediaType).Scan(&documents); err != nil {
		return nil, 0, err
	}
	grouped := map[string][]int{}
	for _, preference := range preferences {
		grouped[preference.Family] = append(grouped[preference.Family], preference.ID)
	}
	queries := map[string]string{
		"genre":    "SELECT genre_tmdb_id,count(*) FROM media_genres WHERE media_type=? AND genre_tmdb_id IN (%s) GROUP BY genre_tmdb_id",
		"keyword":  "SELECT keyword_tmdb_id,count(*) FROM media_keywords WHERE media_type=? AND keyword_tmdb_id IN (%s) GROUP BY keyword_tmdb_id",
		"company":  "SELECT company_tmdb_id,count(*) FROM media_companies WHERE media_type=? AND company_tmdb_id IN (%s) GROUP BY company_tmdb_id",
		"cast":     "SELECT person_tmdb_id,count(DISTINCT media_tmdb_id) FROM media_credits WHERE media_type=? AND credit_kind='cast' AND billing_order<20 AND person_tmdb_id IN (%s) GROUP BY person_tmdb_id",
		"director": "SELECT person_tmdb_id,count(DISTINCT media_tmdb_id) FROM media_credits WHERE media_type=? AND credit_kind='crew' AND job IN ('Director','Creator') AND person_tmdb_id IN (%s) GROUP BY person_tmdb_id",
	}
	for family, ids := range grouped {
		ids = uniquePositiveIDs(ids)
		if len(ids) == 0 {
			continue
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		args := make([]any, 0, len(ids)+1)
		args = append(args, mediaType)
		for _, id := range ids {
			args = append(args, id)
		}
		rows, err := s.DB.QueryContext(ctx, fmt.Sprintf(queries[family], placeholders), args...)
		if err != nil {
			return nil, 0, err
		}
		for rows.Next() {
			var id, count int
			if err := rows.Scan(&id, &count); err != nil {
				rows.Close()
				return nil, 0, err
			}
			frequencies[FeatureKey{Family: family, ID: id}] = count
		}
		if err := rows.Close(); err != nil {
			return nil, 0, err
		}
	}
	return frequencies, documents, nil
}

func (s *Store) RankedRecommendationCandidates(ctx context.Context, userID int64, mediaType string, contributions []FeaturePreference, limit int) ([]RankedCatalogMedia, error) {
	if len(contributions) == 0 {
		return []RankedCatalogMedia{}, nil
	}
	if limit <= 0 {
		limit = 600
	}
	connection, err := s.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	tx, err := connection.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `CREATE TEMP TABLE IF NOT EXISTS recommendation_preferences (
		family TEXT NOT NULL,feature_id INTEGER NOT NULL,weight REAL NOT NULL,PRIMARY KEY(family,feature_id)
	)`); err != nil {
		tx.Rollback()
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM recommendation_preferences"); err != nil {
		tx.Rollback()
		return nil, err
	}
	statement, err := tx.PrepareContext(ctx, "INSERT INTO recommendation_preferences(family,feature_id,weight) VALUES(?,?,?)")
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	for _, contribution := range contributions {
		if contribution.Family == "" || contribution.ID <= 0 || contribution.Weight <= 0 {
			continue
		}
		if _, err = statement.ExecContext(ctx, contribution.Family, contribution.ID, contribution.Weight); err != nil {
			statement.Close()
			tx.Rollback()
			return nil, err
		}
	}
	statement.Close()
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if _, err = connection.ExecContext(ctx, "ANALYZE recommendation_preferences"); err != nil {
		return nil, err
	}
	args := []any{
		mediaType, mediaType, mediaType, mediaType, mediaType,
		userID, mediaType,
		userID, mediaType,
		userID, mediaType,
		mediaType, limit,
	}
	query := `WITH family_scores(media_tmdb_id,score) AS (
		SELECT mg.media_tmdb_id,SUM(p.weight) FROM recommendation_preferences p JOIN media_genres mg ON p.family='genre' AND mg.genre_tmdb_id=p.feature_id WHERE mg.media_type=? GROUP BY mg.media_tmdb_id
		UNION ALL SELECT mk.media_tmdb_id,SUM(p.weight) FROM recommendation_preferences p JOIN media_keywords mk ON p.family='keyword' AND mk.keyword_tmdb_id=p.feature_id WHERE mk.media_type=? GROUP BY mk.media_tmdb_id
		UNION ALL SELECT mc.media_tmdb_id,SUM(p.weight) FROM recommendation_preferences p JOIN media_companies mc ON p.family='company' AND mc.company_tmdb_id=p.feature_id WHERE mc.media_type=? GROUP BY mc.media_tmdb_id
		UNION ALL SELECT media_tmdb_id,SUM(weight) FROM (SELECT DISTINCT cr.media_tmdb_id,p.feature_id,p.weight FROM recommendation_preferences p JOIN media_credits cr ON p.family='cast' AND cr.person_tmdb_id=p.feature_id WHERE cr.media_type=? AND cr.credit_kind='cast' AND cr.billing_order<20) GROUP BY media_tmdb_id
		UNION ALL SELECT media_tmdb_id,SUM(weight) FROM (SELECT DISTINCT cr.media_tmdb_id,p.feature_id,p.weight FROM recommendation_preferences p JOIN media_credits cr ON p.family='director' AND cr.person_tmdb_id=p.feature_id WHERE cr.media_type=? AND cr.credit_kind='crew' AND cr.job IN ('Director','Creator')) GROUP BY media_tmdb_id
	), scored AS (SELECT media_tmdb_id,SUM(score) score FROM family_scores GROUP BY media_tmdb_id), excluded(tmdb_id) AS (
		SELECT i.tmdb_id FROM media_lists l JOIN media_list_items i ON i.list_id=l.id WHERE l.user_id=? AND i.media_type=?
		UNION SELECT tmdb_id FROM progress WHERE user_id=? AND CASE media_type WHEN 'show' THEN 'tv' ELSE media_type END=? AND media_type IN ('movie','tv','show') AND (started=1 OR watched=1)
		UNION SELECT show_tmdb_id FROM progress WHERE user_id=? AND ?='tv' AND media_type='episode' AND show_tmdb_id>0 AND (started=1 OR watched=1)
	) SELECT c.media_type,c.tmdb_id,c.title,c.original_title,c.overview,c.poster_path,c.backdrop_path,c.release_date,c.vote_average,c.vote_count,c.popularity,c.updated_at,scored.score
	FROM scored JOIN catalog_media c ON c.media_type=? AND c.tmdb_id=scored.media_tmdb_id
	LEFT JOIN excluded e ON e.tmdb_id=c.tmdb_id WHERE e.tmdb_id IS NULL
	ORDER BY scored.score DESC,c.vote_average DESC,c.vote_count DESC,c.popularity DESC LIMIT ?`
	rows, err := connection.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("rank recommendation candidates: %w", err)
	}
	defer rows.Close()
	result := []RankedCatalogMedia{}
	for rows.Next() {
		var ranked RankedCatalogMedia
		if err := rows.Scan(&ranked.Media.MediaType, &ranked.Media.TMDBID, &ranked.Media.Title, &ranked.Media.OriginalTitle, &ranked.Media.Overview, &ranked.Media.PosterPath, &ranked.Media.BackdropPath, &ranked.Media.ReleaseDate, &ranked.Media.VoteAverage, &ranked.Media.VoteCount, &ranked.Media.Popularity, &ranked.Media.UpdatedAt, &ranked.Score); err != nil {
			return nil, err
		}
		result = append(result, ranked)
	}
	return result, rows.Err()
}
