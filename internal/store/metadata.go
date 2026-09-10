package store

import (
	"context"
	"strings"
)

// MetadataDocument is a persisted, shared localized TMDB document. The store keeps
// the payload opaque; interpretation belongs to the server feature that owns
// the lookup.
type MetadataDocument struct {
	MediaType string
	TMDBID    int
	Language  string
	Payload   []byte
}

func (s *Store) MediaMetadata(ctx context.Context, languages ...string) ([]MetadataDocument, error) {
	query := "SELECT media_type,tmdb_id,language,payload FROM metadata WHERE media_type IN ('movie','tv')"
	args := []any{}
	if len(languages) > 0 {
		placeholders := make([]string, 0, len(languages))
		for _, language := range languages {
			placeholders = append(placeholders, "?")
			args = append(args, NormalizeLanguage(language))
		}
		query += " AND language IN (" + strings.Join(placeholders, ",") + ")"
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	documents := []MetadataDocument{}
	for rows.Next() {
		var document MetadataDocument
		if err := rows.Scan(&document.MediaType, &document.TMDBID, &document.Language, &document.Payload); err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}
