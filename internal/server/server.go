package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/config"
	"github.com/victor141516/Nestrr/internal/integrations"
	"github.com/victor141516/Nestrr/internal/keywords"
	"github.com/victor141516/Nestrr/internal/media"
	"github.com/victor141516/Nestrr/internal/ratings"
	"github.com/victor141516/Nestrr/internal/recommendations"
	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

type Server struct {
	Config          config.Config
	Store           *store.Store
	Auth            *auth.Manager
	TMDB            *tmdb.Client
	Ratings         *ratings.Service
	Recommendations *recommendations.Service
	Keywords        *keywords.Catalog
	Media           media.Resolver
	Integrations    *integrations.Manager
	Log             *slog.Logger
	HTTP            *http.Client
}
type contextKey string

const claimsKey contextKey = "claims"

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, s.accessLog, s.securityHeaders, s.cors)
	r.Get("/healthz", s.health)
	r.Get("/api/v1/setup", s.setup)
	r.Get("/api/v1/images/cache", s.imageProxy)
	r.Post("/api/v1/auth/bootstrap", s.bootstrap)
	r.Post("/api/v1/auth/login", s.login)
	r.Post("/api/v1/auth/refresh", s.refresh)
	r.Post("/api/v1/auth/logout", s.logout)
	r.Get("/api/v1/auth/media-check", s.mediaCheck)
	r.Group(func(r chi.Router) {
		r.Use(s.authenticate)
		r.Get("/api/v1/me", s.me)
		r.Patch("/api/v1/me/settings", s.settings)
		r.Route("/api/v1/admin", func(r chi.Router) {
			r.Use(s.adminOnly)
			r.Get("/users", s.listUsers)
			r.Post("/users", s.createUser)
			r.Patch("/users/{id}", s.updateUser)
			r.Put("/users/{id}/password", s.resetPassword)
			r.Delete("/users/{id}", s.deleteUser)
			r.Post("/users/{id}/impersonate", s.impersonate)
			r.Post("/impersonation/stop", s.stopImpersonation)
			r.Get("/state", s.adminState)
			r.Get("/recommendations/catalog", s.recommendationCatalogStatus)
			r.Post("/recommendations/catalog/run", s.runRecommendationCatalog)
			r.Delete("/metadata", s.clearMetadata)
		})
		r.Get("/api/v1/catalog/local/search/{kind}", s.localCatalogSearch)
		r.Get("/api/v1/catalog/*", s.catalog)
		r.Get("/api/v1/recommendations", s.recommendations)
		r.Get("/api/v1/keywords/search", s.keywordSearch)
		r.Get("/api/v1/people/search", s.peopleSearch)
		r.Get("/api/v1/people/local-media", s.localMediaByPeople)
		r.Get("/api/v1/people/discover/movie", s.discoverMoviesByPeople)
		r.Get("/api/v1/companies/search", s.companySearch)
		r.Get("/api/v1/companies/local-media", s.localMediaByCompanies)
		r.Get("/api/v1/images/{asset}", s.image)
		r.Post("/api/v1/availability/check", s.checkAvailability)
		r.Get("/api/v1/availability/stream", s.streamAvailability)
		r.Get("/api/v1/library", s.library)
		r.Get("/api/v1/progress", s.listProgress)
		r.Get("/api/v1/progress/{kind}/{id}", s.getProgress)
		r.Post("/api/v1/playback/events", s.playbackEvent)
		r.Post("/api/v1/playback/navigation", s.playbackNavigation)
		r.Post("/api/v1/progress/{kind}/{id}/watched", s.setWatched)
		r.Post("/api/v1/progress/episodes/watched", s.setEpisodesWatched)
		r.Get("/api/v1/lists", s.listMediaLists)
		r.Post("/api/v1/lists", s.createMediaList)
		r.Patch("/api/v1/lists/{listID}", s.renameMediaList)
		r.Delete("/api/v1/lists/{listID}", s.deleteMediaList)
		r.Get("/api/v1/lists/{listID}/items", s.listMediaListItems)
		r.Post("/api/v1/lists/{listID}/items", s.addMediaListItem)
		r.Post("/api/v1/lists/{listID}/bulk-import", s.bulkImportMediaList)
		r.Delete("/api/v1/lists/{listID}/items/{kind}/{id}", s.removeMediaListItem)
		r.Get("/api/v1/list-memberships", s.listMemberships)
		r.Get("/api/v1/integrations/status", s.integrationStatus)
		r.Get("/api/v1/torrents/search", s.torrentSearch)
		r.Post("/api/v1/torrents/submit", s.torrentSubmit)
		r.Post("/api/v1/torrents/force-start", s.torrentForceStart)
	})
	r.Handle("/*", s.frontend())
	return r
}

func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.Log.Info("http request", "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "bytes", ww.BytesWritten(), "durationMs", time.Since(start).Milliseconds(), "requestId", middleware.GetReqID(r.Context()))
	})
}
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: https://image.tmdb.org; style-src 'self' 'unsafe-inline'; script-src 'self'; frame-src 'self' "+origin(s.Config.PlarrHost)+"; connect-src 'self' "+origin(s.Config.PlarrHost))
		next.ServeHTTP(w, r)
	})
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o := r.Header.Get("Origin")
		if o != "" && strings.EqualFold(strings.TrimRight(o, "/"), s.Config.PublicOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(h, "Bearer ") {
			problem(w, http.StatusUnauthorized, "authentication_required", "A valid bearer access token is required.")
			return
		}
		c, err := s.Auth.Parse(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			problem(w, http.StatusUnauthorized, "invalid_token", "The access token is invalid or expired.")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, c)))
	})
}
func (s *Server) adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := claims(r)
		if !c.IsAdmin && c.ActorAdminID == 0 {
			problem(w, 403, "administrator_required", "Administrator access is required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func claims(r *http.Request) auth.Claims {
	c, _ := r.Context().Value(claimsKey).(auth.Claims)
	return c
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.DB.PingContext(r.Context()); err != nil {
		problem(w, 503, "database_unavailable", "Database unavailable.")
		return
	}
	jsonResponse(w, 200, map[string]any{"status": "ok", "service": "nestrr"})
}
func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	exists, err := s.Store.AdminExists(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"needsBootstrap": !exists, "plarrConfigured": s.Config.PlarrHost != "", "rottenTomatoesEnabled": s.Ratings != nil && s.Ratings.Enabled()})
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Language string `json:"language"`
	Native   bool   `json:"native"`
}

var accentColors = map[string]bool{
	"#ff624f": true,
	"#f59e0b": true,
	"#84cc16": true,
	"#14b8a6": true,
	"#38bdf8": true,
	"#818cf8": true,
	"#a78bfa": true,
	"#f472b6": true,
}

