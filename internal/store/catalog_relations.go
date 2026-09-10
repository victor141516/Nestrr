package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *Store) PutGenreCatalog(ctx context.Context, mediaType, language string, genres []Genre) error {
	if mediaType != "movie" && mediaType != "tv" {
		return fmt.Errorf("invalid genre media type")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	language = NormalizeLanguage(language)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, genre := range genres {
		name := strings.TrimSpace(genre.Name)
		if genre.ID <= 0 || name == "" {
			continue
		}
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO genres(media_type,tmdb_id) VALUES(?,?)", mediaType, genre.ID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO genre_names(media_type,genre_tmdb_id,language,name,normalized_name,updated_at)
			VALUES(?,?,?,?,?,?) ON CONFLICT(media_type,genre_tmdb_id,language) DO UPDATE SET
			name=excluded.name,normalized_name=excluded.normalized_name,updated_at=excluded.updated_at`, mediaType, genre.ID, language, name, NormalizePersonQuery(name), now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ReplaceMediaGenres(ctx context.Context, mediaType string, mediaTMDBID int, language string, genres []Genre) error {
	if !validCatalogKey(CatalogKey{MediaType: mediaType, TMDBID: mediaTMDBID}) {
		return fmt.Errorf("invalid media genre identity")
	}
	language = NormalizeLanguage(language)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, "DELETE FROM media_genres WHERE media_type=? AND media_tmdb_id=?", mediaType, mediaTMDBID); err != nil {
		return err
	}
	seen := map[int]bool{}
	for _, genre := range genres {
		if genre.ID <= 0 || seen[genre.ID] {
			continue
		}
		seen[genre.ID] = true
		if _, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO genres(media_type,tmdb_id) VALUES(?,?)", mediaType, genre.ID); err != nil {
			return err
		}
		if name := strings.TrimSpace(genre.Name); name != "" {
			if _, err = tx.ExecContext(ctx, `INSERT INTO genre_names(media_type,genre_tmdb_id,language,name,normalized_name,updated_at)
				VALUES(?,?,?,?,?,?) ON CONFLICT(media_type,genre_tmdb_id,language) DO UPDATE SET
				name=excluded.name,normalized_name=excluded.normalized_name,updated_at=excluded.updated_at`, mediaType, genre.ID, language, name, NormalizePersonQuery(name), now); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO media_genres(media_type,media_tmdb_id,genre_tmdb_id,updated_at) VALUES(?,?,?,?)`, mediaType, mediaTMDBID, genre.ID, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ReplaceMediaKeywords(ctx context.Context, mediaType string, mediaTMDBID int, keywords []Keyword) error {
	if !validCatalogKey(CatalogKey{MediaType: mediaType, TMDBID: mediaTMDBID}) {
		return fmt.Errorf("invalid media keyword identity")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, "DELETE FROM media_keywords WHERE media_type=? AND media_tmdb_id=?", mediaType, mediaTMDBID); err != nil {
		return err
	}
	seen := map[int]bool{}
	for _, keyword := range keywords {
		if keyword.ID <= 0 || seen[keyword.ID] {
			continue
		}
		seen[keyword.ID] = true
		name := strings.TrimSpace(keyword.Name)
		if _, err = tx.ExecContext(ctx, `INSERT INTO keywords(tmdb_id,name,normalized_name,updated_at) VALUES(?,?,?,?)
			ON CONFLICT(tmdb_id) DO UPDATE SET
			name=CASE WHEN excluded.name<>'' THEN excluded.name ELSE keywords.name END,
			normalized_name=CASE WHEN excluded.normalized_name<>'' THEN excluded.normalized_name ELSE keywords.normalized_name END,
			updated_at=excluded.updated_at`, keyword.ID, name, NormalizePersonQuery(name), now); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO media_keywords(media_type,media_tmdb_id,keyword_tmdb_id,updated_at) VALUES(?,?,?,?)`, mediaType, mediaTMDBID, keyword.ID, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ReplaceKeywordCatalog(ctx context.Context, keywords []Keyword, updatedAt time.Time) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := updatedAt.UTC().Format(time.RFC3339Nano)
	statement, err := tx.PrepareContext(ctx, `INSERT INTO keywords(tmdb_id,name,normalized_name,updated_at) VALUES(?,?,?,?)
		ON CONFLICT(tmdb_id) DO UPDATE SET name=excluded.name,normalized_name=excluded.normalized_name,updated_at=excluded.updated_at`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, keyword := range keywords {
		name := strings.TrimSpace(keyword.Name)
		if keyword.ID <= 0 || name == "" {
			continue
		}
		if _, err = statement.ExecContext(ctx, keyword.ID, name, NormalizePersonQuery(name), stamp); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO app_state(key,value) VALUES('keyword_catalog_updated_at',?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, stamp); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) KeywordCatalogState(ctx context.Context) (int, time.Time, error) {
	var count int
	if err := s.DB.QueryRowContext(ctx, "SELECT count(*) FROM keywords WHERE name<>''").Scan(&count); err != nil {
		return 0, time.Time{}, err
	}
	var raw string
	err := s.DB.QueryRowContext(ctx, "SELECT value FROM app_state WHERE key='keyword_catalog_updated_at'").Scan(&raw)
	if err != nil && !isNoRows(err) {
		return 0, time.Time{}, err
	}
	updatedAt, _ := time.Parse(time.RFC3339Nano, raw)
	return count, updatedAt, nil
}

func (s *Store) SearchKeywords(ctx context.Context, query, phase string, limit int) ([]Keyword, error) {
	normalized := NormalizePersonQuery(query)
	if normalized == "" || limit <= 0 {
		return []Keyword{}, nil
	}
	where := "substr(normalized_name,1,length(?))=?"
	args := []any{normalized, normalized, limit}
	if phase == "contains" {
		where = "instr(normalized_name,?)>1"
		args = []any{normalized, limit}
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,name FROM keywords WHERE name<>'' AND `+where+`
		ORDER BY normalized_name,tmdb_id LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanKeywords(rows)
}

func (s *Store) KeywordsByIDs(ctx context.Context, ids []int) ([]Keyword, error) {
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 {
		return []Keyword{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT tmdb_id,name FROM keywords WHERE tmdb_id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values, err := scanKeywords(rows)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]Keyword, len(values))
	for _, value := range values {
		byID[value.ID] = value
	}
	ordered := make([]Keyword, 0, len(values))
	for _, id := range ids {
		if value, ok := byID[id]; ok {
			ordered = append(ordered, value)
		}
	}
	return ordered, nil
}

func scanKeywords(rows *sql.Rows) ([]Keyword, error) {
	values := []Keyword{}
	for rows.Next() {
		var value Keyword
		if err := rows.Scan(&value.ID, &value.Name); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func isNoRows(err error) bool { return err == sql.ErrNoRows }
