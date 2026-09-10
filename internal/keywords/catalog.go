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
	"sort"
	"strings"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

const (
	defaultExportBase = "https://files.tmdb.org/p/exports"
	refreshInterval   = 7 * 24 * time.Hour
	maxExportBytes    = 16 << 20
)

type Entry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Catalog imports TMDB's weekly export directly into SQLite. The export is
// transient input only; the database is the sole persistent keyword source.
type Catalog struct {
	store      *store.Store
	exportBase string
	http       *http.Client
	log        *slog.Logger
	now        func() time.Time
}

func New(database *store.Store, client *http.Client, log *slog.Logger) *Catalog {
	if client == nil {
		client = http.DefaultClient
	}
	if log == nil {
		log = slog.Default()
	}
	return &Catalog{store: database, exportBase: defaultExportBase, http: client, log: log, now: time.Now}
}

func (c *Catalog) Start(ctx context.Context) {
	if c.store == nil {
		return
	}
	if ready, updatedAt := c.state(ctx); !ready || c.now().Sub(updatedAt) >= refreshInterval {
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
				if ready, updatedAt := c.state(ctx); !ready || c.now().Sub(updatedAt) >= refreshInterval {
					if err := c.refresh(ctx); err != nil {
						c.log.Warn("TMDB keyword export refresh failed", "operation", "keywords.refresh", "error", err)
					}
				}
			}
		}
	}()
}

func (c *Catalog) state(ctx context.Context) (bool, time.Time) {
	count, updatedAt, err := c.store.KeywordCatalogState(ctx)
	return err == nil && count > 0, updatedAt
}

func (c *Catalog) Ready(ctx context.Context) bool {
	ready, _ := c.state(ctx)
	return ready
}

func (c *Catalog) UpdatedAt(ctx context.Context) time.Time {
	_, updatedAt := c.state(ctx)
	return updatedAt
}

func (c *Catalog) Prefix(ctx context.Context, query string, limit int) ([]Entry, error) {
	return c.search(ctx, query, "prefix", limit)
}

func (c *Catalog) Contains(ctx context.Context, query string, limit int) ([]Entry, error) {
	return c.search(ctx, query, "contains", limit)
}

func (c *Catalog) search(ctx context.Context, query, phase string, limit int) ([]Entry, error) {
	values, err := c.store.SearchKeywords(ctx, query, phase, limit)
	if err != nil {
		return nil, err
	}
	results := make([]Entry, len(values))
	for index, value := range values {
		results[index] = Entry{ID: value.ID, Name: value.Name}
	}
	return results, nil
}

func (c *Catalog) ByIDs(ctx context.Context, ids []int) ([]Entry, error) {
	values, err := c.store.KeywordsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]Entry, len(values))
	for index, value := range values {
		results[index] = Entry{ID: value.ID, Name: value.Name}
	}
	return results, nil
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
	values := make([]store.Keyword, len(entries))
	for index, entry := range entries {
		values[index] = store.Keyword{ID: entry.ID, Name: entry.Name}
	}
	now := c.now()
	if err := c.store.ReplaceKeywordCatalog(ctx, values, now); err != nil {
		return err
	}
	c.log.Info("TMDB keyword export imported", "operation", "keywords.refresh", "entries", len(entries))
	return nil
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
		if entry.ID <= 0 || entry.Name == "" {
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
		leftName, rightName := strings.ToLower(entries[left].Name), strings.ToLower(entries[right].Name)
		if leftName == rightName {
			return entries[left].ID < entries[right].ID
		}
		return leftName < rightName
	})
	return entries, nil
}