func validateUsername(v string) error {
	v = strings.TrimSpace(v)
	if len(v) < 2 || len(v) > 40 {
		return errors.New("username must contain between 2 and 40 characters")
	}
	return nil
}
func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	if err := validateUsername(in.Username); err != nil {
		problem(w, 400, "invalid_username", err.Error())
		return
	}
	h, err := auth.HashPassword(in.Password)
	if err != nil {
		problem(w, 400, "invalid_password", err.Error())
		return
	}
	if in.Language == "" {
		in.Language = "en-US"
	}
	u, err := s.Store.CreateFirstAdmin(r.Context(), strings.TrimSpace(in.Username), h, in.Language)
	if err != nil {
		problem(w, 409, "bootstrap_unavailable", "The administrator has already been created.")
		return
	}
	t, err := s.Auth.Issue(r.Context(), u, 0)
	if err != nil {
		internal(w, err)
		return
	}
	s.sendTokens(w, t, in.Native)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	u, h, err := s.Store.UserByUsername(r.Context(), strings.TrimSpace(in.Username))
	if err != nil || !auth.CheckPassword(h, in.Password) {
		problem(w, 401, "invalid_credentials", "Username or password is incorrect.")
		return
	}
	t, err := s.Auth.Issue(r.Context(), u, 0)
	if err != nil {
		internal(w, err)
		return
	}
	s.sendTokens(w, t, in.Native)
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.validOrigin(r) {
		problem(w, 403, "origin_rejected", "Refresh requests must originate from the configured Nestrr origin.")
		return
	}
	raw := cookie(r, "nestrr_refresh")
	native := false
	if raw == "" {
		var in struct {
			RefreshToken string `json:"refreshToken"`
			Native       bool   `json:"native"`
		}
		if !decode(w, r, &in) {
			return
		}
		raw = in.RefreshToken
		native = in.Native
	}
	if raw == "" {
		problem(w, 401, "refresh_required", "A refresh token is required.")
		return
	}
	t, err := s.Auth.Rotate(r.Context(), raw)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefresh) {
			clearRefresh(w, s.Config.SecureCookies)
			s.clearMediaCookie(w)
			problem(w, 401, "invalid_refresh", "The refresh session is invalid or expired.")
			return
		}
		if s.Log != nil {
			s.Log.Error("refresh session rotation failed", "operation", "auth.refresh", "error", err)
		}
		problem(w, http.StatusServiceUnavailable, "refresh_unavailable", "The session could not be refreshed right now.")
		return
	}
	s.sendTokens(w, t, native)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	raw := cookie(r, "nestrr_refresh")
	if raw != "" {
		_ = s.Auth.Revoke(r.Context(), raw)
	}
	clearRefresh(w, s.Config.SecureCookies)
	s.clearMediaCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) sendTokens(w http.ResponseWriter, t auth.Tokens, native bool) {
	if !native {
		if err := s.setMediaCookie(w, t); err != nil {
			internal(w, err)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "nestrr_refresh", Value: t.RefreshToken, Path: "/api/v1/auth", HttpOnly: true, Secure: s.Config.SecureCookies, SameSite: http.SameSiteStrictMode, MaxAge: int(s.Config.RefreshTTL.Seconds())})
		t.RefreshToken = ""
	}
	jsonResponse(w, 200, t)
}
func (s *Server) validOrigin(r *http.Request) bool {
	o := strings.TrimRight(r.Header.Get("Origin"), "/")
	return o == "" || strings.EqualFold(o, s.Config.PublicOrigin)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	c := claims(r)
	u, err := s.Store.UserByID(r.Context(), c.UserID)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"user": u, "impersonating": c.ActorAdminID != 0, "actorAdminId": c.ActorAdminID})
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Language    *string `json:"language"`
		AccentColor *string `json:"accentColor"`
	}
	if !decode(w, r, &in) {
		return
	}
	u, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	language, accentColor := u.Language, u.AccentColor
	if in.Language != nil {
		language = strings.TrimSpace(*in.Language)
		if len(language) < 2 || len(language) > 20 {
			problem(w, 400, "invalid_language", "Language must be a BCP 47 language tag.")
			return
		}
	}
	if in.AccentColor != nil {
		accentColor = strings.ToLower(strings.TrimSpace(*in.AccentColor))
		if !accentColors[accentColor] {
			problem(w, 400, "invalid_accent_color", "Accent color is not supported.")
			return
		}
	}
	u, err = s.Store.SetPreferences(r.Context(), claims(r).UserID, language, accentColor)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, u)
}
func (s *Server) clearMetadata(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.ClearMetadata(r.Context()); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	u, err := s.Store.ListUsers(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"users": u})
}
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in credentials
	if !decode(w, r, &in) {
		return
	}
	if err := validateUsername(in.Username); err != nil {
		problem(w, 400, "invalid_username", err.Error())
		return
	}
	h, err := auth.HashPassword(in.Password)
	if err != nil {
		problem(w, 400, "invalid_password", err.Error())
		return
	}
	if in.Language == "" {
		in.Language = "en-US"
	}
	u, err := s.Store.CreateUser(r.Context(), claims(r).UserID, strings.TrimSpace(in.Username), h, in.Language)
	if err != nil {
		problem(w, 409, "username_unavailable", "That username is already in use.")
		return
	}
	jsonResponse(w, 201, u)
}
func pathID(r *http.Request) (int64, error) { return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64) }
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		problem(w, 400, "invalid_user", "Invalid user identifier.")
		return
	}
	var in struct{ Username, Language string }
	if !decode(w, r, &in) {
		return
	}
	existing, err := s.Store.UserByID(r.Context(), id)
	if err != nil {
		problem(w, 404, "user_not_found", "The user does not exist.")
		return
	}
	if strings.TrimSpace(in.Username) == "" {
		in.Username = existing.Username
	}
	if strings.TrimSpace(in.Language) == "" {
		in.Language = existing.Language
	}
	if err = validateUsername(in.Username); err != nil {
		problem(w, 400, "invalid_username", err.Error())
		return
	}
	u, err := s.Store.UpdateUser(r.Context(), claims(r).UserID, id, in.Username, in.Language)
	if err != nil {
		problem(w, 409, "update_failed", "The user could not be updated.")
		return
	}
	jsonResponse(w, 200, u)
}
func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		problem(w, 400, "invalid_user", "Invalid user identifier.")
		return
	}
	var in struct{ Password string }
	if !decode(w, r, &in) {
		return
	}
	h, err := auth.HashPassword(in.Password)
	if err != nil {
		problem(w, 400, "invalid_password", err.Error())
		return
	}
	if err = s.Store.ResetPassword(r.Context(), claims(r).UserID, id, h); err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		problem(w, 400, "invalid_user", "Invalid user identifier.")
		return
	}
	if err = s.Store.DeleteUser(r.Context(), claims(r).UserID, id); err != nil {
		problem(w, 400, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(204)
}
func (s *Server) impersonate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		problem(w, 400, "invalid_user", "Invalid user identifier.")
		return
	}
	u, err := s.Store.UserByID(r.Context(), id)
	if err != nil || u.IsAdmin {
		problem(w, 400, "invalid_impersonation", "Only a normal user can be impersonated.")
		return
	}
	actor := claims(r).UserID
	if claims(r).ActorAdminID != 0 {
		actor = claims(r).ActorAdminID
	}
	s.Store.Audit(r.Context(), actor, id, "user.impersonate", nil)
	t, err := s.Auth.Issue(r.Context(), u, actor)
	if err != nil {
		internal(w, err)
		return
	}
	s.sendTokens(w, t, false)
}
func (s *Server) stopImpersonation(w http.ResponseWriter, r *http.Request) {
	c := claims(r)
	if c.ActorAdminID == 0 {
		problem(w, 400, "not_impersonating", "This session is not impersonating a user.")
		return
	}
	u, err := s.Store.UserByID(r.Context(), c.ActorAdminID)
	if err != nil || !u.IsAdmin {
		problem(w, 403, "administrator_missing", "The administrator identity is unavailable.")
		return
	}
	s.Store.Audit(r.Context(), u.ID, c.UserID, "user.impersonation_stop", nil)
	t, err := s.Auth.Issue(r.Context(), u, 0)
	if err != nil {
		internal(w, err)
		return
	}
	s.sendTokens(w, t, false)
}
func (s *Server) adminState(w http.ResponseWriter, r *http.Request) {
	counts, err := s.Store.DebugCounts(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"counts": counts, "integrations": s.Integrations.Statuses(), "config": map[string]any{"movieLibraryRoot": s.Config.MovieLibraryRoot, "showLibraryRoot": s.Config.ShowLibraryRoot, "imageCacheDir": s.Config.ImageCacheDir, "plarrHost": s.Config.PlarrHost, "tmdbConfigured": s.Config.TMDBBearerToken != "", "movieCategory": s.Config.QBittorrentMovieCategory, "showCategory": s.Config.QBittorrentShowCategory}})
}

