package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Company struct {
	TMDBID        int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logoPath,omitempty"`
	OriginCountry string `json:"originCountry,omitempty"`
}

func upsertCompany(ctx context.Context, tx *sql.Tx, company Company, updatedAt string) error {
	if company.TMDBID <= 0 || strings.TrimSpace(company.Name) == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO companies(tmdb_id,name,normalized_name,logo_path,origin_country,updated_at)
		VALUES(?,?,?,?,?,?)
		ON CONFLICT(tmdb_id) DO UPDATE SET
		 name=excluded.name,
		 normalized_name=excluded.normalized_name,
		 logo_path=CASE WHEN excluded.logo_path<>'' THEN excluded.logo_path ELSE companies.logo_path END,
		 origin_country=CASE WHEN excluded.origin_country<>'' THEN excluded.origin_country ELSE companies.origin_country END,
		 updated_at=excluded.updated_at`, company.TMDBID, strings.TrimSpace(company.Name), NormalizePersonQuery(company.Name), strings.TrimSpace(company.LogoPath), strings.TrimSpace(company.OriginCountry), updatedAt)
	return err
}

func (s *Store) ReplaceMediaCompanies(ctx context.Context, mediaType string, mediaTMDBID int, companies []Company) error {
	if mediaType == "" || mediaTMDBID <= 0 {
		return fmt.Errorf("invalid media company identity")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "DELETE FROM media_companies WHERE media_type=? AND media_tmdb_id=?", mediaType, mediaTMDBID); err != nil {
		return err
	}
	seen := map[int]bool{}
	for _, company := range companies {
		if company.TMDBID <= 0 || seen[company.TMDBID] {
			continue
		}
		seen[company.TMDBID] = true
		if err := upsertCompany(ctx, tx, company, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_companies(media_type,media_tmdb_id,company_tmdb_id,updated_at) VALUES(?,?,?,?)`, mediaType, mediaTMDBID, company.TMDBID, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanCompanies(rows *sql.Rows) ([]Company, error) {
	companies := []Company{}
	for rows.Next() {
		var company Company
		if err := rows.Scan(&company.TMDBID, &company.Name, &company.LogoPath, &company.OriginCountry); err != nil {
			return nil, err
		}
		companies = append(companies, company)
	}
	return companies, rows.Err()
}

func (s *Store) SearchCompanies(ctx context.Context, query string, limit int) ([]Company, error) {
	normalized := NormalizePersonQuery(query)
	if normalized == "" {
		return []Company{}, nil
	}
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,name,logo_path,origin_country FROM companies
		WHERE instr(normalized_name,?)>0
		ORDER BY CASE WHEN normalized_name=? THEN 0 WHEN substr(normalized_name,1,length(?))=? THEN 1 ELSE 2 END,name COLLATE NOCASE LIMIT ?`, normalized, normalized, normalized, normalized, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCompanies(rows)
}

func (s *Store) CompaniesByIDs(ctx context.Context, ids []int) ([]Company, error) {
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 {
		return []Company{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,name,logo_path,origin_country FROM companies WHERE tmdb_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	companies, err := scanCompanies(rows)
	if err != nil {
		return nil, err
	}
	byID := map[int]Company{}
	for _, company := range companies {
		byID[company.TMDBID] = company
	}
	ordered := make([]Company, 0, len(companies))
	for _, id := range ids {
		if company, ok := byID[id]; ok {
			ordered = append(ordered, company)
		}
	}
	return ordered, nil
}

func (s *Store) PutCompanySearch(ctx context.Context, query, language string, companies []Company) error {
	normalized := NormalizePersonQuery(query)
	if normalized == "" {
		return nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO company_searches(normalized_query,language,fetched_at) VALUES(?,?,?) ON CONFLICT(normalized_query,language) DO UPDATE SET fetched_at=excluded.fetched_at`, normalized, language, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM company_search_results WHERE normalized_query=? AND language=?", normalized, language); err != nil {
		return err
	}
	seen := map[int]bool{}
	for order, company := range companies {
		if company.TMDBID <= 0 || seen[company.TMDBID] {
			continue
		}
		seen[company.TMDBID] = true
		if err := upsertCompany(ctx, tx, company, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO company_search_results(normalized_query,language,company_tmdb_id,result_order) VALUES(?,?,?,?)`, normalized, language, company.TMDBID, order); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CachedCompanySearch(ctx context.Context, query, language string, maxAge time.Duration) ([]Company, bool, error) {
	normalized := NormalizePersonQuery(query)
	var fetched string
	if err := s.DB.QueryRowContext(ctx, "SELECT fetched_at FROM company_searches WHERE normalized_query=? AND language=?", normalized, language).Scan(&fetched); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	fetchedAt, err := time.Parse(time.RFC3339Nano, fetched)
	if err != nil || time.Since(fetchedAt) > maxAge {
		return nil, false, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT c.tmdb_id,c.name,c.logo_path,c.origin_country FROM company_search_results r JOIN companies c ON c.tmdb_id=r.company_tmdb_id WHERE r.normalized_query=? AND r.language=? ORDER BY r.result_order`, normalized, language)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	companies, err := scanCompanies(rows)
	return companies, true, err
}

func (s *Store) LocalMediaForCompanies(ctx context.Context, language, mediaType string, companyIDs []int) ([]json.RawMessage, error) {
	companyIDs = uniquePositiveIDs(companyIDs)
	if len(companyIDs) == 0 {
		return []json.RawMessage{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(companyIDs)), ",")
	args := []any{mediaType}
	for _, id := range companyIDs {
		args = append(args, id)
	}
	args = append(args, len(companyIDs), mediaType, NormalizeLanguage(language))
	rows, err := s.DB.QueryContext(ctx, `SELECT m.payload FROM metadata m JOIN (
		SELECT media_tmdb_id FROM media_companies WHERE media_type=? AND company_tmdb_id IN (`+placeholders+`) GROUP BY media_tmdb_id HAVING COUNT(DISTINCT company_tmdb_id)=?
	) related ON related.media_tmdb_id=m.tmdb_id WHERE m.media_type=? AND m.language=?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	results := []json.RawMessage{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		results = append(results, json.RawMessage(payload))
	}
	return results, rows.Err()
}
