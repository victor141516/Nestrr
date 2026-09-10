package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestSharedMetadataReusesLocalizedDocumentAcrossUsers(t *testing.T) {
	database := testStore(t)
	ctx := context.Background()
	first, err := database.CreateFirstAdmin(ctx, "first", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	second, err := database.CreateUser(ctx, first.ID, "second", "hash", "en-US")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.PutMetadata(ctx, "movie", 42, "es-es", []byte(`{"title":"Compartida"}`)); err != nil {
		t.Fatal(err)
	}
	for _, user := range []User{first, second} {
		payload, err := database.GetMetadata(ctx, "movie", 42, "ES_es")
		if err != nil || string(payload) != `{"title":"Compartida"}` {
			t.Fatalf("user %d could not reuse shared metadata: %s %v", user.ID, payload, err)
		}
	}
	if err := database.PutMetadata(ctx, "movie", 42, "en-US", []byte(`{"title":"Shared"}`)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.DB.QueryRow("SELECT count(*) FROM metadata WHERE media_type='movie' AND tmdb_id=42").Scan(&count); err != nil || count != 2 {
		t.Fatalf("language variants were not independently shared: count=%d err=%v", count, err)
	}
	if err := database.DeleteUser(ctx, first.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.GetMetadata(ctx, "movie", 42, "es-ES"); err != nil {
		t.Fatalf("deleting a user removed shared metadata: %v", err)
	}
}

func TestStoreUsesWALWithMultipleConnections(t *testing.T) {
	database := testStore(t)
	if maximum := database.DB.Stats().MaxOpenConnections; maximum != 8 {
		t.Fatalf("expected an eight-connection pool, got %d", maximum)
	}
	var journalMode string
	if err := database.DB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil || journalMode != "wal" {
		t.Fatalf("journal mode=%q err=%v", journalMode, err)
	}
	read, err := database.DB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	var count int
	if err := read.QueryRow("SELECT count(*) FROM metadata").Scan(&count); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := database.PutMetadata(ctx, "movie", 99, "en-US", []byte(`{"id":99}`)); err != nil {
		t.Fatalf("WAL writer was blocked by a reader: %v", err)
	}
}

func TestLegacyMetadataMigrationNormalizesAndKeepsNewestDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = raw.Exec(`CREATE TABLE metadata (
		user_id INTEGER NOT NULL,
		media_type TEXT NOT NULL,
		tmdb_id INTEGER NOT NULL,
		language TEXT NOT NULL,
		payload BLOB NOT NULL,
		updated_at TEXT NOT NULL,
		PRIMARY KEY(user_id,media_type,tmdb_id,language)
	);
	INSERT INTO metadata VALUES
		(1,'movie',7,'es-es','{"title":"Antigua"}','2026-01-01T00:00:00Z'),
		(2,'movie',7,'ES_es','{"title":"Nueva"}','2026-02-01T00:00:00Z'),
		(1,'movie',7,'en-US','{"title":"English"}','2026-01-01T00:00:00Z');`)
	if closeErr := raw.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	database, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	payload, err := database.GetMetadata(context.Background(), "movie", 7, "es-ES")
	if err != nil || string(payload) != `{"title":"Nueva"}` {
		t.Fatalf("migration selected wrong document: %s %v", payload, err)
	}
	var count int
	if err := database.DB.QueryRow("SELECT count(*) FROM metadata WHERE media_type='movie' AND tmdb_id=7").Scan(&count); err != nil || count != 2 {
		t.Fatalf("migration did not deduplicate normalized languages: count=%d err=%v", count, err)
	}
	hasUserID, err := database.tableHasColumn("metadata", "user_id")
	if err != nil || hasUserID {
		t.Fatalf("legacy user ownership remains: hasUserID=%v err=%v", hasUserID, err)
	}
}
