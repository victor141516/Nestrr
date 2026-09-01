package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/config"
	"github.com/victor141516/Nestrr/internal/integrations"
	"github.com/victor141516/Nestrr/internal/keywords"
	"github.com/victor141516/Nestrr/internal/media"
	"github.com/victor141516/Nestrr/internal/ratings"
	"github.com/victor141516/Nestrr/internal/server"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)
	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		log.Error("database open failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := &http.Client{Timeout: 20 * time.Second}
	integration := integrations.New(cfg.JackettHost, cfg.JackettAPIKey, cfg.QBittorrentHost, cfg.QBittorrentUsername, cfg.QBittorrentPassword, cfg.QBittorrentShowCategory, cfg.QBittorrentMovieCategory, log)
	integration.Start(ctx)
	keywordCatalog := keywords.New(cfg.KeywordCachePath, client, log)
	keywordCatalog.Start(ctx)
	ratingService := &ratings.Service{MDBListBaseURL: cfg.MDBListBaseURL, MDBListAPIKey: cfg.MDBListAPIKey, OMDbBaseURL: cfg.OMDbBaseURL, OMDbAPIKey: cfg.OMDbAPIKey, HTTP: client, Store: db, Log: log}
	app := &server.Server{Config: cfg, Store: db, Auth: &auth.Manager{Store: db, Secret: []byte(cfg.JWTSecret), AccessTTL: cfg.AccessTTL, RefreshTTL: cfg.RefreshTTL}, TMDB: &tmdb.Client{BaseURL: cfg.TMDBBaseURL, Token: cfg.TMDBBearerToken, TTL: cfg.TMDBCacheTTL, HTTP: client, Store: db, Log: log}, Ratings: ratingService, Keywords: keywordCatalog, Media: media.Resolver{MoviesRoot: cfg.MovieLibraryRoot, ShowsRoot: cfg.ShowLibraryRoot}, Integrations: integration, Log: log, HTTP: client}
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: app.Handler(), ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 90 * time.Second}
	go func() {
		log.Info("Nestrr listening", "address", cfg.ListenAddr, "publicOrigin", cfg.PublicOrigin)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", "error", err)
			stop()
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