var catalogPath = regexp.MustCompile(`^(search/(movie|tv)|discover/(movie|tv)|genre/(movie|tv)/list|configuration/countries|certification/movie/list|(movie|tv)/\d+/(recommendations|keywords|credits)|collection/\d+(/recommendations)?|(movie|tv)/\d+|tv/\d+/season/\d+(/episode/\d+)?)$`)

func (s *Server) keywordSearch(w http.ResponseWriter, r *http.Request) {
	if s.Keywords == nil || !s.Keywords.Ready(r.Context()) {
		problem(w, http.StatusServiceUnavailable, "keyword_catalog_unavailable", "The keyword catalog is unavailable.")
		return
	}
	if rawIDs := strings.TrimSpace(r.URL.Query().Get("ids")); rawIDs != "" {
		parts := strings.Split(rawIDs, ",")
		if len(parts) > 50 {
			problem(w, http.StatusBadRequest, "too_many_keywords", "At most 50 keywords can be resolved at once.")
			return
		}
		ids := make([]int, 0, len(parts))
		for _, part := range parts {
			id, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || id <= 0 {
				problem(w, http.StatusBadRequest, "invalid_keyword_ids", "Keyword identifiers must be positive integers.")
				return
			}
			ids = append(ids, id)
		}
		results, err := s.Keywords.ByIDs(r.Context(), ids)
		if err != nil {
			internal(w, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": results})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) < 2 {
		problem(w, http.StatusBadRequest, "keyword_query_too_short", "Enter at least two characters.")
		return
	}
	limit := 12
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = min(parsed, 30)
		}
	}
	phase := r.URL.Query().Get("phase")
	var results []keywords.Entry
	var searchErr error
	switch phase {
	case "", "prefix":
		phase = "prefix"
		results, searchErr = s.Keywords.Prefix(r.Context(), query, limit)
	case "contains":
		results, searchErr = s.Keywords.Contains(r.Context(), query, limit)
	default:
		problem(w, http.StatusBadRequest, "invalid_keyword_phase", "Keyword search phase must be prefix or contains.")
		return
	}
	if searchErr != nil {
		internal(w, searchErr)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{
		"results":   results,
		"phase":     phase,
		"updatedAt": s.Keywords.UpdatedAt(r.Context()).UTC().Format(time.RFC3339),
	})
}

func cacheableCatalogDetail(detail bool, query url.Values) bool {
	return detail && query.Get("append_to_response") == ""
}

func (s *Server) catalog(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(chi.URLParam(r, "*"), "/")
	if !catalogPath.MatchString(path) {
		problem(w, 404, "catalog_route_not_found", "Unknown catalog operation.")
		return
	}
	c := claims(r)
	u, err := s.Store.UserByID(r.Context(), c.UserID)
	if err != nil {
		internal(w, err)
		return
	}
	q := r.URL.Query()
	effectiveLanguage := store.NormalizeLanguage(q.Get("language"))
	if q.Get("language") == "" {
		effectiveLanguage = store.NormalizeLanguage(u.Language)
	}
	q.Set("language", effectiveLanguage)
	refresh := q.Get("refresh") == "true"
	q.Del("refresh")
	parts := strings.Split(path, "/")
	kind, id, detail, showID, seasonNumber := "", 0, false, 0, 0
	if len(parts) >= 2 && (parts[0] == "movie" || parts[0] == "tv" || parts[0] == "collection") {
		kind = parts[0]
		id, _ = strconv.Atoi(parts[1])
		detail = len(parts) == 2
	}
	if kind == "tv" && len(parts) >= 4 && parts[2] == "season" {
		kind = "season"
		showID = id
		seasonNumber, _ = strconv.Atoi(parts[3])
		id = showID*1000 + seasonNumber
		detail = true
		if len(parts) >= 6 {
			kind = "episode"
			episode, _ := strconv.Atoi(parts[5])
			id = showID*1000000 + seasonNumber*1000 + episode
		}
	}
	// append_to_response creates a richer representation used by Explore (for
	// example US release/content ratings). It must not be satisfied by, or
	// overwrite, the ordinary shared localized detail document.
	cacheableDetail := cacheableCatalogDetail(detail, q)
	if cacheableDetail && !refresh {
		if b, e := s.Store.GetMetadata(r.Context(), kind, id, effectiveLanguage); e == nil {
			if kind == "movie" {
				s.storeMovieCompanies(r, id, b)
			}
			w.Header().Set("X-Nestrr-Metadata", "stored")
			jsonBytes(w, 200, "application/json", s.enrichCatalogRatings(r.Context(), path, b, false))
			return
		}
	}
	res, err := s.TMDB.Get(r.Context(), "/"+path, q, refresh)
	if err != nil {
		if res.Status == 0 {
			problem(w, 503, "tmdb_unavailable", "TMDB is unavailable or not configured.")
			return
		}
		jsonBytes(w, res.Status, res.ContentType, res.Body)
		return
	}
	body := res.Body
	if len(parts) == 3 && parts[0] == "genre" && (parts[1] == "movie" || parts[1] == "tv") && parts[2] == "list" {
		var document struct {
			Genres []store.Genre `json:"genres"`
		}
		if json.Unmarshal(res.Body, &document) == nil {
			_ = s.Store.PutGenreCatalog(r.Context(), parts[1], effectiveLanguage, document.Genres)
		}
	}
	if cacheableDetail && kind == "movie" {
		s.storeMovieCompanies(r, id, res.Body)
	}
	if len(parts) == 3 && (parts[0] == "movie" || parts[0] == "tv") && parts[2] == "credits" {
		if err := s.storeMediaCredits(r.Context(), parts[0], id, res.Body); err != nil {
			s.Log.Warn("TMDB credits could not be indexed", "operation", "credits.index", "mediaType", parts[0], "tmdbId", id, "error", err)
		}
	}
	if cacheableDetail {
		if s.Store.PutMetadata(r.Context(), kind, id, effectiveLanguage, res.Body) == nil {
			if persisted, persistedErr := s.Store.GetMetadata(r.Context(), kind, id, effectiveLanguage); persistedErr == nil {
				body = persisted
			}
		}
		if refresh && kind == "tv" {
			if err := s.refreshShowHierarchy(r.Context(), id, effectiveLanguage, res.Body); err != nil {
				s.Log.Warn("show child discovery failed", "operation", "metadata.refresh_hierarchy", "tmdbId", id, "error", err)
			}
		}
	}
	if res.Stale {
		w.Header().Set("X-Nestrr-Cache", "stale")
	}
	jsonBytes(w, res.Status, res.ContentType, s.enrichCatalogRatings(r.Context(), path, body, refresh))
}

