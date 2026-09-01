package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

type TorrentAssociation struct {
	InfoHash string
	Category string
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 username TEXT NOT NULL COLLATE NOCASE UNIQUE,
 password_hash TEXT NOT NULL,
 is_admin INTEGER NOT NULL DEFAULT 0,
 language TEXT NOT NULL DEFAULT 'en-US',
 accent_color TEXT NOT NULL DEFAULT '#ff624f',
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS one_administrator ON users(is_admin) WHERE is_admin=1;
CREATE TABLE IF NOT EXISTS refresh_sessions (
 id TEXT PRIMARY KEY,
 token_hash TEXT NOT NULL UNIQUE,
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 actor_admin_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
 expires_at TEXT NOT NULL,
 revoked_at TEXT,
 created_at TEXT NOT NULL,
 rotated_from TEXT
);
CREATE TABLE IF NOT EXISTS audit_log (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 actor_user_id INTEGER,
 target_user_id INTEGER,
 action TEXT NOT NULL,
 details TEXT NOT NULL DEFAULT '{}',
 occurred_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS http_cache (
 cache_key TEXT PRIMARY KEY,
 status INTEGER NOT NULL,
 content_type TEXT NOT NULL,
 body BLOB NOT NULL,
 fetched_at TEXT NOT NULL,
 expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS metadata (
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 language TEXT NOT NULL,
 payload BLOB NOT NULL,
 updated_at TEXT NOT NULL,
 PRIMARY KEY(user_id,media_type,tmdb_id,language)
);
CREATE TABLE IF NOT EXISTS external_ratings (
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 rotten_tomatoes_score INTEGER NOT NULL DEFAULT 0,
 found INTEGER NOT NULL DEFAULT 0,
 provider TEXT NOT NULL DEFAULT '',
 fetched_at TEXT NOT NULL,
 PRIMARY KEY(media_type,tmdb_id)
);
CREATE TABLE IF NOT EXISTS image_assets (
 source_url TEXT PRIMARY KEY,
 local_name TEXT NOT NULL UNIQUE,
 content_type TEXT NOT NULL,
 fetched_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS availability (
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 canonical_path TEXT NOT NULL,
 available INTEGER NOT NULL,
 checked_at TEXT NOT NULL,
 PRIMARY KEY(media_type,tmdb_id)
);
CREATE TABLE IF NOT EXISTS progress (
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 show_tmdb_id INTEGER NOT NULL DEFAULT 0,
 season_number INTEGER NOT NULL DEFAULT 0,
 episode_number INTEGER NOT NULL DEFAULT 0,
 position_seconds REAL NOT NULL DEFAULT 0,
 duration_seconds REAL NOT NULL DEFAULT 0,
 watched INTEGER NOT NULL DEFAULT 0,
 watched_at TEXT,
 audio_language TEXT,
 audio_title TEXT,
 subtitle_language TEXT,
 subtitle_title TEXT,
 subtitles_enabled INTEGER NOT NULL DEFAULT 0,
 last_event_id TEXT,
 last_session_id TEXT,
 updated_at TEXT NOT NULL,
 PRIMARY KEY(user_id,media_type,tmdb_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS progress_event_id ON progress(user_id,last_event_id) WHERE last_event_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS media_lists (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 system_key TEXT,
 name TEXT NOT NULL COLLATE NOCASE,
 color TEXT NOT NULL,
 icon TEXT NOT NULL,
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS media_lists_system_key ON media_lists(user_id,system_key) WHERE system_key IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS media_lists_custom_name ON media_lists(user_id,name) WHERE system_key IS NULL;
CREATE TABLE IF NOT EXISTS media_list_items (
 list_id INTEGER NOT NULL REFERENCES media_lists(id) ON DELETE CASCADE,
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 title TEXT NOT NULL DEFAULT '',
 original_title TEXT NOT NULL DEFAULT '',
 overview TEXT NOT NULL DEFAULT '',
 poster_path TEXT NOT NULL DEFAULT '',
 release_date TEXT NOT NULL DEFAULT '',
 vote_average REAL NOT NULL DEFAULT 0,
 vote_count INTEGER NOT NULL DEFAULT 0,
 popularity REAL NOT NULL DEFAULT 0,
 runtime INTEGER NOT NULL DEFAULT 0,
 added_at TEXT NOT NULL,
 PRIMARY KEY(list_id,media_type,tmdb_id)
);
CREATE INDEX IF NOT EXISTS media_list_items_added ON media_list_items(list_id,added_at DESC);
CREATE TABLE IF NOT EXISTS torrent_associations (
 user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 media_type TEXT NOT NULL,
 tmdb_id INTEGER NOT NULL,
 info_hash TEXT NOT NULL,
 category TEXT NOT NULL,
 created_at TEXT NOT NULL,
 PRIMARY KEY(user_id,media_type,tmdb_id,info_hash)
);
CREATE TABLE IF NOT EXISTS app_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);
`
	if _, err := s.DB.Exec(schema); err != nil {
		return err
	}
	if err := s.ensureColumn("users", "accent_color", "TEXT NOT NULL DEFAULT '#ff624f'"); err != nil {
		return err
	}
	if err := s.ensureColumn("media_list_items", "overview", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	// Playback context was added after the first releases. Keep migration
	// idempotent so existing installations gain the columns without replacing
	// their database.
	for _, column := range []struct {
		name       string
		definition string
	}{
		{"show_tmdb_id", "INTEGER NOT NULL DEFAULT 0"},
		{"season_number", "INTEGER NOT NULL DEFAULT 0"},
		{"episode_number", "INTEGER NOT NULL DEFAULT 0"},
		{"started", "INTEGER NOT NULL DEFAULT 0"},
		{"started_at", "TEXT"},
	} {
		if err := s.ensureColumn("progress", column.name, column.definition); err != nil {
			return err
		}
	}
	// Existing installations already treated any persisted position (or a
	// completed row) as started. Preserve that meaning while new playback rows
	// can remain private preference-only rows until the 30 second threshold.
	if _, err := s.DB.Exec("UPDATE progress SET started=1,started_at=COALESCE(started_at,updated_at) WHERE started=0 AND (position_seconds>0 OR watched=1)"); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureColumn(table, name, definition string) error {
	rows, err := s.DB.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var columnName, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if columnName == name {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if found {
		return nil
	}
	_, err = s.DB.Exec("ALTER TABLE " + table + " ADD COLUMN " + name + " " + definition)
	return err
}

type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	IsAdmin     bool      `json:"isAdmin"`
	Language    string    `json:"language"`
	AccentColor string    `json:"accentColor"`
	CreatedAt   time.Time `json:"createdAt"`
}

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var u User
	var admin int
	var created string
	err := row.Scan(&u.ID, &u.Username, &admin, &u.Language, &u.AccentColor, &created)
	u.IsAdmin = admin == 1
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return u, err
}
func (s *Store) AdminExists(ctx context.Context) (bool, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE is_admin=1").Scan(&n)
	return n > 0, err
}
func (s *Store) CreateFirstAdmin(ctx context.Context, username, hash, language string) (User, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM users WHERE is_admin=1").Scan(&n); err != nil {
		return User{}, err
	}
	if n > 0 {
		return User{}, errors.New("administrator already exists")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, "INSERT INTO users(username,password_hash,is_admin,language,created_at,updated_at) VALUES(?,?,1,?,?,?)", username, hash, language, now, now)
	if err != nil {
		return User{}, err
	}
	id, _ := res.LastInsertId()
	if _, err = tx.ExecContext(ctx, "INSERT INTO audit_log(actor_user_id,target_user_id,action,occurred_at) VALUES(?,?,?,?)", id, id, "admin.bootstrap", now); err != nil {
		return User{}, err
	}
	if err = tx.Commit(); err != nil {
		return User{}, err
	}
	return s.UserByID(ctx, id)
}
func (s *Store) CreateUser(ctx context.Context, actor int64, username, hash, language string) (User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.DB.ExecContext(ctx, "INSERT INTO users(username,password_hash,is_admin,language,created_at,updated_at) VALUES(?,?,0,?,?,?)", username, hash, language, now, now)
	if err != nil {
		return User{}, err
	}
	id, _ := res.LastInsertId()
	s.Audit(ctx, actor, id, "user.create", map[string]any{"username": username})
	return s.UserByID(ctx, id)
}
func (s *Store) UserByUsername(ctx context.Context, username string) (User, string, error) {
	var u User
	var hash, created string
	var admin int
	err := s.DB.QueryRowContext(ctx, "SELECT id,username,password_hash,is_admin,language,accent_color,created_at FROM users WHERE username=?", username).Scan(&u.ID, &u.Username, &hash, &admin, &u.Language, &u.AccentColor, &created)
	u.IsAdmin = admin == 1
	u.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return u, hash, err
}
func (s *Store) UserByID(ctx context.Context, id int64) (User, error) {
	return scanUser(s.DB.QueryRowContext(ctx, "SELECT id,username,is_admin,language,accent_color,created_at FROM users WHERE id=?", id))
}
func (s *Store) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,username,is_admin,language,accent_color,created_at FROM users ORDER BY is_admin DESC,username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		u, e := scanUser(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
func (s *Store) UpdateUser(ctx context.Context, actor, id int64, username, language string) (User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.ExecContext(ctx, "UPDATE users SET username=?,language=?,updated_at=? WHERE id=?", username, language, now, id)
	if err != nil {
		return User{}, err
	}
	s.Audit(ctx, actor, id, "user.update", map[string]any{"username": username, "language": language})
	return s.UserByID(ctx, id)
}
func (s *Store) SetLanguage(ctx context.Context, id int64, language string) (User, error) {
	u, err := s.UserByID(ctx, id)
	if err != nil {
		return u, err
	}
	return s.UpdateUser(ctx, id, id, u.Username, language)
}

func (s *Store) SetPreferences(ctx context.Context, id int64, language, accentColor string) (User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.DB.ExecContext(ctx, "UPDATE users SET language=?,accent_color=?,updated_at=? WHERE id=?", language, accentColor, now, id)
	if err != nil {
		return User{}, err
	}
	s.Audit(ctx, id, id, "user.preferences", map[string]any{"language": language, "accentColor": accentColor})
	return s.UserByID(ctx, id)
}
func (s *Store) ResetPassword(ctx context.Context, actor, id int64, hash string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, "UPDATE users SET password_hash=?,updated_at=? WHERE id=?", hash, now, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE refresh_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL", now, id); err != nil {
		return err
	}
	if err = tx.Commit(); err == nil {
		s.Audit(ctx, actor, id, "user.password_reset", nil)
	}
	return err
}
func (s *Store) DeleteUser(ctx context.Context, actor, id int64) error {
	u, err := s.UserByID(ctx, id)
	if err != nil {
		return err
	}
	if u.IsAdmin {
		return errors.New("administrator cannot be deleted")
	}
	s.Audit(ctx, actor, id, "user.delete", map[string]any{"username": u.Username})
	_, err = s.DB.ExecContext(ctx, "DELETE FROM users WHERE id=?", id)
	return err
}

type RefreshSession struct {
	ID, TokenHash, ExpiresAt string
	UserID                   int64
	ActorAdminID             *int64
}

func (s *Store) SaveRefresh(ctx context.Context, r RefreshSession, rotatedFrom string) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO refresh_sessions(id,token_hash,user_id,actor_admin_id,expires_at,created_at,rotated_from) VALUES(?,?,?,?,?,?,?)", r.ID, r.TokenHash, r.UserID, r.ActorAdminID, r.ExpiresAt, time.Now().UTC().Format(time.RFC3339Nano), nullable(rotatedFrom))
	return err
}
func (s *Store) RotateRefresh(ctx context.Context, oldHash string, next RefreshSession) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oldID string
	var expires, revoked sql.NullString
	if err = tx.QueryRowContext(ctx, "SELECT id,expires_at,revoked_at FROM refresh_sessions WHERE token_hash=?", oldHash).Scan(&oldID, &expires, &revoked); err != nil {
		return err
	}
	if revoked.Valid {
		return errors.New("refresh session already used")
	}
	exp, _ := time.Parse(time.RFC3339Nano, expires.String)
	if time.Now().After(exp) {
		return errors.New("refresh session expired")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, "UPDATE refresh_sessions SET revoked_at=? WHERE id=? AND revoked_at IS NULL", now, oldID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO refresh_sessions(id,token_hash,user_id,actor_admin_id,expires_at,created_at,rotated_from) VALUES(?,?,?,?,?,?,?)", next.ID, next.TokenHash, next.UserID, next.ActorAdminID, next.ExpiresAt, now, oldID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) RefreshByHash(ctx context.Context, hash string) (RefreshSession, error) {
	var r RefreshSession
	var actor sql.NullInt64
	var revoked sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT id,token_hash,user_id,actor_admin_id,expires_at,revoked_at FROM refresh_sessions WHERE token_hash=?", hash).Scan(&r.ID, &r.TokenHash, &r.UserID, &actor, &r.ExpiresAt, &revoked)
	if actor.Valid {
		r.ActorAdminID = &actor.Int64
	}
	if revoked.Valid {
		return r, errors.New("refresh session revoked")
	}
	return r, err
}
func (s *Store) RevokeRefresh(ctx context.Context, hash string) error {
	_, err := s.DB.ExecContext(ctx, "UPDATE refresh_sessions SET revoked_at=? WHERE token_hash=? AND revoked_at IS NULL", time.Now().UTC().Format(time.RFC3339Nano), hash)
	return err
}

func (s *Store) Audit(ctx context.Context, actor, target int64, action string, details any) {
	b, _ := json.Marshal(details)
	_, _ = s.DB.ExecContext(ctx, "INSERT INTO audit_log(actor_user_id,target_user_id,action,details,occurred_at) VALUES(?,?,?,?,?)", actor, target, action, string(b), time.Now().UTC().Format(time.RFC3339Nano))
}
func (s *Store) ClearMetadata(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM metadata WHERE user_id=?", userID)
	return err
}
func (s *Store) PutMetadata(ctx context.Context, userID int64, kind string, id int, language string, payload []byte) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO metadata(user_id,media_type,tmdb_id,language,payload,updated_at) VALUES(?,?,?,?,?,?) ON CONFLICT(user_id,media_type,tmdb_id,language) DO UPDATE SET payload=excluded.payload,updated_at=excluded.updated_at", userID, kind, id, language, payload, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) GetMetadata(ctx context.Context, userID int64, kind string, id int, language string) ([]byte, error) {
	var b []byte
	err := s.DB.QueryRowContext(ctx, "SELECT payload FROM metadata WHERE user_id=? AND media_type=? AND tmdb_id=? AND language=?", userID, kind, id, language).Scan(&b)
	return b, err
}

type ExternalRating struct {
	MediaType           string `json:"mediaType"`
	TMDBID              int    `json:"tmdbId"`
	RottenTomatoesScore int    `json:"rottenTomatoesScore,omitempty"`
	Found               bool   `json:"found"`
	Provider            string `json:"provider,omitempty"`
	FetchedAt           string `json:"fetchedAt"`
}

func (s *Store) PutExternalRating(ctx context.Context, rating ExternalRating) error {
	fetchedAt := rating.FetchedAt
	if fetchedAt == "" {
		fetchedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO external_ratings(media_type,tmdb_id,rotten_tomatoes_score,found,provider,fetched_at)
		VALUES(?,?,?,?,?,?) ON CONFLICT(media_type,tmdb_id) DO UPDATE SET
		rotten_tomatoes_score=excluded.rotten_tomatoes_score,found=excluded.found,
		provider=excluded.provider,fetched_at=excluded.fetched_at`, rating.MediaType, rating.TMDBID,
		rating.RottenTomatoesScore, rating.Found, rating.Provider, fetchedAt)
	return err
}

func (s *Store) GetExternalRating(ctx context.Context, mediaType string, tmdbID int) (ExternalRating, error) {
	var rating ExternalRating
	var found int
	err := s.DB.QueryRowContext(ctx, `SELECT media_type,tmdb_id,rotten_tomatoes_score,found,provider,fetched_at
		FROM external_ratings WHERE media_type=? AND tmdb_id=?`, mediaType, tmdbID).Scan(
		&rating.MediaType, &rating.TMDBID, &rating.RottenTomatoesScore, &found, &rating.Provider, &rating.FetchedAt)
	rating.Found = found != 0
	return rating, err
}

func (s *Store) PutImageAsset(ctx context.Context, sourceURL, localName, contentType string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO image_assets(source_url,local_name,content_type,fetched_at)
		VALUES(?,?,?,?) ON CONFLICT(source_url) DO UPDATE SET local_name=excluded.local_name,
		content_type=excluded.content_type,fetched_at=excluded.fetched_at`, sourceURL, localName, contentType, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) ImageAsset(ctx context.Context, sourceURL string) (string, string, error) {
	var name, contentType string
	err := s.DB.QueryRowContext(ctx, "SELECT local_name,content_type FROM image_assets WHERE source_url=?", sourceURL).Scan(&name, &contentType)
	return name, contentType, err
}

type CacheEntry struct {
	Status               int
	ContentType          string
	Body                 []byte
	FetchedAt, ExpiresAt time.Time
}

func (s *Store) GetCache(ctx context.Context, key string) (CacheEntry, error) {
	var e CacheEntry
	var f, x string
	err := s.DB.QueryRowContext(ctx, "SELECT status,content_type,body,fetched_at,expires_at FROM http_cache WHERE cache_key=?", key).Scan(&e.Status, &e.ContentType, &e.Body, &f, &x)
	e.FetchedAt, _ = time.Parse(time.RFC3339Nano, f)
	e.ExpiresAt, _ = time.Parse(time.RFC3339Nano, x)
	return e, err
}
func (s *Store) PutCache(ctx context.Context, key string, e CacheEntry) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO http_cache(cache_key,status,content_type,body,fetched_at,expires_at) VALUES(?,?,?,?,?,?) ON CONFLICT(cache_key) DO UPDATE SET status=excluded.status,content_type=excluded.content_type,body=excluded.body,fetched_at=excluded.fetched_at,expires_at=excluded.expires_at", key, e.Status, e.ContentType, e.Body, e.FetchedAt.Format(time.RFC3339Nano), e.ExpiresAt.Format(time.RFC3339Nano))
	return err
}

type Progress struct {
	MediaType        string  `json:"mediaType"`
	TMDBID           int     `json:"tmdbId"`
	ShowTMDBID       int     `json:"showTmdbId,omitempty"`
	SeasonNumber     int     `json:"seasonNumber,omitempty"`
	EpisodeNumber    int     `json:"episodeNumber,omitempty"`
	PositionSeconds  float64 `json:"positionSeconds"`
	DurationSeconds  float64 `json:"durationSeconds"`
	Watched          bool    `json:"watched"`
	Started          bool    `json:"started"`
	StartedAt        *string `json:"startedAt,omitempty"`
	WatchedAt        *string `json:"watchedAt,omitempty"`
	AudioLanguage    string  `json:"audioLanguage,omitempty"`
	AudioTitle       string  `json:"audioTitle,omitempty"`
	SubtitleLanguage string  `json:"subtitleLanguage,omitempty"`
	SubtitleTitle    string  `json:"subtitleTitle,omitempty"`
	SubtitlesEnabled bool    `json:"subtitlesEnabled"`
	LastEventID      string  `json:"lastEventId,omitempty"`
	LastSessionID    string  `json:"lastSessionId,omitempty"`
	UpdatedAt        string  `json:"updatedAt"`
	ResumeSeconds    float64 `json:"resumeSeconds"`
}

func (s *Store) UpsertProgress(ctx context.Context, userID int64, p Progress, qualified bool) (Progress, error) {
	old, _ := s.GetProgress(ctx, userID, p.MediaType, p.TMDBID)
	started := old.Started || qualified
	// A first visit may persist language preferences, but it must not create a
	// resumable title until 30 seconds of continuous playback have elapsed.
	position := p.PositionSeconds
	duration := p.DurationSeconds
	if !started {
		position = 0
		duration = 0
	}
	watched := old.Watched || (started && duration > 0 && position/duration >= .95)
	var watchedAt any = nil
	if old.WatchedAt != nil {
		watchedAt = *old.WatchedAt
	} else if watched {
		watchedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var startedAt any
	if old.StartedAt != nil {
		startedAt = *old.StartedAt
	} else if started {
		startedAt = now
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO progress(user_id,media_type,tmdb_id,show_tmdb_id,season_number,episode_number,position_seconds,duration_seconds,watched,watched_at,audio_language,audio_title,subtitle_language,subtitle_title,subtitles_enabled,last_event_id,last_session_id,updated_at,started,started_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(user_id,media_type,tmdb_id) DO UPDATE SET show_tmdb_id=CASE WHEN excluded.show_tmdb_id>0 THEN excluded.show_tmdb_id ELSE progress.show_tmdb_id END,season_number=CASE WHEN excluded.season_number>0 THEN excluded.season_number ELSE progress.season_number END,episode_number=CASE WHEN excluded.episode_number>0 THEN excluded.episode_number ELSE progress.episode_number END,position_seconds=excluded.position_seconds,duration_seconds=excluded.duration_seconds,watched=excluded.watched,watched_at=excluded.watched_at,audio_language=COALESCE(excluded.audio_language,progress.audio_language),audio_title=COALESCE(excluded.audio_title,progress.audio_title),subtitle_language=COALESCE(excluded.subtitle_language,progress.subtitle_language),subtitle_title=COALESCE(excluded.subtitle_title,progress.subtitle_title),subtitles_enabled=excluded.subtitles_enabled,last_event_id=excluded.last_event_id,last_session_id=excluded.last_session_id,updated_at=excluded.updated_at,started=excluded.started,started_at=excluded.started_at`, userID, p.MediaType, p.TMDBID, p.ShowTMDBID, p.SeasonNumber, p.EpisodeNumber, position, duration, boolInt(watched), watchedAt, nullString(p.AudioLanguage), nullString(p.AudioTitle), nullString(p.SubtitleLanguage), nullString(p.SubtitleTitle), boolInt(p.SubtitlesEnabled), nullString(p.LastEventID), nullString(p.LastSessionID), now, boolInt(started), startedAt)
	if err != nil {
		return Progress{}, err
	}
	return s.GetProgress(ctx, userID, p.MediaType, p.TMDBID)
}
func (s *Store) GetProgress(ctx context.Context, userID int64, kind string, id int) (Progress, error) {
	var p Progress
	var watched, subs, started int
	var watchedAt, startedAt, audioLang, audioTitle, subLang, subTitle, eventID, sessionID sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT media_type,tmdb_id,show_tmdb_id,season_number,episode_number,position_seconds,duration_seconds,watched,watched_at,audio_language,audio_title,subtitle_language,subtitle_title,subtitles_enabled,last_event_id,last_session_id,updated_at,started,started_at FROM progress WHERE user_id=? AND media_type=? AND tmdb_id=?", userID, kind, id).Scan(&p.MediaType, &p.TMDBID, &p.ShowTMDBID, &p.SeasonNumber, &p.EpisodeNumber, &p.PositionSeconds, &p.DurationSeconds, &watched, &watchedAt, &audioLang, &audioTitle, &subLang, &subTitle, &subs, &eventID, &sessionID, &p.UpdatedAt, &started, &startedAt)
	p.Watched = watched == 1
	p.Started = started == 1
	p.SubtitlesEnabled = subs == 1
	p.AudioLanguage = audioLang.String
	p.AudioTitle = audioTitle.String
	p.SubtitleLanguage = subLang.String
	p.SubtitleTitle = subTitle.String
	p.LastEventID = eventID.String
	p.LastSessionID = sessionID.String
	if watchedAt.Valid {
		p.WatchedAt = &watchedAt.String
	}
	if startedAt.Valid {
		p.StartedAt = &startedAt.String
	}
	p.ResumeSeconds = p.PositionSeconds - 5
	if p.ResumeSeconds < 0 {
		p.ResumeSeconds = 0
	}
	return p, err
}
func (s *Store) ListProgress(ctx context.Context, userID int64) ([]Progress, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT media_type,tmdb_id FROM progress WHERE user_id=? AND (started=1 OR watched=1) ORDER BY updated_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys [][2]any
	for rows.Next() {
		var k string
		var id int
		rows.Scan(&k, &id)
		keys = append(keys, [2]any{k, id})
	}
	out := make([]Progress, 0, len(keys))
	for _, k := range keys {
		p, e := s.GetProgress(ctx, userID, k[0].(string), k[1].(int))
		if e == nil {
			if p.MediaType == "episode" && p.ShowTMDBID == 0 {
				s.enrichLegacyEpisodeContext(ctx, userID, &p)
			}
			out = append(out, p)
		}
	}
	return out, nil
}

// Older playback rows predate explicit show/season context. Episode metadata
// was already cached under a composite show/season key when the episode was
// opened, so it can be used to migrate those rows lazily and without a TMDB
// request on every series visit.
func (s *Store) enrichLegacyEpisodeContext(ctx context.Context, userID int64, p *Progress) {
	rows, err := s.DB.QueryContext(ctx, "SELECT media_type,tmdb_id,payload FROM metadata WHERE user_id=? AND media_type IN ('season','episode') ORDER BY CASE media_type WHEN 'season' THEN 0 ELSE 1 END", userID)
	if err != nil {
		return
	}
	showID, seasonNumber, episodeNumber := 0, 0, 0
	for rows.Next() {
		var mediaType string
		var cacheID int
		var payload []byte
		if rows.Scan(&mediaType, &cacheID, &payload) != nil {
			continue
		}
		if mediaType == "season" {
			var season struct {
				SeasonNumber int `json:"season_number"`
				Episodes     []struct {
					ID            int `json:"id"`
					EpisodeNumber int `json:"episode_number"`
				} `json:"episodes"`
			}
			if json.Unmarshal(payload, &season) != nil || season.SeasonNumber <= 0 || cacheID%1000 != season.SeasonNumber {
				continue
			}
			for _, episode := range season.Episodes {
				if episode.ID == p.TMDBID {
					showID, seasonNumber, episodeNumber = cacheID/1000, season.SeasonNumber, episode.EpisodeNumber
					break
				}
			}
			if showID > 0 && episodeNumber > 0 {
				break
			}
			continue
		}
		var episode struct {
			ID            int `json:"id"`
			SeasonNumber  int `json:"season_number"`
			EpisodeNumber int `json:"episode_number"`
		}
		if json.Unmarshal(payload, &episode) != nil || episode.ID != p.TMDBID || cacheID == episode.ID || episode.SeasonNumber <= 0 || episode.EpisodeNumber <= 0 {
			continue
		}
		remainder := episode.SeasonNumber*1000 + episode.EpisodeNumber
		if cacheID <= remainder || (cacheID-remainder)%1000000 != 0 {
			continue
		}
		showID = (cacheID - remainder) / 1000000
		if showID <= 0 {
			continue
		}
		seasonNumber, episodeNumber = episode.SeasonNumber, episode.EpisodeNumber
		break
	}
	_ = rows.Close()
	if showID <= 0 {
		return
	}
	p.ShowTMDBID = showID
	p.SeasonNumber = seasonNumber
	p.EpisodeNumber = episodeNumber
	_, _ = s.DB.ExecContext(ctx, "UPDATE progress SET show_tmdb_id=?,season_number=?,episode_number=? WHERE user_id=? AND media_type='episode' AND tmdb_id=?", showID, seasonNumber, episodeNumber, userID, p.TMDBID)
}

type MediaList struct {
	ID        int64  `json:"id"`
	SystemKey string `json:"systemKey,omitempty"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Icon      string `json:"icon"`
	ItemCount int    `json:"itemCount"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type MediaListItem struct {
	MediaType           string  `json:"mediaType"`
	TMDBID              int     `json:"tmdbId"`
	Title               string  `json:"title"`
	OriginalTitle       string  `json:"originalTitle,omitempty"`
	Overview            string  `json:"overview,omitempty"`
	PosterPath          string  `json:"posterPath,omitempty"`
	ReleaseDate         string  `json:"releaseDate,omitempty"`
	VoteAverage         float64 `json:"voteAverage,omitempty"`
	VoteCount           int     `json:"voteCount,omitempty"`
	Popularity          float64 `json:"popularity,omitempty"`
	Runtime             int     `json:"runtime,omitempty"`
	RottenTomatoesScore *int    `json:"rottenTomatoesScore,omitempty"`
	AddedAt             string  `json:"addedAt,omitempty"`
}

type ListMembership struct {
	ListID    int64  `json:"listId"`
	SystemKey string `json:"systemKey,omitempty"`
	MediaType string `json:"mediaType"`
	TMDBID    int    `json:"tmdbId"`
	AddedAt   string `json:"addedAt"`
}

var systemLists = []struct{ Key, Name, Color, Icon string }{
	{"favorites", "Favorites", "coral", "heart"},
	{"watch-later", "Watch Later", "amber", "bookmark"},
	{"watched", "Watched", "teal", "checks"},
}

func (s *Store) EnsureSystemLists(ctx context.Context, userID int64) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, list := range systemLists {
		if _, err := s.DB.ExecContext(ctx, `INSERT INTO media_lists(user_id,system_key,name,color,icon,created_at,updated_at) SELECT ?,?,?,?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM media_lists WHERE user_id=? AND system_key=?)`, userID, list.Key, list.Name, list.Color, list.Icon, now, now, userID, list.Key); err != nil {
			return err
		}
	}
	return nil
}

func scanMediaList(row interface{ Scan(...any) error }) (MediaList, error) {
	var list MediaList
	var key sql.NullString
	err := row.Scan(&list.ID, &key, &list.Name, &list.Color, &list.Icon, &list.ItemCount, &list.CreatedAt, &list.UpdatedAt)
	list.SystemKey = key.String
	return list, err
}

func (s *Store) MediaLists(ctx context.Context, userID int64) ([]MediaList, error) {
	if err := s.EnsureSystemLists(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT l.id,l.system_key,l.name,l.color,l.icon,count(i.tmdb_id),l.created_at,l.updated_at FROM media_lists l LEFT JOIN media_list_items i ON i.list_id=l.id WHERE l.user_id=? GROUP BY l.id ORDER BY CASE l.system_key WHEN 'favorites' THEN 0 WHEN 'watch-later' THEN 1 WHEN 'watched' THEN 2 ELSE 3 END,l.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lists := []MediaList{}
	for rows.Next() {
		list, err := scanMediaList(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	return lists, rows.Err()
}

func (s *Store) MediaListByID(ctx context.Context, userID, listID int64) (MediaList, error) {
	return scanMediaList(s.DB.QueryRowContext(ctx, `SELECT l.id,l.system_key,l.name,l.color,l.icon,count(i.tmdb_id),l.created_at,l.updated_at FROM media_lists l LEFT JOIN media_list_items i ON i.list_id=l.id WHERE l.user_id=? AND l.id=? GROUP BY l.id`, userID, listID))
}

func (s *Store) SystemMediaList(ctx context.Context, userID int64, key string) (MediaList, error) {
	if err := s.EnsureSystemLists(ctx, userID); err != nil {
		return MediaList{}, err
	}
	return scanMediaList(s.DB.QueryRowContext(ctx, `SELECT l.id,l.system_key,l.name,l.color,l.icon,count(i.tmdb_id),l.created_at,l.updated_at FROM media_lists l LEFT JOIN media_list_items i ON i.list_id=l.id WHERE l.user_id=? AND l.system_key=? GROUP BY l.id`, userID, key))
}

func (s *Store) CreateMediaList(ctx context.Context, userID int64, name string) (MediaList, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.DB.ExecContext(ctx, `INSERT INTO media_lists(user_id,name,color,icon,created_at,updated_at) VALUES(?,?,?,?,?,?)`, userID, name, "violet", "list", now, now)
	if err != nil {
		return MediaList{}, err
	}
	id, _ := res.LastInsertId()
	return s.MediaListByID(ctx, userID, id)
}

func (s *Store) RenameMediaList(ctx context.Context, userID, listID int64, name string) (MediaList, error) {
	result, err := s.DB.ExecContext(ctx, `UPDATE media_lists SET name=?,updated_at=? WHERE id=? AND user_id=? AND system_key IS NULL`, name, time.Now().UTC().Format(time.RFC3339Nano), listID, userID)
	if err != nil {
		return MediaList{}, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return MediaList{}, sql.ErrNoRows
	}
	return s.MediaListByID(ctx, userID, listID)
}

func (s *Store) DeleteMediaList(ctx context.Context, userID, listID int64) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM media_lists WHERE id=? AND user_id=? AND system_key IS NULL`, listID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) AddMediaListItem(ctx context.Context, userID, listID int64, item MediaListItem) error {
	if _, err := s.MediaListByID(ctx, userID, listID); err != nil {
		return err
	}
	addedAt := item.AddedAt
	if addedAt == "" {
		addedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO media_list_items(list_id,media_type,tmdb_id,title,original_title,overview,poster_path,release_date,vote_average,vote_count,popularity,runtime,added_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(list_id,media_type,tmdb_id) DO UPDATE SET title=excluded.title,original_title=excluded.original_title,overview=excluded.overview,poster_path=excluded.poster_path,release_date=excluded.release_date,vote_average=excluded.vote_average,vote_count=excluded.vote_count,popularity=excluded.popularity,runtime=excluded.runtime`, listID, item.MediaType, item.TMDBID, item.Title, item.OriginalTitle, item.Overview, item.PosterPath, item.ReleaseDate, item.VoteAverage, item.VoteCount, item.Popularity, item.Runtime, addedAt)
	return err
}

func (s *Store) MediaListContains(ctx context.Context, userID, listID int64, mediaType string, tmdbID int) (bool, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `SELECT count(*) FROM media_list_items i JOIN media_lists l ON l.id=i.list_id WHERE i.list_id=? AND i.media_type=? AND i.tmdb_id=? AND l.user_id=?`, listID, mediaType, tmdbID, userID).Scan(&count)
	return count > 0, err
}

func (s *Store) MediaListItemAddedAt(ctx context.Context, userID, listID int64, mediaType string, tmdbID int) (string, error) {
	var addedAt string
	err := s.DB.QueryRowContext(ctx, `SELECT i.added_at FROM media_list_items i JOIN media_lists l ON l.id=i.list_id WHERE i.list_id=? AND i.media_type=? AND i.tmdb_id=? AND l.user_id=?`, listID, mediaType, tmdbID, userID).Scan(&addedAt)
	return addedAt, err
}

func (s *Store) RemoveMediaListItem(ctx context.Context, userID, listID int64, mediaType string, tmdbID int) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM media_list_items WHERE list_id=? AND media_type=? AND tmdb_id=? AND EXISTS(SELECT 1 FROM media_lists WHERE id=? AND user_id=?)`, listID, mediaType, tmdbID, listID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) SetSystemListMembership(ctx context.Context, userID int64, key string, item MediaListItem, present bool) error {
	list, err := s.SystemMediaList(ctx, userID, key)
	if err != nil {
		return err
	}
	if present {
		return s.AddMediaListItem(ctx, userID, list.ID, item)
	}
	err = s.RemoveMediaListItem(ctx, userID, list.ID, item.MediaType, item.TMDBID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (s *Store) ListMemberships(ctx context.Context, userID int64) ([]ListMembership, error) {
	if err := s.EnsureSystemLists(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT i.list_id,l.system_key,i.media_type,i.tmdb_id,i.added_at FROM media_list_items i JOIN media_lists l ON l.id=i.list_id WHERE l.user_id=? ORDER BY i.added_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListMembership{}
	for rows.Next() {
		var item ListMembership
		var key sql.NullString
		if err := rows.Scan(&item.ListID, &key, &item.MediaType, &item.TMDBID, &item.AddedAt); err != nil {
			return nil, err
		}
		item.SystemKey = key.String
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MediaListItems(ctx context.Context, userID, listID int64, mediaType, sortBy, direction string) ([]MediaListItem, error) {
	if _, err := s.MediaListByID(ctx, userID, listID); err != nil {
		return nil, err
	}
	columns := map[string]string{"addedAt": "added_at", "title": "title COLLATE NOCASE", "releaseDate": "release_date", "rating": "CASE WHEN vote_count>=100 THEN vote_average ELSE -1 END", "popularity": "popularity", "runtime": "runtime"}
	column := columns[sortBy]
	if column == "" {
		column = "added_at"
	}
	order := "DESC"
	if strings.EqualFold(direction, "asc") {
		order = "ASC"
	}
	query := `SELECT media_type,tmdb_id,title,original_title,overview,poster_path,release_date,vote_average,vote_count,popularity,runtime,added_at FROM media_list_items WHERE list_id=?`
	args := []any{listID}
	if mediaType == "movie" || mediaType == "tv" {
		query += " AND media_type=?"
		args = append(args, mediaType)
	}
	query += " ORDER BY " + column + " " + order + ",added_at DESC"
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MediaListItem{}
	for rows.Next() {
		var item MediaListItem
		if err := rows.Scan(&item.MediaType, &item.TMDBID, &item.Title, &item.OriginalTitle, &item.Overview, &item.PosterPath, &item.ReleaseDate, &item.VoteAverage, &item.VoteCount, &item.Popularity, &item.Runtime, &item.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetEpisodeWatched(ctx context.Context, userID int64, p Progress, watched bool) (Progress, error) {
	old, _ := s.GetProgress(ctx, userID, "episode", p.TMDBID)
	if p.ShowTMDBID == 0 {
		p.ShowTMDBID = old.ShowTMDBID
	}
	if p.SeasonNumber == 0 {
		p.SeasonNumber = old.SeasonNumber
	}
	if p.EpisodeNumber == 0 {
		p.EpisodeNumber = old.EpisodeNumber
	}
	if p.DurationSeconds <= 0 {
		p.DurationSeconds = old.DurationSeconds
	}
	if watched {
		if p.DurationSeconds > 0 {
			p.PositionSeconds = p.DurationSeconds
		}
		p.Started, p.Watched = true, true
		now := time.Now().UTC().Format(time.RFC3339Nano)
		_, err := s.DB.ExecContext(ctx, `INSERT INTO progress(user_id,media_type,tmdb_id,show_tmdb_id,season_number,episode_number,position_seconds,duration_seconds,watched,watched_at,subtitles_enabled,updated_at,started,started_at) VALUES(?,?,?,?,?,?,?,?,1,?,0,?,1,?) ON CONFLICT(user_id,media_type,tmdb_id) DO UPDATE SET show_tmdb_id=excluded.show_tmdb_id,season_number=excluded.season_number,episode_number=excluded.episode_number,position_seconds=CASE WHEN progress.duration_seconds>0 THEN progress.duration_seconds ELSE progress.position_seconds END,watched=1,watched_at=COALESCE(progress.watched_at,excluded.watched_at),updated_at=excluded.updated_at,started=1,started_at=COALESCE(progress.started_at,excluded.started_at)`, userID, "episode", p.TMDBID, p.ShowTMDBID, p.SeasonNumber, p.EpisodeNumber, p.PositionSeconds, p.DurationSeconds, now, now, now)
		if err != nil {
			return Progress{}, err
		}
	} else {
		_, err := s.DB.ExecContext(ctx, `UPDATE progress SET watched=0,watched_at=NULL,updated_at=? WHERE user_id=? AND media_type='episode' AND tmdb_id=?`, time.Now().UTC().Format(time.RFC3339Nano), userID, p.TMDBID)
		if err != nil {
			return Progress{}, err
		}
	}
	return s.GetProgress(ctx, userID, "episode", p.TMDBID)
}

// ShowFullyWatched only returns true when metadata for every season currently
// advertised by TMDB is cached. This conservative rule prevents a partial UI
// load from accidentally marking a long-running show as complete. Season 0 is
// deliberately included, so released specials participate in completion.
func (s *Store) ShowFullyWatched(ctx context.Context, userID int64, showID int) (bool, error) {
	var showPayload []byte
	err := s.DB.QueryRowContext(ctx, `SELECT payload FROM metadata WHERE user_id=? AND media_type='tv' AND tmdb_id=? ORDER BY updated_at DESC LIMIT 1`, userID, showID).Scan(&showPayload)
	if err != nil {
		return false, err
	}
	var show struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
		} `json:"seasons"`
	}
	if err := json.Unmarshal(showPayload, &show); err != nil {
		return false, err
	}
	seasonNumbers := map[int]bool{}
	for _, season := range show.Seasons {
		if season.SeasonNumber >= 0 {
			seasonNumbers[season.SeasonNumber] = true
		}
	}
	if len(seasonNumbers) == 0 {
		return false, nil
	}
	released := map[int]bool{}
	loaded := map[int]bool{}
	rows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id,payload FROM metadata WHERE user_id=? AND media_type='season' AND tmdb_id>=? AND tmdb_id<? ORDER BY updated_at DESC`, userID, showID*1000, (showID+1)*1000)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	today := time.Now().UTC().Format("2006-01-02")
	for rows.Next() {
		var key int
		var payload []byte
		if err := rows.Scan(&key, &payload); err != nil {
			return false, err
		}
		seasonNumber := key - showID*1000
		if loaded[seasonNumber] {
			continue
		}
		var season struct {
			Episodes []struct {
				ID      int    `json:"id"`
				AirDate string `json:"air_date"`
			} `json:"episodes"`
		}
		if json.Unmarshal(payload, &season) != nil {
			continue
		}
		loaded[seasonNumber] = true
		for _, episode := range season.Episodes {
			if episode.ID > 0 && (episode.AirDate == "" || episode.AirDate <= today) {
				released[episode.ID] = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	for number := range seasonNumbers {
		if !loaded[number] {
			return false, nil
		}
	}
	if len(released) == 0 {
		return false, nil
	}
	progressRows, err := s.DB.QueryContext(ctx, `SELECT tmdb_id FROM progress WHERE user_id=? AND media_type='episode' AND show_tmdb_id=? AND watched=1`, userID, showID)
	if err != nil {
		return false, err
	}
	defer progressRows.Close()
	watched := map[int]bool{}
	for progressRows.Next() {
		var id int
		if progressRows.Scan(&id) == nil {
			watched[id] = true
		}
	}
	for id := range released {
		if !watched[id] {
			return false, nil
		}
	}
	return true, progressRows.Err()
}

func (s *Store) PutAvailability(ctx context.Context, kind string, id int, path string, available bool) error {
	_, err := s.DB.ExecContext(ctx, "INSERT INTO availability(media_type,tmdb_id,canonical_path,available,checked_at) VALUES(?,?,?,?,?) ON CONFLICT(media_type,tmdb_id) DO UPDATE SET canonical_path=excluded.canonical_path,available=excluded.available,checked_at=excluded.checked_at", kind, id, path, boolInt(available), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) AssociateTorrent(ctx context.Context, userID int64, kind string, id int, hash, category string) error {
	_, err := s.DB.ExecContext(ctx, "INSERT OR IGNORE INTO torrent_associations(user_id,media_type,tmdb_id,info_hash,category,created_at) VALUES(?,?,?,?,?,?)", userID, kind, id, hash, category, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) LatestTorrentAssociation(ctx context.Context, userID int64, kind string, id int) (TorrentAssociation, error) {
	associations, err := s.TorrentAssociations(ctx, userID, kind, id)
	if err != nil {
		return TorrentAssociation{}, err
	}
	if len(associations) == 0 {
		return TorrentAssociation{}, sql.ErrNoRows
	}
	return associations[0], nil
}

func (s *Store) TorrentAssociations(ctx context.Context, userID int64, kind string, id int) ([]TorrentAssociation, error) {
	alias := kind
	if kind == "show" {
		alias = "tv"
	} else if kind == "tv" {
		alias = "show"
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT info_hash, category
FROM torrent_associations
WHERE user_id=? AND tmdb_id=? AND media_type IN (?,?)
ORDER BY created_at DESC, rowid DESC`, userID, id, kind, alias)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	associations := []TorrentAssociation{}
	for rows.Next() {
		var association TorrentAssociation
		if err := rows.Scan(&association.InfoHash, &association.Category); err != nil {
			return nil, err
		}
		associations = append(associations, association)
	}
	return associations, rows.Err()
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func nullString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (s *Store) DebugCounts(ctx context.Context) (map[string]int, error) {
	out := map[string]int{}
	for _, t := range []string{"users", "metadata", "external_ratings", "image_assets", "http_cache", "availability", "progress", "media_lists", "media_list_items", "torrent_associations"} {
		var n int
		if err := s.DB.QueryRowContext(ctx, fmt.Sprintf("SELECT count(*) FROM %s", t)).Scan(&n); err != nil {
			return nil, err
		}
		out[t] = n
	}
	return out, nil
}
