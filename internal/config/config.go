package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr               string
	DatabasePath             string
	ImageCacheDir            string
	MovieLibraryRoot         string
	ShowLibraryRoot          string
	TMDBBearerToken          string
	TMDBBaseURL              string
	TMDBImageBaseURL         string
	TMDBCacheTTL             time.Duration
	MDBListAPIKey            string
	MDBListBaseURL           string
	OMDbAPIKey               string
	OMDbBaseURL              string
	PlarrHost                string
	PlarrInternalHost        string
	JackettHost              string
	JackettAPIKey            string
	QBittorrentHost          string
	QBittorrentUsername      string
	QBittorrentPassword      string
	QBittorrentShowCategory  string
	QBittorrentMovieCategory string
	JWTSecret                string
	AccessTTL                time.Duration
	RefreshTTL               time.Duration
	PublicOrigin             string
	TrustedProxy             bool
	LogLevel                 slog.Level
	FrontendDir              string
	SecureCookies            bool
}

func Load() (Config, error) {
	cwd, _ := os.Getwd()
	c := Config{
		ListenAddr:               env("NESTRR_LISTEN_ADDR", ":8080"),
		DatabasePath:             env("NESTRR_DATABASE_PATH", filepath.Join(cwd, "data", "nestrr.db")),
		ImageCacheDir:            env("NESTRR_IMAGE_CACHE_DIR", filepath.Join(cwd, "data", "images")),
		MovieLibraryRoot:         env("NESTRR_MOVIE_LIBRARY_ROOT", filepath.Join(cwd, "media", "Movies")),
		ShowLibraryRoot:          env("NESTRR_SHOW_LIBRARY_ROOT", filepath.Join(cwd, "media", "Shows")),
		TMDBBearerToken:          strings.TrimSpace(os.Getenv("NESTRR_TMDB_BEARER_TOKEN")),
		TMDBBaseURL:              strings.TrimRight(env("NESTRR_TMDB_BASE_URL", "https://api.themoviedb.org/3"), "/"),
		TMDBImageBaseURL:         strings.TrimRight(env("NESTRR_TMDB_IMAGE_BASE_URL", "https://image.tmdb.org/t/p/original"), "/"),
		TMDBCacheTTL:             duration("NESTRR_TMDB_CACHE_TTL", time.Hour),
		MDBListAPIKey:            strings.TrimSpace(os.Getenv("NESTRR_MDBLIST_API_KEY")),
		MDBListBaseURL:           strings.TrimRight(env("NESTRR_MDBLIST_BASE_URL", "https://api.mdblist.com"), "/"),
		OMDbAPIKey:               strings.TrimSpace(os.Getenv("NESTRR_OMDB_API_KEY")),
		OMDbBaseURL:              strings.TrimRight(env("NESTRR_OMDB_BASE_URL", "https://www.omdbapi.com"), "/"),
		PlarrHost:                strings.TrimRight(strings.TrimSpace(os.Getenv("NESTRR_PLARR_HOST")), "/"),
		PlarrInternalHost:        strings.TrimRight(strings.TrimSpace(os.Getenv("NESTRR_PLARR_INTERNAL_HOST")), "/"),
		JackettHost:              strings.TrimRight(strings.TrimSpace(os.Getenv("NESTRR_JACKETT_HOST")), "/"),
		JackettAPIKey:            strings.TrimSpace(os.Getenv("NESTRR_JACKETT_API_KEY")),
		QBittorrentHost:          strings.TrimRight(strings.TrimSpace(os.Getenv("NESTRR_QBITTORRENT_HOST")), "/"),
		QBittorrentUsername:      os.Getenv("NESTRR_QBITTORRENT_USERNAME"),
		QBittorrentPassword:      os.Getenv("NESTRR_QBITTORRENT_PASSWORD"),
		QBittorrentShowCategory:  env("NESTRR_QBITTORRENT_SHOW_CATEGORY", "Shows"),
		QBittorrentMovieCategory: env("NESTRR_QBITTORRENT_MOVIE_CATEGORY", "Movies"),
		JWTSecret:                os.Getenv("NESTRR_JWT_SECRET"),
		AccessTTL:                duration("NESTRR_ACCESS_TOKEN_TTL", 2*time.Hour),
		RefreshTTL:               duration("NESTRR_REFRESH_TOKEN_TTL", 30*24*time.Hour),
		PublicOrigin:             strings.TrimRight(env("NESTRR_PUBLIC_ORIGIN", "http://localhost:8080"), "/"),
		TrustedProxy:             boolean("NESTRR_TRUSTED_PROXY", false),
		LogLevel:                 level("NESTRR_LOG_LEVEL", slog.LevelInfo),
		FrontendDir:              env("NESTRR_FRONTEND_DIR", filepath.Join(cwd, "web", "dist")),
	}
	if c.JWTSecret == "" {
		return c, errors.New("NESTRR_JWT_SECRET is required")
	}
	if c.PlarrInternalHost == "" {
		c.PlarrInternalHost = c.PlarrHost
	}
	if len(c.JWTSecret) < 32 {
		return c, errors.New("NESTRR_JWT_SECRET must contain at least 32 characters")
	}
	if u, err := url.Parse(c.PublicOrigin); err != nil || u.Scheme == "" || u.Host == "" {
		return c, errors.New("NESTRR_PUBLIC_ORIGIN must be an absolute URL")
	} else {
		c.SecureCookies = u.Scheme == "https"
	}
	if err := os.MkdirAll(filepath.Dir(c.DatabasePath), 0o750); err != nil {
		return c, err
	}
	for _, dir := range []string{c.ImageCacheDir, c.MovieLibraryRoot, c.ShowLibraryRoot} {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return c, err
		}
	}
	return c, nil
}
func level(name string, fallback slog.Level) slog.Level {
	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(env(name, fallback.String()))); err != nil {
		return fallback
	}
	return parsed
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}
func duration(name string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
func boolean(name string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
