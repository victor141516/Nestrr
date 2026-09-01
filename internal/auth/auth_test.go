package auth

import (
	"context"
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
