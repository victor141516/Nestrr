package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Person struct {
	TMDBID             int     `json:"id"`
	Name               string  `json:"name"`
	ProfilePath        string  `json:"profilePath,omitempty"`
	KnownForDepartment string  `json:"knownForDepartment,omitempty"`
	Popularity         float64 `json:"popularity,omitempty"`
	Adult              bool    `json:"adult,omitempty"`
}

type MediaCredit struct {
	MediaType    string
	MediaTMDBID  int
	CreditKey    string
	PersonTMDBID int
	CreditKind   string
	Department   string
	Job          string
	Character    string
	Order        int
}

func NormalizePersonQuery(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func upsertPerson(ctx context.Context, tx *sql.Tx, person Person, updatedAt string) error {
	if person.TMDBID <= 0 || strings.TrimSpace(person.Name) == "" {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO people(tmdb_id,name,normalized_name,profile_path,known_for_department,popularity,adult,updated_at)
		VALUES(?,?,?,?,?,?,?,?)
		ON CONFLICT(tmdb_id) DO UPDATE SET
		 name=excluded.name,
		 normalized_name=excluded.normalized_name,
		 profile_path=CASE WHEN excluded.profile_path<>'' THEN excluded.profile_path ELSE people.profile_path END,
		 known_for_department=CASE WHEN excluded.known_for_department<>'' THEN excluded.known_for_department ELSE people.known_for_department END,
		 popularity=MAX(people.popularity,excluded.popularity),
		 adult=excluded.adult,
		 updated_at=excluded.updated_at`,
		person.TMDBID, strings.TrimSpace(person.Name), NormalizePersonQuery(person.Name), strings.TrimSpace(person.ProfilePath), strings.TrimSpace(person.KnownForDepartment), person.Popularity, person.Adult, updatedAt)
	return err
}

func (s *Store) ReplaceMediaCredits(ctx context.Context, mediaType string, mediaTMDBID int, people []Person, credits []MediaCredit) error {
	if mediaType == "" || mediaTMDBID <= 0 {
		return fmt.Errorf("invalid media credit identity")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, person := range people {
		if err := upsertPerson(ctx, tx, person, now); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM media_credits WHERE media_type=? AND media_tmdb_id=?", mediaType, mediaTMDBID); err != nil {
		return err
	}
	for _, credit := range credits {
		if credit.PersonTMDBID <= 0 || credit.CreditKey == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO media_credits(media_type,media_tmdb_id,credit_key,person_tmdb_id,credit_kind,department,job,character_name,billing_order,updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?)`, mediaType, mediaTMDBID, credit.CreditKey, credit.PersonTMDBID, credit.CreditKind, credit.Department, credit.Job, credit.Character, credit.Order, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func scanPeople(rows *sql.Rows) ([]Person, error) {
	people := []Person{}
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.TMDBID, &person.Name, &person.ProfilePath, &person.KnownForDepartment, &person.Popularity, &person.Adult); err != nil {
			return nil, err
		}
		people = append(people, person)
	}
	return people, rows.Err()
}

func (s *Store) SearchPeople(ctx context.Context, query string, limit int) ([]Person, error) {
	normalized := NormalizePersonQuery(query)
	if normalized == "" {
		return []Person{}, nil
	}
	if limit <= 0 || limit > 30 {
		limit = 12
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,name,profile_path,known_for_department,popularity,adult
		FROM people
		WHERE adult=0 AND instr(normalized_name,?)>0
		ORDER BY CASE WHEN normalized_name=? THEN 0 WHEN substr(normalized_name,1,length(?))=? THEN 1 ELSE 2 END,
		 popularity DESC,name COLLATE NOCASE
		LIMIT ?`, normalized, normalized, normalized, normalized, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPeople(rows)
}

func (s *Store) PeopleByIDs(ctx context.Context, ids []int) ([]Person, error) {
	ids = uniquePositiveIDs(ids)
	if len(ids) == 0 {
		return []Person{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,name,profile_path,known_for_department,popularity,adult FROM people WHERE tmdb_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	people, err := scanPeople(rows)
	if err != nil {
		return nil, err
	}
	byID := make(map[int]Person, len(people))
	for _, person := range people {
		byID[person.TMDBID] = person
	}
	ordered := make([]Person, 0, len(people))
	for _, id := range ids {
		if person, ok := byID[id]; ok {
			ordered = append(ordered, person)
		}
	}
	return ordered, nil
}

func (s *Store) PutPersonSearch(ctx context.Context, query, language string, people []Person) error {
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO person_searches(normalized_query,language,fetched_at) VALUES(?,?,?)
		ON CONFLICT(normalized_query,language) DO UPDATE SET fetched_at=excluded.fetched_at`, normalized, language, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM person_search_results WHERE normalized_query=? AND language=?", normalized, language); err != nil {
		return err
	}
	seen := map[int]bool{}
	for order, person := range people {
		if person.TMDBID <= 0 || seen[person.TMDBID] {
			continue
		}
		seen[person.TMDBID] = true
		if err := upsertPerson(ctx, tx, person, now); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO person_search_results(normalized_query,language,person_tmdb_id,result_order) VALUES(?,?,?,?)`, normalized, language, person.TMDBID, order); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) CachedPersonSearch(ctx context.Context, query, language string, maxAge time.Duration) ([]Person, bool, error) {
	normalized := NormalizePersonQuery(query)
	var fetched string
	if err := s.DB.QueryRowContext(ctx, "SELECT fetched_at FROM person_searches WHERE normalized_query=? AND language=?", normalized, language).Scan(&fetched); err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	fetchedAt, err := time.Parse(time.RFC3339Nano, fetched)
	if err != nil || time.Since(fetchedAt) > maxAge {
		return nil, false, nil
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT p.tmdb_id,p.name,p.profile_path,p.known_for_department,p.popularity,p.adult
		FROM person_search_results r JOIN people p ON p.tmdb_id=r.person_tmdb_id
		WHERE r.normalized_query=? AND r.language=? ORDER BY r.result_order`, normalized, language)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	people, err := scanPeople(rows)
	return people, true, err
}

func uniquePositiveIDs(ids []int) []int {
	seen := map[int]bool{}
	result := []int{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result
}

func (s *Store) mediaIDsForCredits(ctx context.Context, mediaType, kind, job string, people []int) ([]int, error) {
	people = uniquePositiveIDs(people)
	if len(people) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(people)), ",")
	args := []any{mediaType, kind}
	query := `SELECT media_tmdb_id FROM media_credits WHERE media_type=? AND credit_kind=?`
	if job != "" {
		query += " AND job=?"
		args = append(args, job)
	}
	query += " AND person_tmdb_id IN (" + placeholders + ") GROUP BY media_tmdb_id HAVING COUNT(DISTINCT person_tmdb_id)=?"
	for _, id := range people {
		args = append(args, id)
	}
	args = append(args, len(people))
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func intersectIDs(groups ...[]int) []int {
	if len(groups) == 0 {
		return nil
	}
	counts := map[int]int{}
	for _, group := range groups {
		seen := map[int]bool{}
		for _, id := range group {
			if !seen[id] {
				seen[id] = true
				counts[id]++
			}
		}
	}
	result := []int{}
	for id, count := range counts {
		if count == len(groups) {
			result = append(result, id)
		}
	}
	sort.Ints(result)
	return result
}

func (s *Store) LocalMediaForPeople(ctx context.Context, language, mediaType string, castIDs, directorIDs []int) ([]json.RawMessage, error) {
	groups := [][]int{}
	if len(castIDs) > 0 {
		ids, err := s.mediaIDsForCredits(ctx, mediaType, "cast", "", castIDs)
		if err != nil {
			return nil, err
		}
		groups = append(groups, ids)
	}
	if len(directorIDs) > 0 {
		ids, err := s.mediaIDsForCredits(ctx, mediaType, "crew", "Director", directorIDs)
		if err != nil {
			return nil, err
		}
		groups = append(groups, ids)
	}
	ids := intersectIDs(groups...)
	if len(ids) == 0 {
		return []json.RawMessage{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := []any{mediaType, NormalizeLanguage(language)}
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT payload FROM metadata WHERE media_type=? AND language=? AND tmdb_id IN (`+placeholders+`)`, args...)
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
