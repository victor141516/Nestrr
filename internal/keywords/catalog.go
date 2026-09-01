package keywords

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultExportBase = "https://files.tmdb.org/p/exports"
	refreshInterval   = 7 * 24 * time.Hour
	maxExportBytes    = 16 << 20
)

type Entry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	normalized string
}

type Catalog struct {
	path       string
	exportBase string
	http       *http.Client
	log        *slog.Logger
	now        func() time.Time

	mu        sync.RWMutex
	entries   []Entry
	byID      map[int]Entry
	updatedAt time.Time
}

func New(path string, client *http.Client, log *slog.Logger) *Catalog {
	if client == nil {
		client = http.DefaultClient
	}
	if log == nil {
		log = slog.Default()
	}
	return &Catalog{
		path:       path,
		exportBase: defaultExportBase,
		http:       client,
		log:        log,
		now:        time.Now,
		byID:       make(map[int]Entry),
	}
}

// Start loads the last valid export before serving requests, refreshes it when
// it is older than one week, and checks daily so transient failures retry soon.
func (c *Catalog) Start(ctx context.Context) {
	loaded := c.loadFile() == nil
	if !loaded || c.stale() {
		if err := c.refresh(ctx); err != nil {
			c.log.Warn("TMDB keyword export refresh failed", "operation", "keywords.refresh", "error", err)
		}
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if c.stale() {
					if err := c.refresh(ctx); err != nil {
						c.log.Warn("TMDB keyword export refresh failed", "operation", "keywords.refresh", "error", err)
					}
				}
			}
		}
	}()
}

func (c *Catalog) Ready() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries) > 0
}

func (c *Catalog) UpdatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updatedAt
}

func (c *Catalog) Prefix(query string, limit int) []Entry {
	query = normalize(query)
	if query == "" || limit <= 0 {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	start := sort.Search(len(c.entries), func(index int) bool {
		return c.entries[index].normalized >= query
	})
	results := make([]Entry, 0, limit)
	for index := start; index < len(c.entries) && len(results) < limit; index++ {
		entry := c.entries[index]
		if !strings.HasPrefix(entry.normalized, query) {
			break
		}
		results = append(results, publicEntry(entry))
	}
	return results
}

func (c *Catalog) Contains(query string, limit int) []Entry {
	query = normalize(query)
	if query == "" || limit <= 0 {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	results := make([]Entry, 0, limit)
	for _, entry := range c.entries {
		if strings.HasPrefix(entry.normalized, query) || !strings.Contains(entry.normalized, query) {
			continue
		}
		results = append(results, publicEntry(entry))
		if len(results) == limit {
			break
		}
	}
	return results
}

func (c *Catalog) ByIDs(ids []int) []Entry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	results := make([]Entry, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		if entry, ok := c.byID[id]; ok {
			results = append(results, publicEntry(entry))
		}
	}
	return results
}

func publicEntry(entry Entry) Entry {
	entry.normalized = ""
	return entry
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (c *Catalog) stale() bool {
	c.mu.RLock()
	updatedAt := c.updatedAt
	c.mu.RUnlock()
	return updatedAt.IsZero() || c.now().Sub(updatedAt) >= refreshInterval
}

func (c *Catalog) loadFile() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}
	entries, err := decodeExport(data)
	if err != nil {
		return err
	}
	info, err := os.Stat(c.path)
	if err != nil {
		return err
	}
	c.replace(entries, info.ModTime())
	return nil
}

func (c *Catalog) refresh(ctx context.Context) error {
	data, err := c.downloadLatest(ctx)
	if err != nil {
		return err
	}
	entries, err := decodeExport(data)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(c.path), "tmdb-keywords-*.json.gz")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err = temporary.Write(data); err == nil {
		err = temporary.Sync()
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(temporaryPath, c.path); err != nil {
		return err
	}
	now := c.now()
	c.replace(entries, now)
	c.log.Info("TMDB keyword export loaded", "operation", "keywords.refresh", "entries", len(entries))
	return nil
}

func (c *Catalog) replace(entries []Entry, updatedAt time.Time) {
	byID := make(map[int]Entry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	c.mu.Lock()
	c.entries = entries
	c.byID = byID
	c.updatedAt = updatedAt
	c.mu.Unlock()
}

func (c *Catalog) downloadLatest(ctx context.Context) ([]byte, error) {
	var lastError error
	now := c.now().UTC()
	for daysAgo := 0; daysAgo < 7; daysAgo++ {
		date := now.AddDate(0, 0, -daysAgo).Format("01_02_2006")
		endpoint := strings.TrimRight(c.exportBase, "/") + "/keyword_ids_" + date + ".json.gz"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		res, err := c.http.Do(req)
		if err != nil {
			lastError = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, maxExportBytes+1))
		res.Body.Close()
		if readErr != nil {
			lastError = readErr
			continue
		}
		if res.StatusCode == http.StatusNotFound {
			lastError = fmt.Errorf("keyword export %s is not available", date)
			continue
		}
		if res.StatusCode/100 != 2 {
			lastError = fmt.Errorf("keyword export returned HTTP %d", res.StatusCode)
			continue
		}
		if len(body) > maxExportBytes {
			return nil, errors.New("keyword export exceeds size limit")
		}
		return body, nil
	}
	if lastError == nil {
		lastError = errors.New("no recent keyword export is available")
	}
	return nil, lastError
}

func decodeExport(data []byte) ([]Entry, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open keyword export: %w", err)
	}
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	entries := make([]Entry, 0, 100_000)
	seen := make(map[int]struct{}, 100_000)
	for {
		var entry Entry
		if err := decoder.Decode(&entry); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, fmt.Errorf("decode keyword export: %w", err)
		}
		entry.Name = strings.TrimSpace(entry.Name)
		entry.normalized = normalize(entry.Name)
		if entry.ID <= 0 || entry.normalized == "" {
			continue
		}
		if _, duplicate := seen[entry.ID]; duplicate {
			continue
		}
		seen[entry.ID] = struct{}{}
		entries = append(entries, entry)
	}
	if len(entries) < 1_000 {
		return nil, fmt.Errorf("keyword export contains only %d valid entries", len(entries))
	}
	sort.Slice(entries, func(left, right int) bool {
		if entries[left].normalized == entries[right].normalized {
			return entries[left].ID < entries[right].ID
		}
		return entries[left].normalized < entries[right].normalized
	})
	return entries, nil
}
