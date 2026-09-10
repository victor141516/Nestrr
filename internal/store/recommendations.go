package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type CatalogKey struct {
	MediaType string `json:"mediaType"`
	TMDBID    int    `json:"tmdbId"`
}

type CatalogFeatures struct {
	Genres    []int `json:"genres"`
	Keywords  []int `json:"keywords"`
	Companies []int `json:"companies"`
	Cast      []int `json:"cast"`
	Directors []int `json:"directors"`
}

type CatalogMedia struct {
	CatalogKey
	Payload       []byte
	Features      []byte
	Title         string
	OriginalTitle string
	Overview      string
	PosterPath    string
	BackdropPath  string
	ReleaseDate   string
	VoteAverage   float64
	VoteCount     int
	Popularity    float64
	UpdatedAt     string
}

type RecommendationSeed struct {
	UserID int64
	CatalogKey
	Weight float64
}

type RecommendationRunRecord struct {
	ID         int64
	State      string
	Payload    []byte
	StartedAt  string
	UpdatedAt  string
	FinishedAt string
}

func validCatalogKey(key CatalogKey) bool {
	return (key.MediaType == "movie" || key.MediaType == "tv") && key.TMDBID > 0
}

func (s *Store) PutCatalogMedia(ctx context.Context, media CatalogMedia) error {
	if !validCatalogKey(media.CatalogKey) || len(media.Payload) == 0 || len(media.Features) == 0 {
		return fmt.Errorf("invalid complete catalog media")
	}
	var features CatalogFeatures
	if err := json.Unmarshal(media.Features, &features); err != nil {
		return fmt.Errorf("invalid catalog features: %w", err)
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `INSERT INTO catalog_media(media_type,tmdb_id,payload,features,title,original_title,overview,poster_path,backdrop_path,release_date,vote_average,vote_count,popularity,updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(media_type,tmdb_id) DO UPDATE SET payload=excluded.payload,features=excluded.features,title=excluded.title,original_title=excluded.original_title,overview=excluded.overview,poster_path=excluded.poster_path,backdrop_path=excluded.backdrop_path,release_date=excluded.release_date,vote_average=excluded.vote_average,vote_count=excluded.vote_count,popularity=excluded.popularity,updated_at=excluded.updated_at`,
		media.MediaType, media.TMDBID, media.Payload, media.Features, media.Title, media.OriginalTitle, media.Overview, media.PosterPath, media.BackdropPath, media.ReleaseDate, media.VoteAverage, media.VoteCount, media.Popularity, now); err != nil {
		return err
	}
	if err = replaceCatalogTitleAliases(ctx, tx, media.MediaType, media.TMDBID, "catalog", "", mediaTitleAliases(media.MediaType, "", media.Payload, media.Title, media.OriginalTitle)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM recommendation_candidates WHERE media_type=? AND tmdb_id=?", media.MediaType, media.TMDBID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM media_genres WHERE media_type=? AND media_tmdb_id=?", media.MediaType, media.TMDBID); err != nil {
		return err
	}
	for _, genreID := range uniquePositiveIDs(features.Genres) {
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO genres(media_type,tmdb_id) VALUES(?,?)", media.MediaType, genreID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO media_genres(media_type,media_tmdb_id,genre_tmdb_id,updated_at) VALUES(?,?,?,?)", media.MediaType, media.TMDBID, genreID, now); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM media_keywords WHERE media_type=? AND media_tmdb_id=?", media.MediaType, media.TMDBID); err != nil {
		return err
	}
	for _, keywordID := range uniquePositiveIDs(features.Keywords) {
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO keywords(tmdb_id,name,normalized_name,updated_at) VALUES(?, '', '', ?)", keywordID, now); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO media_keywords(media_type,media_tmdb_id,keyword_tmdb_id,updated_at) VALUES(?,?,?,?)", media.MediaType, media.TMDBID, keywordID, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanCatalogMedia(row interface{ Scan(...any) error }) (CatalogMedia, error) {
	var media CatalogMedia
	err := row.Scan(&media.MediaType, &media.TMDBID, &media.Payload, &media.Features, &media.Title, &media.OriginalTitle, &media.Overview, &media.PosterPath, &media.BackdropPath, &media.ReleaseDate, &media.VoteAverage, &media.VoteCount, &media.Popularity, &media.UpdatedAt)
	return media, err
}

const catalogMediaColumns = `media_type,tmdb_id,payload,features,title,original_title,overview,poster_path,backdrop_path,release_date,vote_average,vote_count,popularity,updated_at`

func (s *Store) CatalogMediaByKey(ctx context.Context, key CatalogKey) (CatalogMedia, error) {
	return scanCatalogMedia(s.DB.QueryRowContext(ctx, "SELECT "+catalogMediaColumns+" FROM catalog_media WHERE media_type=? AND tmdb_id=?", key.MediaType, key.TMDBID))
}

func (s *Store) CatalogMediaEntries(ctx context.Context) ([]CatalogMedia, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT "+catalogMediaColumns+" FROM catalog_media ORDER BY media_type,tmdb_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []CatalogMedia{}
	for rows.Next() {
		entry, err := scanCatalogMedia(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) CatalogMediaRankingEntries(ctx context.Context) ([]CatalogMedia, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT media_type,tmdb_id,features,title,original_title,overview,poster_path,backdrop_path,release_date,vote_average,vote_count,popularity,updated_at FROM catalog_media ORDER BY media_type,tmdb_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []CatalogMedia{}
	for rows.Next() {
		var media CatalogMedia
		if err := rows.Scan(&media.MediaType, &media.TMDBID, &media.Features, &media.Title, &media.OriginalTitle, &media.Overview, &media.PosterPath, &media.BackdropPath, &media.ReleaseDate, &media.VoteAverage, &media.VoteCount, &media.Popularity, &media.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, media)
	}
	return entries, rows.Err()
}

func (s *Store) CatalogMediaCount(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, "SELECT count(*) FROM catalog_media").Scan(&count)
	return count, err
}

func (s *Store) MissingCatalogKeys(ctx context.Context, keys []CatalogKey) ([]CatalogKey, error) {
	missing := make([]CatalogKey, 0, len(keys))
	seen := map[CatalogKey]bool{}
	for _, key := range keys {
		if !validCatalogKey(key) || seen[key] {
			continue
		}
		seen[key] = true
		var exists int
		err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM catalog_media WHERE media_type=? AND tmdb_id=?", key.MediaType, key.TMDBID).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			missing = append(missing, key)
			continue
		}
		if err != nil {
			return nil, err
		}
	}
	return missing, nil
}

func (s *Store) AddRecommendationCandidates(ctx context.Context, keys []CatalogKey) (int, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	added := 0
	seen := map[CatalogKey]bool{}
	for _, key := range keys {
		if !validCatalogKey(key) || seen[key] {
			continue
		}
		seen[key] = true
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO recommendation_candidates(media_type,tmdb_id,status,attempts,last_error,discovered_at,updated_at)
			SELECT ?,?,'pending',0,'',?,? WHERE NOT EXISTS(SELECT 1 FROM catalog_media WHERE media_type=? AND tmdb_id=?)`, key.MediaType, key.TMDBID, now, now, key.MediaType, key.TMDBID)
		if err != nil {
			return 0, err
		}
		if count, _ := result.RowsAffected(); count > 0 {
			added++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return added, nil
}

func (s *Store) PendingRecommendationCandidates(ctx context.Context, limit int) ([]CatalogKey, error) {
	if limit <= 0 {
		limit = 100000
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT media_type,tmdb_id FROM recommendation_candidates
		WHERE status IN ('pending','failed') AND attempts<4 ORDER BY discovered_at,media_type,tmdb_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []CatalogKey{}
	for rows.Next() {
		var key CatalogKey
		if err := rows.Scan(&key.MediaType, &key.TMDBID); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

func (s *Store) StartRecommendationCandidate(ctx context.Context, key CatalogKey) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE recommendation_candidates SET status='processing',attempts=attempts+1,last_error='',updated_at=? WHERE media_type=? AND tmdb_id=?`, time.Now().UTC().Format(time.RFC3339Nano), key.MediaType, key.TMDBID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) FailRecommendationCandidate(ctx context.Context, key CatalogKey, failure string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE recommendation_candidates SET status='failed',last_error=?,updated_at=? WHERE media_type=? AND tmdb_id=?`, failure, time.Now().UTC().Format(time.RFC3339Nano), key.MediaType, key.TMDBID)
	return err
}

func (s *Store) ResetRecommendationCandidates(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE recommendation_candidates SET status='pending',attempts=CASE WHEN status='failed' AND attempts>=4 THEN 0 ELSE attempts END,updated_at=? WHERE status IN ('processing','failed')`, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) RecommendationCandidateCounts(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{"pending": 0, "processing": 0, "failed": 0}
	rows, err := s.DB.QueryContext(ctx, "SELECT status,count(*) FROM recommendation_candidates GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

func (s *Store) RecommendationDiscoveryPage(ctx context.Context, mediaType, signalType string, signalID int, sortBy string) (int, error) {
	var page int
	err := s.DB.QueryRowContext(ctx, `SELECT next_page FROM recommendation_discovery_cursors WHERE media_type=? AND signal_type=? AND signal_id=? AND sort_by=?`, mediaType, signalType, signalID, sortBy).Scan(&page)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	if page < 1 {
		page = 1
	}
	return page, err
}

func (s *Store) SetRecommendationDiscoveryPage(ctx context.Context, mediaType, signalType string, signalID int, sortBy string, nextPage int) error {
	if nextPage < 1 || nextPage > 500 {
		nextPage = 1
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO recommendation_discovery_cursors(media_type,signal_type,signal_id,sort_by,next_page,updated_at)
		VALUES(?,?,?,?,?,?) ON CONFLICT(media_type,signal_type,signal_id,sort_by) DO UPDATE SET next_page=excluded.next_page,updated_at=excluded.updated_at`, mediaType, signalType, signalID, sortBy, nextPage, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) RecommendationSeeds(ctx context.Context, userID int64) ([]RecommendationSeed, error) {
	listFilter, progressFilter, episodeProgressFilter := "", "", ""
	args := []any{}
	if userID > 0 {
		listFilter = " AND l.user_id=?"
		progressFilter = " AND p.user_id=?"
		episodeProgressFilter = " AND p.user_id=?"
		args = append(args, userID, userID, userID)
	}
	query := `SELECT user_id,media_type,tmdb_id,SUM(weight) FROM (
		SELECT l.user_id,i.media_type,i.tmdb_id,CASE l.system_key WHEN 'favorites' THEN 5.0 ELSE 2.0 END AS weight
		FROM media_lists l JOIN media_list_items i ON i.list_id=l.id
		WHERE l.system_key IN ('favorites','watched')` + listFilter + `
		UNION ALL
		SELECT p.user_id,CASE p.media_type WHEN 'show' THEN 'tv' ELSE p.media_type END,p.tmdb_id,2.0
		FROM progress p WHERE p.watched=1 AND p.media_type IN ('movie','tv','show')` + progressFilter + `
		UNION ALL
		SELECT p.user_id,'tv',p.show_tmdb_id,2.0
		FROM progress p WHERE p.watched=1 AND p.media_type='episode' AND p.show_tmdb_id>0` + episodeProgressFilter + `
		GROUP BY p.user_id,p.show_tmdb_id
	) GROUP BY user_id,media_type,tmdb_id`
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seeds := []RecommendationSeed{}
	for rows.Next() {
		var seed RecommendationSeed
		if err := rows.Scan(&seed.UserID, &seed.MediaType, &seed.TMDBID, &seed.Weight); err != nil {
			return nil, err
		}
		if validCatalogKey(seed.CatalogKey) {
			seeds = append(seeds, seed)
		}
	}
	return seeds, rows.Err()
}

func (s *Store) RecommendationExclusions(ctx context.Context, userID int64) (map[CatalogKey]bool, error) {
	excluded := map[CatalogKey]bool{}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.media_type,i.tmdb_id FROM media_lists l JOIN media_list_items i ON i.list_id=l.id WHERE l.user_id=?
		UNION SELECT CASE media_type WHEN 'show' THEN 'tv' ELSE media_type END,tmdb_id FROM progress WHERE user_id=? AND media_type IN ('movie','tv','show') AND (started=1 OR watched=1)
		UNION SELECT 'tv',show_tmdb_id FROM progress WHERE user_id=? AND media_type='episode' AND show_tmdb_id>0 AND (started=1 OR watched=1)`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var key CatalogKey
		if err := rows.Scan(&key.MediaType, &key.TMDBID); err != nil {
			return nil, err
		}
		if validCatalogKey(key) {
			excluded[key] = true
		}
	}
	return excluded, rows.Err()
}

func (s *Store) CreateRecommendationRun(ctx context.Context, state string, payload []byte) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.DB.ExecContext(ctx, `INSERT INTO recommendation_runs(state,payload,started_at,updated_at) VALUES(?,?,?,?)`, state, payload, now, now)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) UpdateRecommendationRun(ctx context.Context, id int64, state string, payload []byte, finished bool) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var finishedAt any
	if finished {
		finishedAt = now
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE recommendation_runs SET state=?,payload=?,updated_at=?,finished_at=COALESCE(?,finished_at) WHERE id=?`, state, payload, now, finishedAt, id)
	return err
}

func (s *Store) LatestRecommendationRun(ctx context.Context) (RecommendationRunRecord, error) {
	var record RecommendationRunRecord
	var finished sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT id,state,payload,started_at,updated_at,finished_at FROM recommendation_runs ORDER BY id DESC LIMIT 1`).Scan(&record.ID, &record.State, &record.Payload, &record.StartedAt, &record.UpdatedAt, &finished)
	if finished.Valid {
		record.FinishedAt = finished.String
	}
	return record, err
}