func (s *Server) refreshShowHierarchy(ctx context.Context, showID int, language string, showBody []byte) error {
	var show struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
		} `json:"seasons"`
	}
	if err := json.Unmarshal(showBody, &show); err != nil {
		return err
	}
	for _, season := range show.Seasons {
		if season.SeasonNumber < 0 {
			continue
		}
		res, err := s.TMDB.Get(ctx, fmt.Sprintf("/tv/%d/season/%d", showID, season.SeasonNumber), url.Values{"language": {language}}, true)
		if err != nil {
			return err
		}
		seasonKey := showID*1000 + season.SeasonNumber
		if err = s.Store.PutMetadata(ctx, "season", seasonKey, language, res.Body); err != nil {
			return err
		}
	}
	return nil
}

type availabilityInput struct {
	Kind             string `json:"kind"`
	TMDBID           int    `json:"tmdbId"`
	Title            string `json:"title"`
	OriginalTitle    string `json:"originalTitle"`
	OriginalLanguage string `json:"originalLanguage"`
	Year             int    `json:"year"`
	Show             string `json:"show"`
	OriginalShow     string `json:"originalShow"`
	ShowTMDBID       int    `json:"showTmdbId"`
	Season           int    `json:"season"`
	Episode          int    `json:"episode"`
}
type availabilityOutput struct {
	availabilityInput
	Available           bool                      `json:"available"`
	Path                string                    `json:"path"`
	RelativePath        string                    `json:"relativePath,omitempty"`
	PlayerURL           string                    `json:"playerUrl,omitempty"`
	OriginalDownloadURL string                    `json:"originalDownloadUrl,omitempty"`
	Error               string                    `json:"error,omitempty"`
	Torrent             *integrations.TorrentInfo `json:"torrent,omitempty"`
}

func (s *Server) resolveAvailability(ctx context.Context, in availabilityInput) availabilityOutput {
	return s.resolveAvailabilityForUser(ctx, 0, in)
}

func (s *Server) resolveAvailabilityForUser(ctx context.Context, userID int64, in availabilityInput) availabilityOutput {
	out := availabilityOutput{availabilityInput: in}
	var p string
	var ok bool
	var err error
	switch in.Kind {
	case "movie":
		var canonicalTitle string
		canonicalTitle, err = s.englishCanonicalName(ctx, in)
		if err == nil {
			p, ok, err = s.Media.Movie(canonicalTitle, in.Year)
		}
	case "show":
		var canonicalShow string
		canonicalShow, err = s.englishCanonicalName(ctx, in)
		if err == nil {
			p, ok, err = s.Media.Show(canonicalShow)
		}
	case "episode":
		var canonicalShow string
		canonicalShow, err = s.englishCanonicalName(ctx, in)
		if err == nil {
			p, ok, err = s.Media.Episode(canonicalShow, in.Season, in.Episode)
		}
	default:
		err = errors.New("unsupported media kind")
	}
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Available = ok
	out.Path = p
	base := filepath.Dir(s.Config.MovieLibraryRoot)
	if rel, e := filepath.Rel(base, p); e == nil {
		out.RelativePath = filepath.ToSlash(rel)
	}
	if ok && in.Kind != "show" && s.Config.PlarrHost != "" {
		encoded := base64.RawURLEncoding.EncodeToString([]byte(out.RelativePath))
		out.PlayerURL = s.Config.PlarrHost + "/#/play/" + encoded
		out.OriginalDownloadURL = s.Config.PlarrHost + "/api/v1/media/" + encoded + "/download/original"
	}
	_ = s.Store.PutAvailability(ctx, in.Kind, in.TMDBID, p, ok)
	if !ok && userID > 0 {
		associations, associationErr := s.Store.TorrentAssociations(ctx, userID, in.Kind, in.TMDBID)
		if associationErr == nil {
			for _, association := range associations {
				if torrent, found, torrentErr := s.Integrations.Torrent(ctx, association.InfoHash); torrentErr == nil && found {
					out.Torrent = torrentWithMostProgress(out.Torrent, torrent)
				}
			}
		}
	}
	return out
}

