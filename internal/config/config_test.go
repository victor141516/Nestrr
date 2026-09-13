package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultAccessTokenTTLIsTwoHours(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NESTRR_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("NESTRR_ACCESS_TOKEN_TTL", "")
	t.Setenv("NESTRR_DATABASE_PATH", filepath.Join(root, "nestrr.db"))
	t.Setenv("NESTRR_IMAGE_CACHE_DIR", filepath.Join(root, "images"))
	t.Setenv("NESTRR_MOVIE_LIBRARY_ROOT", filepath.Join(root, "Movies"))
	t.Setenv("NESTRR_SHOW_LIBRARY_ROOT", filepath.Join(root, "Shows"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccessTTL != 2*time.Hour {
		t.Fatalf("AccessTTL = %s; want 2h", cfg.AccessTTL)
	}
}

func TestMediaCookieRequiresSharedHTTPSHosts(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NESTRR_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("NESTRR_DATABASE_PATH", filepath.Join(root, "nestrr.db"))
	t.Setenv("NESTRR_IMAGE_CACHE_DIR", filepath.Join(root, "images"))
	t.Setenv("NESTRR_MOVIE_LIBRARY_ROOT", filepath.Join(root, "Movies"))
	t.Setenv("NESTRR_SHOW_LIBRARY_ROOT", filepath.Join(root, "Shows"))
	t.Setenv("NESTRR_PUBLIC_ORIGIN", "https://nestrr.example.test")
	t.Setenv("NESTRR_PLARR_HOST", "https://plarr.example.test")
	for _, domain := range []string{"test", "attacker.test", "example.test.evil"} {
		t.Setenv("NESTRR_MEDIA_COOKIE_DOMAIN", domain)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted invalid cookie domain %q", domain)
		}
	}
	t.Setenv("NESTRR_MEDIA_COOKIE_DOMAIN", ".example.test")
	if c, err := Load(); err != nil || c.MediaCookieDomain != "example.test" {
		t.Fatalf("valid shared domain rejected: %v", err)
	}
	t.Setenv("NESTRR_PLARR_HOST", "http://plarr.example.test")
	if _, err := Load(); err == nil {
		t.Fatal("accepted insecure player origin")
	}
}
