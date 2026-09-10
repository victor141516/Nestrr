package auth

import (
	"context"
	"database/sql"
	"github.com/victor141516/Nestrr/internal/store"
	"path/filepath"
	"testing"
	"time"
)

func TestRefreshTokensRotateOnce(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	h, _ := HashPassword("a secure password")
	u, _ := s.CreateFirstAdmin(ctx, "admin", h, "en-US")
	m := Manager{Store: s, Secret: []byte("a secret that is definitely longer than thirty two bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour}
	first, err := m.Issue(ctx, u, 0)
	if err != nil {
		t.Fatal(err)
	}
	next, err := m.Rotate(ctx, first.RefreshToken)
	if err != nil || next.RefreshToken == first.RefreshToken {
		t.Fatalf("rotation failed: %v", err)
	}
	if _, err = m.Rotate(ctx, first.RefreshToken); err == nil {
		t.Fatal("used refresh token rotated twice")
	}
	claims, err := m.Parse(next.AccessToken)
	if err != nil || claims.UserID != u.ID || !claims.IsAdmin {
		t.Fatalf("access claims invalid: %#v %v", claims, err)
	}
}

func TestRefreshRotationWaitsForConcurrentWriter(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "auth.db")
	s, err := store.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	h, _ := HashPassword("a secure password")
	u, _ := s.CreateFirstAdmin(ctx, "admin", h, "en-US")
	m := Manager{Store: s, Secret: []byte("a secret that is definitely longer than thirty two bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour}
	first, err := m.Issue(ctx, u, 0)
	if err != nil {
		t.Fatal(err)
	}

	writer, err := sql.Open("sqlite", databasePath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	conn, err := writer.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(ctx, "INSERT INTO audit_log(action,occurred_at) VALUES(?,?)", "test.writer", time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(150 * time.Millisecond)
		_, _ = conn.ExecContext(context.Background(), "COMMIT")
		close(released)
	}()

	next, err := m.Rotate(ctx, first.RefreshToken)
	<-released
	if err != nil {
		t.Fatalf("rotation failed while a concurrent writer briefly held SQLite: %v", err)
	}
	if next.RefreshToken == "" || next.RefreshToken == first.RefreshToken {
		t.Fatal("rotation did not issue a new refresh token")
	}
}