// englishCanonicalName keeps the filesystem contract independent from the
// user's metadata language. Original TMDB names are already canonical for
// English-language media; other media are resolved through an English TMDB
// document and remain cached by the TMDB client.
func (s *Server) englishCanonicalName(ctx context.Context, in availabilityInput) (string, error) {
	isOriginalEnglish := strings.EqualFold(strings.Split(in.OriginalLanguage, "-")[0], "en")
	if isOriginalEnglish {
		if in.Kind == "movie" && strings.TrimSpace(in.OriginalTitle) != "" {
			return in.OriginalTitle, nil
		}
		if (in.Kind == "show" || in.Kind == "episode") && strings.TrimSpace(in.OriginalShow) != "" {
			return in.OriginalShow, nil
		}
	}

	kind, id := "movie", in.TMDBID
	if in.Kind == "show" {
		kind = "tv"
	} else if in.Kind == "episode" {
		kind, id = "tv", in.ShowTMDBID
	}
	if id <= 0 || s.TMDB == nil {
		return "", errors.New("canonical English TMDB identity is unavailable")
	}
	res, err := s.TMDB.Get(ctx, fmt.Sprintf("/%s/%d", kind, id), url.Values{"language": {"en-US"}}, false)
	if err != nil {
		return "", fmt.Errorf("could not resolve canonical English TMDB name: %w", err)
	}
	var doc struct {
		Title string `json:"title"`
		Name  string `json:"name"`
	}
	if err = json.Unmarshal(res.Body, &doc); err != nil {
		return "", fmt.Errorf("invalid canonical English TMDB document: %w", err)
	}
	name := doc.Title
	if kind == "tv" {
		name = doc.Name
	}
	if strings.TrimSpace(name) == "" {
		return "", errors.New("canonical English TMDB name is empty")
	}
	return name, nil
}
func (s *Server) checkAvailability(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []availabilityInput `json:"items"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Items) > 200 {
		problem(w, 400, "too_many_items", "At most 200 items may be checked at once.")
		return
	}
	out := make([]availabilityOutput, len(in.Items))
	for i, v := range in.Items {
		out[i] = s.resolveAvailabilityForUser(r.Context(), claims(r).UserID, v)
	}
	jsonResponse(w, 200, map[string]any{"items": out})
}
func (s *Server) streamAvailability(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("items")
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		problem(w, 400, "invalid_items", "Items must be base64url-encoded JSON.")
		return
	}
	var items []availabilityInput
	if json.Unmarshal(b, &items) != nil || len(items) > 200 {
		problem(w, 400, "invalid_items", "Invalid availability items.")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "stream_unsupported", "Streaming is unavailable.")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	for _, v := range items {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		out := s.resolveAvailabilityForUser(r.Context(), claims(r).UserID, v)
		data, _ := json.Marshal(out)
		fmt.Fprintf(w, "event: availability\ndata: %s\n\n", data)
		flusher.Flush()
	}
}
func (s *Server) library(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Media.Enumerate()
	if err != nil {
		internal(w, err)
		return
	}
	u, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	type libraryItem struct {
		media.LibraryEntry
		TMDBID    int             `json:"tmdbId,omitempty"`
		MediaType string          `json:"mediaType,omitempty"`
		Metadata  json.RawMessage `json:"metadata,omitempty"`
	}
	items := make([]libraryItem, 0, len(entries))
	for _, entry := range entries {
		item := libraryItem{LibraryEntry: entry}
		path := "/search/movie"
		query := entry.Title
		if entry.Kind == "episode" {
			path, query = "/search/tv", entry.Show
			item.MediaType = "tv"
		} else {
			item.MediaType = "movie"
		}
		q := url.Values{"language": {u.Language}, "query": {query}}
		if entry.Year > 0 {
			q.Set("primary_release_year", strconv.Itoa(entry.Year))
		}
		if result, e := s.TMDB.Get(r.Context(), path, q, false); e == nil {
			var doc struct {
				Results []json.RawMessage `json:"results"`
			}
			if json.Unmarshal(result.Body, &doc) == nil && len(doc.Results) > 0 {
				var identity struct {
					ID int `json:"id"`
				}
				_ = json.Unmarshal(doc.Results[0], &identity)
				item.TMDBID, item.Metadata = identity.ID, doc.Results[0]
				if entry.Kind == "movie" {
					_ = s.Store.PutAvailability(r.Context(), "movie", identity.ID, filepath.Join(s.Config.MovieLibraryRoot, filepath.Base(entry.Path)), true)
				}
			}
		}
		items = append(items, item)
	}
	documents := make([]ratingDocument, 0, len(items))
	for index := range items {
		if len(items[index].Metadata) > 0 && items[index].TMDBID > 0 {
			documents = append(documents, ratingDocument{MediaType: items[index].MediaType, Payload: &items[index].Metadata})
		}
	}
	s.enrichRatingDocuments(r.Context(), documents, false)
	jsonResponse(w, 200, map[string]any{"items": items})
}

func (s *Server) listProgress(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.ListProgress(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": p})
}
func (s *Server) getProgress(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "invalid_item", "Invalid catalog identifier.")
		return
	}
	p, err := s.Store.GetProgress(r.Context(), claims(r).UserID, chi.URLParam(r, "kind"), id)
	if errors.Is(err, sql.ErrNoRows) {
		jsonResponse(w, 200, map[string]any{"mediaType": chi.URLParam(r, "kind"), "tmdbId": id, "positionSeconds": 0, "resumeSeconds": 0, "watched": false})
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, p)
}

type playbackEvent struct {
	Source           string  `json:"source"`
	Version          int     `json:"version"`
	EventID          string  `json:"eventId"`
	SessionID        string  `json:"sessionId"`
	Type             string  `json:"type"`
	MediaPath        string  `json:"mediaPath"`
	MediaType        string  `json:"mediaType"`
	TMDBID           int     `json:"tmdbId"`
	ShowTMDBID       int     `json:"showTmdbId"`
	SeasonNumber     int     `json:"seasonNumber"`
	EpisodeNumber    int     `json:"episodeNumber"`
	PositionSeconds  float64 `json:"positionSeconds"`
	DurationSeconds  float64 `json:"durationSeconds"`
	Paused           bool    `json:"paused"`
	OccurredAt       string  `json:"occurredAt"`
	AudioLanguage    string  `json:"audioLanguage"`
	AudioTitle       string  `json:"audioTitle"`
	SubtitleLanguage string  `json:"subtitleLanguage"`
	SubtitleTitle    string  `json:"subtitleTitle"`
	SubtitlesEnabled bool    `json:"subtitlesEnabled"`
	Qualified        bool    `json:"qualified"`
}

func (s *Server) playbackEvent(w http.ResponseWriter, r *http.Request) {
	var e playbackEvent
	if !decode(w, r, &e) {
		return
	}
	if e.Source != "plarr" || e.Version != 1 || e.EventID == "" || e.SessionID == "" || e.TMDBID <= 0 || (e.MediaType != "movie" && e.MediaType != "episode") {
		problem(w, 400, "invalid_event", "Invalid Plarr event envelope.")
		return
	}
	allowed := map[string]bool{"ready": true, "opened": true, "play": true, "pause": true, "seek": true, "heartbeat": true, "ended": true, "closed": true, "page-hidden": true, "audio-track-changed": true, "subtitle-track-changed": true, "subtitles-enabled": true, "subtitles-disabled": true, "automatic-rendition-changed": true, "manual-quality-changed": true, "next-requested": true, "previous-requested": true, "playback-error": true}
	if !allowed[e.Type] {
		problem(w, 400, "invalid_event_type", "Unknown playback event type.")
		return
	}
	p := store.Progress{MediaType: e.MediaType, TMDBID: e.TMDBID, ShowTMDBID: e.ShowTMDBID, SeasonNumber: e.SeasonNumber, EpisodeNumber: e.EpisodeNumber, PositionSeconds: e.PositionSeconds, DurationSeconds: e.DurationSeconds, AudioLanguage: e.AudioLanguage, AudioTitle: e.AudioTitle, SubtitleLanguage: e.SubtitleLanguage, SubtitleTitle: e.SubtitleTitle, SubtitlesEnabled: e.SubtitlesEnabled, LastEventID: e.EventID, LastSessionID: e.SessionID}
	p, err := s.Store.UpsertProgress(r.Context(), claims(r).UserID, p, e.Qualified)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			jsonResponse(w, 200, map[string]any{"duplicate": true})
			return
		}
		internal(w, err)
		return
	}
	if p.Watched {
		if p.MediaType == "movie" {
			s.syncCompletedTitle(r.Context(), claims(r).UserID, "movie", p.TMDBID)
		} else if p.ShowTMDBID > 0 {
			if complete, completeErr := s.Store.ShowFullyWatched(r.Context(), claims(r).UserID, p.ShowTMDBID); completeErr == nil && complete {
				s.syncCompletedTitle(r.Context(), claims(r).UserID, "tv", p.ShowTMDBID)
			}
		}
	}
	jsonResponse(w, 200, p)
}

type navigationInput struct {
	Direction string `json:"direction"`
	ShowID    int    `json:"showId"`
	ShowName  string `json:"showName"`
	Season    int    `json:"season"`
	Episode   int    `json:"episode"`
}

func (s *Server) playbackNavigation(w http.ResponseWriter, r *http.Request) {
	var in navigationInput
	if !decode(w, r, &in) {
		return
	}
	if (in.Direction != "next" && in.Direction != "previous") || in.ShowID <= 0 || in.Season <= 0 || in.Episode <= 0 || in.ShowName == "" {
		problem(w, 400, "invalid_navigation", "A valid show episode and direction are required.")
		return
	}
	u, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	type episode struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		SeasonNumber  int    `json:"season_number"`
		EpisodeNumber int    `json:"episode_number"`
		Overview      string `json:"overview"`
		StillPath     string `json:"still_path"`
	}
	type seasonDoc struct {
		Episodes []episode `json:"episodes"`
	}
	load := func(season int) (seasonDoc, error) {
		res, e := s.TMDB.Get(r.Context(), fmt.Sprintf("/tv/%d/season/%d", in.ShowID, season), url.Values{"language": {u.Language}}, false)
		if e != nil {
			return seasonDoc{}, e
		}
		var d seasonDoc
		e = json.Unmarshal(res.Body, &d)
		return d, e
	}
	season := in.Season
	doc, err := load(season)
	if err != nil {
		problem(w, 502, "navigation_failed", "The adjacent episode could not be resolved.")
		return
	}
	idx := -1
	for i, e := range doc.Episodes {
		if e.EpisodeNumber == in.Episode {
			idx = i
			break
		}
	}
	step := 1
	if in.Direction == "previous" {
		step = -1
	}
	target := idx + step
	for attempts := 0; attempts < 100; attempts++ {
		if idx >= 0 && target >= 0 && target < len(doc.Episodes) {
			item := doc.Episodes[target]
			item.SeasonNumber = season
			a := s.resolveAvailabilityForUser(r.Context(), claims(r).UserID, availabilityInput{Kind: "episode", TMDBID: item.ID, Show: in.ShowName, ShowTMDBID: in.ShowID, Season: season, Episode: item.EpisodeNumber})
			jsonResponse(w, 200, map[string]any{"item": item, "showId": in.ShowID, "showName": in.ShowName, "availability": a})
			return
		}
		season += step
		if season <= 0 {
			break
		}
		doc, err = load(season)
		if err != nil {
			break
		}
		if len(doc.Episodes) == 0 {
			continue
		}
		idx = 0
		if step > 0 {
			target = 0
		} else {
			target = len(doc.Episodes) - 1
		}
	}
	jsonResponse(w, 200, map[string]any{"endOfSeries": true})
}
func (s *Server) setWatched(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "invalid_item", "Invalid catalog identifier.")
		return
	}
	var in struct {
		Watched       bool                `json:"watched"`
		ShowTMDBID    int                 `json:"showTmdbId"`
		SeasonNumber  int                 `json:"seasonNumber"`
		EpisodeNumber int                 `json:"episodeNumber"`
		Item          store.MediaListItem `json:"item"`
	}
	if !decode(w, r, &in) {
		return
	}
	kind := chi.URLParam(r, "kind")
	if kind == "episode" {
		p, progressErr := s.Store.SetEpisodeWatched(r.Context(), claims(r).UserID, store.Progress{MediaType: "episode", TMDBID: id, ShowTMDBID: in.ShowTMDBID, SeasonNumber: in.SeasonNumber, EpisodeNumber: in.EpisodeNumber}, in.Watched)
		if progressErr != nil {
			internal(w, progressErr)
			return
		}
		if in.Watched && p.ShowTMDBID > 0 {
			if complete, completeErr := s.Store.ShowFullyWatched(r.Context(), claims(r).UserID, p.ShowTMDBID); completeErr == nil && complete {
				s.syncCompletedTitle(r.Context(), claims(r).UserID, "tv", p.ShowTMDBID)
			}
		}
		jsonResponse(w, 200, p)
		return
	}
	if kind != "movie" && kind != "tv" {
		problem(w, 400, "invalid_item", "Only movies, series, and episodes can be marked watched.")
		return
	}
	item := in.Item
	item.MediaType, item.TMDBID = kind, id
	if item.Title == "" {
		item = s.mediaListSnapshot(r.Context(), claims(r).UserID, kind, id)
	}
	if err := s.Store.SetSystemListMembership(r.Context(), claims(r).UserID, "watched", item, in.Watched); err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"watched": in.Watched})
}

func (s *Server) setEpisodesWatched(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Watched  bool `json:"watched"`
		Episodes []struct {
			TMDBID, ShowTMDBID, SeasonNumber, EpisodeNumber int
		} `json:"episodes"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Episodes) == 0 || len(in.Episodes) > 500 {
		problem(w, 400, "invalid_episodes", "Between 1 and 500 episodes are required.")
		return
	}
	items := make([]store.Progress, 0, len(in.Episodes))
	for _, episode := range in.Episodes {
		if episode.TMDBID <= 0 {
			problem(w, 400, "invalid_episode", "Every episode needs a catalog identifier.")
			return
		}
		p, err := s.Store.SetEpisodeWatched(r.Context(), claims(r).UserID, store.Progress{MediaType: "episode", TMDBID: episode.TMDBID, ShowTMDBID: episode.ShowTMDBID, SeasonNumber: episode.SeasonNumber, EpisodeNumber: episode.EpisodeNumber}, in.Watched)
		if err != nil {
			internal(w, err)
			return
		}
		items = append(items, p)
	}
	if in.Watched && len(items) > 0 && items[0].ShowTMDBID > 0 {
		if complete, err := s.Store.ShowFullyWatched(r.Context(), claims(r).UserID, items[0].ShowTMDBID); err == nil && complete {
			s.syncCompletedTitle(r.Context(), claims(r).UserID, "tv", items[0].ShowTMDBID)
		}
	}
	jsonResponse(w, 200, map[string]any{"items": items})
}

