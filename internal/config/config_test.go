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

func TestPlaybackAPIKeyRequiresPlarrHost(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NESTRR_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("NESTRR_DATABASE_PATH", filepath.Join(root, "nestrr.db"))
	t.Setenv("NESTRR_IMAGE_CACHE_DIR", filepath.Join(root, "images"))
	t.Setenv("NESTRR_MOVIE_LIBRARY_ROOT", filepath.Join(root, "Movies"))
	t.Setenv("NESTRR_SHOW_LIBRARY_ROOT", filepath.Join(root, "Shows"))
	t.Setenv("NESTRR_PLARR_API_KEY", "secret")
	if _, err := Load(); err == nil {
		t.Fatal("accepted a playback API key without a Plarr host")
	}
	t.Setenv("NESTRR_PLARR_INTERNAL_HOST", "http://plarr:8090")
	if c, err := Load(); err != nil || c.PlarrAPIKey != "secret" {
		t.Fatalf("valid Plarr grant configuration rejected: %v", err)
	}
}
