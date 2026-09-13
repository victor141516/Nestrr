package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/victor141516/Nestrr/internal/auth"
)

const mediaCookieName = "__Secure-nestrr_media"

// A separate audience makes this cookie useless for Nestrr's Bearer API.
// Only enable sharing when every subdomain under MediaCookieDomain is trusted.
func (s *Server) setMediaCookie(w http.ResponseWriter, tokens auth.Tokens) error {
	if s.Config.MediaCookieDomain == "" {
		return nil
	}
	c, err := s.Auth.Parse(tokens.AccessToken)
	if err != nil {
		return err
	}
	c.Audience = jwt.ClaimStrings{"nestrr-media"}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(s.Auth.Secret)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: mediaCookieName, Value: raw, Domain: s.Config.MediaCookieDomain, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: int(tokens.ExpiresIn)})
	return nil
}

func (s *Server) clearMediaCookie(w http.ResponseWriter) {
	if s.Config.MediaCookieDomain != "" {
		http.SetCookie(w, &http.Cookie{Name: mediaCookieName, Domain: s.Config.MediaCookieDomain, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	}
}

// Caddy forward_auth checks this endpoint before proxying public Plarr requests.
// Login and proactive access-token refresh renew the media cookie together.
func (s *Server) mediaCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if s.Config.MediaCookieDomain == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var c auth.Claims
	token, err := jwt.ParseWithClaims(cookie(r, mediaCookieName), &c, func(token *jwt.Token) (any, error) {
		return s.Auth.Secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithAudience("nestrr-media"), jwt.WithIssuer("nestrr"), jwt.WithExpirationRequired(), jwt.WithTimeFunc(time.Now))
	if err == nil && token.Valid {
		if _, err = s.Store.UserByID(r.Context(), c.UserID); err == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Redirect(w, r, s.Config.PublicOrigin, http.StatusFound)
		return
	}
	problem(w, http.StatusUnauthorized, "media_auth_required", "Sign in to Nestrr.")
}