func validListItem(item store.MediaListItem) bool {
	return (item.MediaType == "movie" || item.MediaType == "tv") && item.TMDBID > 0
}

func (s *Server) listMediaLists(w http.ResponseWriter, r *http.Request) {
	lists, err := s.Store.MediaLists(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"lists": lists})
}

func (s *Server) createMediaList(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 60 {
		problem(w, 400, "invalid_list_name", "List names must contain between 1 and 60 characters.")
		return
	}
	list, err := s.Store.CreateMediaList(r.Context(), claims(r).UserID, in.Name)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			problem(w, 409, "list_name_exists", "A custom list with that name already exists.")
			return
		}
		internal(w, err)
		return
	}
	jsonResponse(w, 201, list)
}

func listIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "listID"), 10, 64)
}

func (s *Server) renameMediaList(w http.ResponseWriter, r *http.Request) {
	id, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 60 {
		problem(w, 400, "invalid_list_name", "List names must contain between 1 and 60 characters.")
		return
	}
	list, err := s.Store.RenameMediaList(r.Context(), claims(r).UserID, id, in.Name)
	if errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_not_found", "The custom list does not exist.")
		return
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			problem(w, 409, "list_name_exists", "A custom list with that name already exists.")
			return
		}
		internal(w, err)
		return
	}
	jsonResponse(w, 200, list)
}

