package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

type catalogTitleAlias struct {
	Language string
	Title    string
}

type LocalCatalogDocument struct {
	CatalogKey
	Payload []byte
}

type mediaTitleDocument struct {
	Title         string `json:"title"`
	Name          string `json:"name"`
	OriginalTitle string `json:"original_title"`
	OriginalName  string `json:"original_name"`
	Translations  struct {
		Results []struct {
			Language string `json:"iso_639_1"`
			Country  string `json:"iso_3166_1"`
			Data     struct {
				Title string `json:"title"`
				Name  string `json:"name"`
			} `json:"data"`
		} `json:"translations"`
	} `json:"translations"`
}

func mediaTitleAliases(mediaType, language string, payload []byte, title, originalTitle string) []catalogTitleAlias {
	var document mediaTitleDocument
	_ = json.Unmarshal(payload, &document)
	if mediaType == "tv" {
		if title == "" {
			title = document.Name
		}
		if originalTitle == "" {
			originalTitle = document.OriginalName
		}
	} else {
		if title == "" {
			title = document.Title
		}
		if originalTitle == "" {
			originalTitle = document.OriginalTitle
		}
	}
	aliases := []catalogTitleAlias{{Language: language, Title: title}, {Title: originalTitle}}
	for _, translation := range document.Translations.Results {
		translatedTitle := translation.Data.Title
		if mediaType == "tv" {
			translatedTitle = translation.Data.Name
		}
		translationLanguage := strings.TrimSpace(translation.Language)
		if country := strings.TrimSpace(translation.Country); translationLanguage != "" && country != "" {
			translationLanguage += "-" + country
		}
		aliases = append(aliases, catalogTitleAlias{Language: translationLanguage, Title: translatedTitle})
	}
	return aliases
}

