package tmdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

type Client struct {
	BaseURL, Token string
	TTL            time.Duration
	HTTP           *http.Client
	Store          *store.Store
	Log            *slog.Logger
	refreshing     sync.Map
}
type Response struct {
	Status      int
	ContentType string
	Body        []byte
	Stale       bool
}

func (c *Client) Get(ctx context.Context, path string, query url.Values, force bool) (Response, error) {
	if c.Token == "" && strings.Contains(c.BaseURL, "themoviedb.org") {
		return Response{}, errors.New("TMDB is not configured")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return Response{}, err
	}
	q := u.Query()
	for k, vals := range query {
		for _, v := range vals {
			q.Add(k, v)
		}
	}
	for k := range q {
		sort.Strings(q[k])
	}
	u.RawQuery = q.Encode()
	keyBytes := sha256.Sum256([]byte(u.String()))
	key := hex.EncodeToString(keyBytes[:])
	if !force {
		if cached, e := c.Store.GetCache(ctx, key); e == nil {
			if time.Now().Before(cached.ExpiresAt) {
				return Response{Status: cached.Status, ContentType: cached.ContentType, Body: cached.Body}, nil
			}
			go c.refresh(key, u.String())
			return Response{Status: cached.Status, ContentType: cached.ContentType, Body: cached.Body, Stale: true}, nil
		}
	}
	return c.fetch(ctx, key, u.String())
}
func (c *Client) refresh(key, u string) {
	if _, loaded := c.refreshing.LoadOrStore(key, true); loaded {
		return
	}
	defer c.refreshing.Delete(key)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := c.fetch(ctx, key, u); err != nil {
		c.Log.Warn("TMDB stale cache refresh failed", "operation", "tmdb.refresh", "error", err)
	}
}
func (c *Client) fetch(ctx context.Context, key, u string) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Response{}, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("TMDB request failed: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return Response{}, err
	}
	ct := res.Header.Get("Content-Type")
	if ct == "" {
		ct = "application/json"
	}
	now := time.Now().UTC()
	entry := store.CacheEntry{Status: res.StatusCode, ContentType: ct, Body: body, FetchedAt: now, ExpiresAt: now.Add(c.TTL)}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		if c.Store.PutCache(ctx, key, entry) == nil {
			// Continue from the persisted representation so cache misses and cache
			// hits follow the same response path.
			if persisted, cacheErr := c.Store.GetCache(ctx, key); cacheErr == nil {
				body = persisted.Body
				ct = persisted.ContentType
			}
		}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Response{Status: res.StatusCode, ContentType: ct, Body: body}, fmt.Errorf("TMDB returned status %d", res.StatusCode)
	}
	return Response{Status: res.StatusCode, ContentType: ct, Body: body}, nil
}