func (s *Server) deleteMediaList(w http.ResponseWriter, r *http.Request) {
	id, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	err = s.Store.DeleteMediaList(r.Context(), claims(r).UserID, id)
	if errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_not_found", "The custom list does not exist.")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMediaListItems(w http.ResponseWriter, r *http.Request) {
	id, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	list, err := s.Store.MediaListByID(r.Context(), claims(r).UserID, id)
	if errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_not_found", "The list does not exist.")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	items, err := s.Store.MediaListItems(r.Context(), claims(r).UserID, id, r.URL.Query().Get("mediaType"), r.URL.Query().Get("sort"), r.URL.Query().Get("direction"))
	if err != nil {
		internal(w, err)
		return
	}
	// Resolve the current shared translation first. The denormalized list row
	// remains a fallback only and never overrides the user's active language.
	for index := range items {
		snapshot := s.mediaListSnapshot(r.Context(), claims(r).UserID, items[index].MediaType, items[index].TMDBID)
		fallbackTitle := fmt.Sprintf("%s %d", strings.ToUpper(items[index].MediaType), items[index].TMDBID)
		if snapshot.Title == fallbackTitle {
			continue
		}
		snapshot.AddedAt = items[index].AddedAt
		items[index] = snapshot
	}
	s.enrichMediaListRatings(r.Context(), items)
	jsonResponse(w, 200, map[string]any{"list": list, "items": items})
}

func (s *Server) addMediaListItem(w http.ResponseWriter, r *http.Request) {
	id, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	var item store.MediaListItem
	if !decode(w, r, &item) {
		return
	}
	if !validListItem(item) {
		problem(w, 400, "invalid_list_item", "A movie or series catalog item is required.")
		return
	}
	if item.AddedAt != "" {
		addedAt, err := time.Parse(time.RFC3339Nano, item.AddedAt)
		if err != nil {
			problem(w, 400, "invalid_added_at", "The original inclusion date is invalid.")
			return
		}
		item.AddedAt = addedAt.UTC().Format(time.RFC3339Nano)
	}
	if item.Title == "" {
		item = s.mediaListSnapshot(r.Context(), claims(r).UserID, item.MediaType, item.TMDBID)
	}
	if err := s.Store.AddMediaListItem(r.Context(), claims(r).UserID, id, item); errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_not_found", "The list does not exist.")
		return
	} else if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 201, map[string]any{"added": true})
}

func (s *Server) removeMediaListItem(w http.ResponseWriter, r *http.Request) {
	listID, err := listIDParam(r)
	if err != nil {
		problem(w, 400, "invalid_list", "Invalid list identifier.")
		return
	}
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		problem(w, 400, "invalid_item", "Invalid catalog identifier.")
		return
	}
	kind := chi.URLParam(r, "kind")
	if kind != "movie" && kind != "tv" {
		problem(w, 400, "invalid_list_item", "Only movies and series belong to lists.")
		return
	}
	err = s.Store.RemoveMediaListItem(r.Context(), claims(r).UserID, listID, kind, id)
	if errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "list_item_not_found", "The item is not in this list.")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMemberships(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListMemberships(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"items": items})
}

func (s *Server) mediaListSnapshot(ctx context.Context, userID int64, kind string, id int) store.MediaListItem {
	item := store.MediaListItem{MediaType: kind, TMDBID: id, Title: fmt.Sprintf("%s %d", strings.ToUpper(kind), id)}
	u, err := s.Store.UserByID(ctx, userID)
	if err != nil {
		return item
	}
	payload, err := s.Store.GetMetadata(ctx, kind, id, u.Language)
	if err != nil {
		return item
	}
	var doc struct {
		Title          string  `json:"title"`
		Name           string  `json:"name"`
		OriginalTitle  string  `json:"original_title"`
		OriginalName   string  `json:"original_name"`
		Overview       string  `json:"overview"`
		PosterPath     string  `json:"poster_path"`
		ReleaseDate    string  `json:"release_date"`
		FirstAirDate   string  `json:"first_air_date"`
		VoteAverage    float64 `json:"vote_average"`
		VoteCount      int     `json:"vote_count"`
		Popularity     float64 `json:"popularity"`
		Runtime        int     `json:"runtime"`
		EpisodeRunTime []int   `json:"episode_run_time"`
	}
	if json.Unmarshal(payload, &doc) != nil {
		return item
	}
	item.Title = doc.Title
	if item.Title == "" {
		item.Title = doc.Name
	}
	item.OriginalTitle = doc.OriginalTitle
	if item.OriginalTitle == "" {
		item.OriginalTitle = doc.OriginalName
	}
	item.Overview = doc.Overview
	item.PosterPath = doc.PosterPath
	item.ReleaseDate = doc.ReleaseDate
	if item.ReleaseDate == "" {
		item.ReleaseDate = doc.FirstAirDate
	}
	item.VoteAverage = doc.VoteAverage
	item.VoteCount = doc.VoteCount
	item.Popularity = doc.Popularity
	item.Runtime = doc.Runtime
	if item.Runtime == 0 && len(doc.EpisodeRunTime) > 0 {
		item.Runtime = doc.EpisodeRunTime[0]
	}
	return item
}

func (s *Server) syncCompletedTitle(ctx context.Context, userID int64, kind string, id int) {
	item := s.mediaListSnapshot(ctx, userID, kind, id)
	_ = s.Store.SetSystemListMembership(ctx, userID, "watched", item, true)
	_ = s.Store.SetSystemListMembership(ctx, userID, "watch-later", item, false)
}

func (s *Server) integrationStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.Integrations.Statuses()
	plarr := integrations.Status{Configured: s.Config.PlarrHost != "", CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if plarr.Configured {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.Config.PlarrInternalHost+"/healthz", nil)
		res, err := s.HTTP.Do(req)
		plarr.Healthy = err == nil && res.StatusCode/100 == 2
		if res != nil {
			res.Body.Close()
		}
		if !plarr.Healthy {
			plarr.Message = "unreachable"
		}
	} else {
		plarr.Message = "not configured"
	}
	statuses["plarr"] = plarr
	jsonResponse(w, 200, map[string]any{"integrations": statuses, "categories": map[string]string{"movie": s.Config.QBittorrentMovieCategory, "show": s.Config.QBittorrentShowCategory}})
}
func (s *Server) torrentSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		problem(w, 400, "query_required", "A search query is required.")
		return
	}
	results, err := s.Integrations.Search(r.Context(), q)
	if err != nil {
		s.Log.Error("Jackett search failed", "operation", "jackett.search", "error", err)
		problem(w, 502, "torrent_search_failed", "The torrent search service could not complete the request.")
		return
	}
	jsonResponse(w, 200, map[string]any{"results": results, "query": q})
}
func (s *Server) torrentSubmit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URL, Category, MediaType string
		TMDBID                   int
	}
	if !decode(w, r, &in) {
		return
	}
	if in.URL == "" || in.Category == "" {
		problem(w, 400, "invalid_submission", "A result URL and category are required.")
		return
	}
	hash, err := s.Integrations.Submit(r.Context(), integrations.SubmitRequest{URL: in.URL, Category: in.Category})
	if err != nil {
		if errors.Is(err, integrations.ErrTorrentDownload) {
			s.Log.Error("Jackett torrent download failed", "operation", "jackett.download")
			problem(w, 502, "torrent_download_failed", "Jackett could not download this torrent. Try another search result.")
			return
		}
		s.Log.Error("qBittorrent submission failed", "operation", "qbittorrent.submit", "error", err)
		problem(w, 502, "torrent_submission_failed", "qBittorrent did not accept the torrent.")
		return
	}
	if hash != "" && in.TMDBID > 0 {
		_ = s.Store.AssociateTorrent(r.Context(), claims(r).UserID, canonicalTorrentMediaType(in.MediaType), in.TMDBID, hash, in.Category)
	}
	jsonResponse(w, 202, map[string]any{"accepted": true, "infoHash": hash})
}

