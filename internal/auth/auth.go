package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/victor141516/Nestrr/internal/store"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID       int64 `json:"uid"`
	ActorAdminID int64 `json:"actor,omitempty"`
	IsAdmin      bool  `json:"admin"`
	jwt.RegisteredClaims
}
type Manager struct {
	Store                 *store.Store
	Secret                []byte
	AccessTTL, RefreshTTL time.Duration
}
type Tokens struct {
	AccessToken   string     `json:"accessToken"`
	ExpiresIn     int64      `json:"expiresIn"`
	RefreshToken  string     `json:"refreshToken,omitempty"`
	User          store.User `json:"user"`
	Impersonating bool       `json:"impersonating"`
}

func HashPassword(password string) (string, error) {
	if len(password) < 10 {
		return "", errors.New("password must contain at least 10 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (m *Manager) Issue(ctx context.Context, user store.User, actorAdminID int64) (Tokens, error) {
	now := time.Now().UTC()
	claims := Claims{UserID: user.ID, ActorAdminID: actorAdminID, IsAdmin: user.IsAdmin && actorAdminID == 0, RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(user.ID, 10), Issuer: "nestrr", Audience: jwt.ClaimStrings{"nestrr-client"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(m.AccessTTL)), ID: random(16)}}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.Secret)
	if err != nil {
		return Tokens{}, err
	}
	raw := random(32)
	session := store.RefreshSession{ID: random(16), TokenHash: tokenHash(raw), UserID: user.ID, ExpiresAt: now.Add(m.RefreshTTL).Format(time.RFC3339Nano)}
	if actorAdminID != 0 {
		session.ActorAdminID = &actorAdminID
	}
	if err = m.Store.SaveRefresh(ctx, session, ""); err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, ExpiresIn: int64(m.AccessTTL.Seconds()), RefreshToken: raw, User: user, Impersonating: actorAdminID != 0}, nil
}

func (m *Manager) Rotate(ctx context.Context, raw string) (Tokens, error) {
	old, err := m.Store.RefreshByHash(ctx, tokenHash(raw))
	if err != nil {
		return Tokens{}, err
	}
	exp, _ := time.Parse(time.RFC3339Nano, old.ExpiresAt)
	if time.Now().After(exp) {
		return Tokens{}, errors.New("refresh token expired")
	}
	u, err := m.Store.UserByID(ctx, old.UserID)
	if err != nil {
		return Tokens{}, err
	}
	actor := int64(0)
	if old.ActorAdminID != nil {
		actor = *old.ActorAdminID
	}
	now := time.Now().UTC()
	claims := Claims{UserID: u.ID, ActorAdminID: actor, IsAdmin: u.IsAdmin && actor == 0, RegisteredClaims: jwt.RegisteredClaims{Subject: strconv.FormatInt(u.ID, 10), Issuer: "nestrr", Audience: jwt.ClaimStrings{"nestrr-client"}, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(m.AccessTTL)), ID: random(16)}}
	access, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.Secret)
	if err != nil {
		return Tokens{}, err
	}
	nextRaw := random(32)
	next := store.RefreshSession{ID: random(16), TokenHash: tokenHash(nextRaw), UserID: u.ID, ExpiresAt: now.Add(m.RefreshTTL).Format(time.RFC3339Nano)}
	if actor != 0 {
		next.ActorAdminID = &actor
	}
	if err = m.Store.RotateRefresh(ctx, tokenHash(raw), next); err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, ExpiresIn: int64(m.AccessTTL.Seconds()), RefreshToken: nextRaw, User: u, Impersonating: actor != 0}, nil
}

func (m *Manager) Parse(raw string) (Claims, error) {
	var c Claims
	t, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.Secret, nil
	}, jwt.WithAudience("nestrr-client"), jwt.WithIssuer("nestrr"), jwt.WithExpirationRequired())
	if err != nil || !t.Valid {
		return c, errors.New("invalid access token")
	}
	return c, nil
}
func (m *Manager) Revoke(ctx context.Context, raw string) error {
	return m.Store.RevokeRefresh(ctx, tokenHash(raw))
}
func tokenHash(raw string) string { v := sha256.Sum256([]byte(raw)); return hex.EncodeToString(v[:]) }
func random(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