func replaceCatalogTitleAliases(ctx context.Context, tx *sql.Tx, mediaType string, tmdbID int, source, language string, aliases []catalogTitleAlias) error {
	language = NormalizeLanguage(language)
	deleteQuery := `DELETE FROM catalog_media_titles WHERE media_type=? AND media_tmdb_id=? AND source=?`
	deleteArgs := []any{mediaType, tmdbID, source}
	if source == "metadata" {
		deleteQuery += " AND language=?"
		deleteArgs = append(deleteArgs, language)
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return err
	}
	ftsDeleteQuery := `DELETE FROM catalog_media_titles_fts WHERE media_type=? AND media_tmdb_id=? AND source=?`
	ftsDeleteArgs := []any{mediaType, tmdbID, source}
	if source == "metadata" {
		ftsDeleteQuery += " AND language=?"
		ftsDeleteArgs = append(ftsDeleteArgs, language)
	}
	if _, err := tx.ExecContext(ctx, ftsDeleteQuery, ftsDeleteArgs...); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, alias := range aliases {
		title := strings.TrimSpace(alias.Title)
		normalized := NormalizePersonQuery(title)
		if normalized == "" {
			continue
		}
		aliasLanguage := NormalizeLanguage(alias.Language)
		if source == "metadata" && strings.TrimSpace(alias.Language) == "" {
			aliasLanguage = language
		}
		key := aliasLanguage + "\x00" + normalized
		if seen[key] {
			continue
		}
		seen[key] = true
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO catalog_media_titles(media_type,media_tmdb_id,source,language,title,normalized_title) VALUES(?,?,?,?,?,?)`, mediaType, tmdbID, source, aliasLanguage, title, normalized); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_media_titles_fts(normalized_title,media_type,media_tmdb_id,source,language,title) VALUES(?,?,?,?,?,?)`, normalized, mediaType, tmdbID, source, aliasLanguage, title); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) migrateCatalogMediaTitles() error {
	const migrationKey = "catalog_media_titles_v1"
	var completed string
	if err := s.DB.QueryRow("SELECT value FROM app_state WHERE key=?", migrationKey).Scan(&completed); err == nil && completed == "complete" {
		return nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	type sourceDocument struct {
		mediaType, source, language, title, originalTitle string
		tmdbID                                            int
		payload                                           []byte
	}
	documents := []sourceDocument{}
	rows, err := s.DB.Query(`SELECT media_type,tmdb_id,'catalog','',payload,title,original_title FROM catalog_media
		UNION ALL SELECT media_type,tmdb_id,'metadata',language,payload,'','' FROM metadata WHERE media_type IN ('movie','tv')`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var document sourceDocument
		if err = rows.Scan(&document.mediaType, &document.tmdbID, &document.source, &document.language, &document.payload, &document.title, &document.originalTitle); err != nil {
			rows.Close()
			return err
		}
		documents = append(documents, document)
	}
	if err = rows.Close(); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, document := range documents {
		aliases := mediaTitleAliases(document.mediaType, document.language, document.payload, document.title, document.originalTitle)
		if err = replaceCatalogTitleAliases(context.Background(), tx, document.mediaType, document.tmdbID, document.source, NormalizeLanguage(document.language), aliases); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`INSERT INTO app_state(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, migrationKey, "complete"); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) migrateCatalogMediaTitleFTS() error {
	const migrationKey = "catalog_media_titles_fts_v1"
	var completed string
	if err := s.DB.QueryRow("SELECT value FROM app_state WHERE key=?", migrationKey).Scan(&completed); err == nil && completed == "complete" {
		return nil
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`DELETE FROM catalog_media_titles_fts`); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO catalog_media_titles_fts(normalized_title,media_type,media_tmdb_id,source,language,title)
		SELECT normalized_title,media_type,media_tmdb_id,source,language,title FROM catalog_media_titles`); err != nil {
		return err
	}
	if _, err = tx.Exec(`INSERT INTO app_state(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, migrationKey, "complete"); err != nil {
		return err
	}
	return tx.Commit()
}

func catalogFTSQuery(normalized string) string {
	return `"` + strings.ReplaceAll(normalized, `"`, `""`) + `"`
}

func (s *Store) SearchLocalCatalog(ctx context.Context, mediaType, query, language string, limit int) ([]LocalCatalogDocument, int, error) {
	normalized := NormalizePersonQuery(query)
	if normalized == "" || (mediaType != "movie" && mediaType != "tv") {
		return []LocalCatalogDocument{}, 0, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	queryIDs := func(searchTable, searchPredicate string, searchArgs []any, queryLimit int) ([]int, error) {
		args := []any{normalized, normalized, normalized, NormalizeLanguage(language), NormalizeLanguage(language), mediaType}
		args = append(args, searchArgs...)
		args = append(args, queryLimit)
		rows, err := s.DB.QueryContext(ctx, `SELECT media_tmdb_id,
		MIN(CASE WHEN normalized_title=? THEN 0 WHEN substr(normalized_title,1,length(?))=? THEN 1 ELSE 2 END) AS match_rank,
		MAX(CASE WHEN language=? OR substr(language,1,2)=substr(?,1,2) THEN 1 ELSE 0 END) AS locale_rank,
		COALESCE(MAX(catalog_media.popularity),0) AS popularity
		FROM `+searchTable+` LEFT JOIN catalog_media ON catalog_media.media_type=`+searchTable+`.media_type AND catalog_media.tmdb_id=`+searchTable+`.media_tmdb_id
		WHERE `+searchTable+`.media_type=? AND `+searchPredicate+`
		GROUP BY media_tmdb_id ORDER BY match_rank,locale_rank DESC,popularity DESC,media_tmdb_id LIMIT ?`, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		ids := []int{}
		for rows.Next() {
			var id, matchRank, localeRank int
			var popularity float64
			if err = rows.Scan(&id, &matchRank, &localeRank, &popularity); err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return ids, rows.Err()
	}
	// Prefix matches can use the ordinary B-tree and are both the most relevant
	// and the cheapest live result. Only consult the trigram index when they do
	// not fill the response.
	ids, err := queryIDs("catalog_media_titles", "normalized_title>=? AND normalized_title<?", []any{normalized, normalized + "\U0010ffff"}, limit)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) < limit && utf8.RuneCountInString(normalized) >= 3 {
		containsIDs, containsErr := queryIDs("catalog_media_titles_fts", "catalog_media_titles_fts MATCH ?", []any{catalogFTSQuery(normalized)}, limit*2)
		if containsErr != nil {
			return nil, 0, containsErr
		}
		seen := make(map[int]bool, len(ids))
		for _, id := range ids {
			seen[id] = true
		}
		for _, id := range containsIDs {
			if !seen[id] {
				ids = append(ids, id)
				seen[id] = true
				if len(ids) == limit {
					break
				}
			}
		}
	}
	documents := make([]LocalCatalogDocument, 0, len(ids))
	for _, id := range ids {
		var payload []byte
		err = s.DB.QueryRowContext(ctx, `SELECT payload FROM metadata WHERE media_type=? AND tmdb_id=? AND language=?`, mediaType, id, NormalizeLanguage(language)).Scan(&payload)
		if errors.Is(err, sql.ErrNoRows) {
			err = s.DB.QueryRowContext(ctx, `SELECT payload FROM catalog_media WHERE media_type=? AND tmdb_id=?`, mediaType, id).Scan(&payload)
		}
		if errors.Is(err, sql.ErrNoRows) {
			err = s.DB.QueryRowContext(ctx, `SELECT payload FROM metadata WHERE media_type=? AND tmdb_id=? ORDER BY CASE WHEN language='en-US' THEN 0 ELSE 1 END,updated_at DESC LIMIT 1`, mediaType, id).Scan(&payload)
		}
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, 0, err
		}
		documents = append(documents, LocalCatalogDocument{CatalogKey: CatalogKey{MediaType: mediaType, TMDBID: id}, Payload: payload})
	}
	return documents, len(documents), nil
}
