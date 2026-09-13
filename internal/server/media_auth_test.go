package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/config"
	"github.com/victor141516/Nestrr/internal/store"
)

func TestMediaProxyAuthentication(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "auth.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	hash, _ := auth.HashPassword("a secure test password")
	user, err := db.CreateFirstAdmin(context.Background(), "admin", hash, "en-US")
	if err != nil {
		t.Fatal(err)
	}
	m := &auth.Manager{Store: db, Secret: []byte("12345678901234567890123456789012"), AccessTTL: time.Hour, RefreshTTL: time.Hour}
	tokens, err := m.Issue(context.Background(), user, 0)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Store: db, Auth: m, Config: config.Config{MediaCookieDomain: "example.test", PublicOrigin: "https://nestrr.example.test"}}
	w := httptest.NewRecorder()
	if err := s.setMediaCookie(w, tokens); err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode || cookie.Domain != "example.test" || cookie.Path != "/" {
		t.Fatalf("unsafe media cookie attributes: %#v", cookie)
	}
	if _, err := m.Parse(cookie.Value); err == nil {
		t.Fatal("media grant authorized the Nestrr API")
	}
	claims, _ := m.Parse(tokens.AccessToken)
	claims.Audience = jwt.ClaimStrings{"nestrr-media"}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.Secret)
	for _, tc := range []struct {
		name, raw, accept string
		status            int
	}{
		{"valid", cookie.Value, "", 204},
		{"missing", "", "", 401},
		{"forged", cookie.Value + "x", "", 401},
		{"wrong audience", tokens.AccessToken, "", 401},
		{"expired", expired, "", 401},
		{"browser login", "", "text/html", 302},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/auth/media-check", nil)
			r.AddCookie(&http.Cookie{Name: mediaCookieName, Value: tc.raw})
			r.Header.Set("Accept", tc.accept)
			w := httptest.NewRecorder()
			s.mediaCheck(w, r)
			if w.Code != tc.status {
				t.Fatalf("got %d, want %d", w.Code, tc.status)
			}
			if tc.status == 302 && w.Header().Get("Location") != s.Config.PublicOrigin {
				t.Fatal("incorrect login redirect")
			}
		})
	}
	w = httptest.NewRecorder()
	s.clearMediaCookie(w)
	if w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("logout did not expire media cookie")
	}
	s.Config.MediaCookieDomain = ""
	w = httptest.NewRecorder()
	s.mediaCheck(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != http.StatusNotFound {
		t.Fatal("disabled proxy auth remained active")
	}
}