func canonicalTorrentMediaType(kind string) string {
	if kind == "tv" {
		return "show"
	}
	return kind
}

func canForceStartTorrent(state string) bool {
	return state == "stalledDL" || state == "queuedDL"
}

func torrentWithMostProgress(current *integrations.TorrentInfo, candidate integrations.TorrentInfo) *integrations.TorrentInfo {
	if current != nil && current.Progress >= candidate.Progress {
		return current
	}
	selected := candidate
	return &selected
}

func (s *Server) torrentForceStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MediaType string
		TMDBID    int
	}
	if !decode(w, r, &in) {
		return
	}
	if in.TMDBID <= 0 || !regexp.MustCompile(`^(movie|tv|show|episode)$`).MatchString(in.MediaType) {
		problem(w, 400, "invalid_torrent", "A valid media item is required.")
		return
	}
	associations, err := s.Store.TorrentAssociations(r.Context(), claims(r).UserID, canonicalTorrentMediaType(in.MediaType), in.TMDBID)
	if err == nil && len(associations) == 0 {
		err = sql.ErrNoRows
	}
	if errors.Is(err, sql.ErrNoRows) {
		problem(w, 404, "torrent_not_found", "No torrent is associated with this media item.")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}
	hashes := make([]string, 0, len(associations))
	hasForceable := false
	for _, association := range associations {
		torrent, found, torrentErr := s.Integrations.Torrent(r.Context(), association.InfoHash)
		if torrentErr != nil {
			problem(w, 502, "torrent_status_failed", "qBittorrent could not report an associated torrent.")
			return
		}
		if !found {
			continue
		}
		hashes = append(hashes, association.InfoHash)
		hasForceable = hasForceable || canForceStartTorrent(torrent.State)
	}
	if len(hashes) == 0 {
		problem(w, 404, "torrent_not_found", "The associated torrent no longer exists in qBittorrent.")
		return
	}
	if !hasForceable {
		problem(w, 409, "torrent_not_forceable", "Only a stalled or queued download can be force started.")
		return
	}
	if err := s.Integrations.ForceStartMany(r.Context(), hashes); err != nil {
		problem(w, 502, "torrent_force_start_failed", "qBittorrent could not force start this torrent.")
		return
	}
	jsonResponse(w, 202, map[string]any{"accepted": true, "forced": len(hashes)})
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	asset := chi.URLParam(r, "asset")
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+\.(jpg|jpeg|png|webp)$`).MatchString(asset) {
		problem(w, 400, "invalid_asset", "Invalid asset name.")
		return
	}
	path := filepath.Join(s.Config.ImageCacheDir, asset)
	if _, err := os.Stat(path); err == nil {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", imageETag(asset))
		http.ServeFile(w, r, path)
		return
	}
	problem(w, 404, "asset_not_found", "Cached image not found.")
}

var tmdbImageSizes = map[string]struct{}{
	"w92": {}, "w154": {}, "w185": {}, "w300": {}, "w342": {},
	"w500": {}, "w780": {}, "w1280": {}, "h632": {}, "original": {},
}

func (s *Server) imageProxy(w http.ResponseWriter, r *http.Request) {
	upstream := r.URL.Query().Get("path")
	size := r.URL.Query().Get("size")
	if size == "" {
		size = "w500"
	}
	if _, ok := tmdbImageSizes[size]; !ok {
		problem(w, http.StatusBadRequest, "invalid_image_size", "Invalid artwork size.")
		return
	}
	name, err := s.cacheImage(r.Context(), upstream, size)
	if err != nil {
		problem(w, http.StatusBadGateway, "image_unavailable", "The artwork could not be cached.")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("ETag", imageETag(name))
	http.ServeFile(w, r, filepath.Join(s.Config.ImageCacheDir, name))
}

func (s *Server) cacheImage(ctx context.Context, upstream, size string) (string, error) {
	if !strings.HasPrefix(upstream, "/") {
		return "", errors.New("invalid TMDB image path")
	}
	cacheKey := "tmdb-image:" + size + ":" + upstream
	// Preserve existing original-image cache entries while giving every resized
	// variant its own immutable asset.
	if size == "original" {
		cacheKey = upstream
	}
	sum := sha256.Sum256([]byte(cacheKey))
	ext := strings.ToLower(filepath.Ext(upstream))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		ext = ".jpg"
	}
	name := hex.EncodeToString(sum[:]) + ext
	target := filepath.Join(s.Config.ImageCacheDir, name)
	if _, err := os.Stat(target); err == nil {
		if _, _, recordErr := s.Store.ImageAsset(ctx, cacheKey); errors.Is(recordErr, sql.ErrNoRows) {
			_ = s.Store.PutImageAsset(ctx, cacheKey, name, mime.TypeByExtension(ext))
		}
		return name, nil
	}
	if err := os.MkdirAll(s.Config.ImageCacheDir, 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tmdbImageURL(s.Config.TMDBImageBaseURL, size, upstream), nil)
	if err != nil {
		return "", err
	}
	res, err := s.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("image upstream returned HTTP %d", res.StatusCode)
	}
	f, err := os.CreateTemp(s.Config.ImageCacheDir, name+".*.tmp")
	if err != nil {
		return "", err
	}
	temporary := f.Name()
	defer os.Remove(temporary)
	written, copyErr := io.Copy(f, io.LimitReader(res.Body, (20<<20)+1))
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if written > 20<<20 {
		return "", errors.New("image exceeds cache size limit")
	}
	if err := os.Rename(temporary, target); err != nil {
		// Another request may have completed the same immutable variant first.
		if _, statErr := os.Stat(target); statErr != nil {
			return "", err
		}
	}
	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(ext)
	}
	if err := s.Store.PutImageAsset(ctx, cacheKey, name, contentType); err != nil {
		return "", err
	}
	return name, nil
}

func tmdbImageURL(base, size, upstream string) string {
	base = strings.TrimRight(base, "/")
	lastSlash := strings.LastIndex(base, "/")
	if lastSlash >= 0 {
		if _, isSize := tmdbImageSizes[base[lastSlash+1:]]; isSize {
			base = base[:lastSlash]
		}
	}
	return base + "/" + size + upstream
}

func imageETag(asset string) string {
	return `"` + strings.TrimSuffix(asset, filepath.Ext(asset)) + `"`
}

func (s *Server) frontend() http.Handler {
	root := os.DirFS(s.Config.FrontendDir)
	fileServer := http.FileServer(http.FS(root))
	index, indexErr := fs.ReadFile(root, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(r.URL.Path)), "/")
		if path != "." {
			if f, err := fs.Stat(root, path); err == nil && !f.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		if indexErr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(index)
	})
}
func origin(v string) string {
	u, err := url.Parse(v)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
func cookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}
func clearRefresh(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: "nestrr_refresh", Path: "/api/v1/auth", HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		problem(w, 400, "invalid_json", "The request body is invalid.")
		return false
	}
	return true
}
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func jsonBytes(w http.ResponseWriter, status int, contentType string, b []byte) {
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(b)
}
func problem(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
func internal(w http.ResponseWriter, err error) {
	slog.Error("request failed", "error", err)
	problem(w, 500, "internal_error", "The request could not be completed.")
}
