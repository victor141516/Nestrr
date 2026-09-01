package integrations

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Status struct {
	Configured bool   `json:"configured"`
	Healthy    bool   `json:"healthy"`
	CheckedAt  string `json:"checkedAt"`
	Message    string `json:"message,omitempty"`
}
type Manager struct {
	JackettHost, JackettKey, QBHost, QBUser, QBPass, ShowCategory, MovieCategory string
	HTTP                                                                         *http.Client
	Log                                                                          *slog.Logger
	mu                                                                           sync.RWMutex
	statuses                                                                     map[string]Status
}

func New(jh, jk, qh, qu, qp, shows, movies string, log *slog.Logger) *Manager {
	jar, _ := cookiejar.New(nil)
	return &Manager{JackettHost: jh, JackettKey: jk, QBHost: qh, QBUser: qu, QBPass: qp, ShowCategory: shows, MovieCategory: movies, HTTP: &http.Client{Timeout: 15 * time.Second, Jar: jar}, Log: log, statuses: map[string]Status{}}
}
func (m *Manager) Start(ctx context.Context) {
	m.check(ctx)
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				m.check(ctx)
			}
		}
	}()
}
func (m *Manager) Statuses() map[string]Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]Status{}
	for k, v := range m.statuses {
		out[k] = v
	}
	return out
}
func (m *Manager) check(ctx context.Context) {
	j := m.health(ctx, "jackett", m.JackettHost, m.JackettHost+"/api/v2.0/server/config?apikey="+url.QueryEscape(m.JackettKey))
	q := m.health(ctx, "qbittorrent", m.QBHost, m.QBHost+"/api/v2/app/version")
	m.mu.Lock()
	m.statuses["jackett"] = j
	m.statuses["qbittorrent"] = q
	m.mu.Unlock()
}
func (m *Manager) health(ctx context.Context, name, host, target string) Status {
	s := Status{Configured: host != "", CheckedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if host == "" {
		s.Message = "not configured"
		return s
	}
	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if name == "qbittorrent" {
		_ = m.loginQB(c)
	}
	req, _ := http.NewRequestWithContext(c, http.MethodGet, target, nil)
	res, err := m.HTTP.Do(req)
	if err != nil {
		s.Message = "unreachable"
		m.Log.Warn("integration health failed", "integration", name, "operation", "health", "error", err)
		return s
	}
	res.Body.Close()
	s.Healthy = res.StatusCode >= 200 && res.StatusCode < 300
	if !s.Healthy {
		s.Message = fmt.Sprintf("HTTP %d", res.StatusCode)
	}
	return s
}

type torznab struct {
	Items []struct {
		Title     string `xml:"title"`
		Link      string `xml:"link"`
		Guid      string `xml:"guid"`
		Indexer   string `xml:"jackettindexer"`
		Enclosure struct {
			URL    string `xml:"url,attr"`
			Length int64  `xml:"length,attr"`
		} `xml:"enclosure"`
		Attrs []struct {
			Name  string `xml:"name,attr"`
			Value string `xml:"value,attr"`
		} `xml:"attr"`
	} `xml:"channel>item"`
}
type SearchResult struct {
	ID, Title, Tracker, Link, MagnetURI string
	Seeders                             int
	Size                                int64
	Published                           string
}

func (m *Manager) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if m.JackettHost == "" || m.JackettKey == "" {
		return nil, errors.New("Jackett is not configured")
	}
	u := m.JackettHost + "/api/v2.0/indexers/all/results/torznab/api?apikey=" + url.QueryEscape(m.JackettKey) + "&t=search&q=" + url.QueryEscape(query)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	res, err := m.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Jackett returned HTTP %d", res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	var feed torznab
	if err = xml.Unmarshal(b, &feed); err != nil {
		return nil, err
	}
	out := make([]SearchResult, 0, len(feed.Items))
	for i, item := range feed.Items {
		r := SearchResult{ID: strconv.Itoa(i), Title: item.Title, Link: item.Enclosure.URL, Size: item.Enclosure.Length}
		if r.Link == "" {
			r.Link = item.Link
		}
		for _, a := range item.Attrs {
			switch strings.ToLower(a.Name) {
			case "seeders":
				r.Seeders, _ = strconv.Atoi(a.Value)
			case "size":
				r.Size, _ = strconv.ParseInt(a.Value, 10, 64)
			case "magneturl":
				r.MagnetURI = a.Value
			case "tracker":
				r.Tracker = a.Value
			case "publishdate":
				r.Published = a.Value
			}
		}
		if r.Tracker == "" {
			r.Tracker = item.Indexer
		}
		if r.Tracker == "" {
			r.Tracker = item.Guid
		}
		out = append(out, r)
	}
	return out, nil
}

func (m *Manager) loginQB(ctx context.Context) error {
	if m.QBHost == "" {
		return errors.New("qBittorrent is not configured")
	}
	form := url.Values{"username": {m.QBUser}, "password": {m.QBPass}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.QBHost+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", m.QBHost)
	res, err := m.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	if res.StatusCode/100 != 2 {
		return errors.New("qBittorrent authentication failed")
	}
	// qBittorrent 5.2 responds with 204 No Content on a successful login,
	// while older releases respond with 200 and "Ok.".
	loginResponse := strings.TrimSpace(string(b))
	if loginResponse != "" && !strings.EqualFold(loginResponse, "Ok.") {
		return errors.New("qBittorrent authentication failed")
	}
	return nil
}

type SubmitRequest struct{ URL, Category string }

type TorrentInfo struct {
	Hash     string  `json:"infoHash"`
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Progress float64 `json:"progress"`
}

var ErrTorrentDownload = errors.New("torrent metadata download failed")

const maxTorrentFileSize = 16 << 20

func (m *Manager) Submit(ctx context.Context, in SubmitRequest) (string, error) {
	hash := extractInfoHash(in.URL)
	var torrentFile []byte
	if hash == "" {
		var err error
		torrentFile, hash, err = m.downloadTorrent(ctx, in.URL)
		if err != nil {
			return "", err
		}
	}
	if err := m.loginQB(ctx); err != nil {
		return "", err
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if len(torrentFile) > 0 {
		part, err := w.CreateFormFile("torrents", "download.torrent")
		if err != nil {
			return "", err
		}
		if _, err = part.Write(torrentFile); err != nil {
			return "", err
		}
	} else if err := w.WriteField("urls", in.URL); err != nil {
		return "", err
	}
	if err := w.WriteField("category", in.Category); err != nil {
		return "", err
	}
	if err := w.WriteField("autoTMM", "true"); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.QBHost+"/api/v2/torrents/add", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Referer", m.QBHost)
	res, err := m.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		return "", fmt.Errorf("qBittorrent rejected submission")
	}

	response := strings.TrimSpace(string(b))
	if response == "" || strings.EqualFold(response, "Ok.") {
		return hash, nil
	}

	// qBittorrent 5.2 returns a JSON submission report instead of "Ok.".
	var report struct {
		AddedTorrentIDs []string `json:"added_torrent_ids"`
		FailureCount    int      `json:"failure_count"`
		PendingCount    int      `json:"pending_count"`
		SuccessCount    int      `json:"success_count"`
	}
	// A pending remote URL is not a successful add: qBittorrent can acknowledge
	// it and then fail to fetch it later. HTTP results are uploaded as .torrent
	// files above, so success here means the torrent was actually parsed/added.
	if err := json.Unmarshal(b, &report); err != nil || report.FailureCount > 0 || report.PendingCount > 0 || report.SuccessCount < 1 {
		return "", fmt.Errorf("qBittorrent rejected submission")
	}
	if len(report.AddedTorrentIDs) > 0 {
		hash = report.AddedTorrentIDs[0]
	}
	return hash, nil
}

func validInfoHash(hash string) bool {
	if len(hash) != 40 && len(hash) != 64 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}

func (m *Manager) Torrent(ctx context.Context, hash string) (TorrentInfo, bool, error) {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if !validInfoHash(hash) {
		return TorrentInfo{}, false, errors.New("invalid torrent info hash")
	}
	if err := m.loginQB(ctx); err != nil {
		return TorrentInfo{}, false, err
	}
	target := m.QBHost + "/api/v2/torrents/info?" + url.Values{"hashes": {hash}}.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	req.Header.Set("Referer", m.QBHost)
	res, err := m.HTTP.Do(req)
	if err != nil {
		return TorrentInfo{}, false, err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return TorrentInfo{}, false, fmt.Errorf("qBittorrent rejected torrent status request")
	}
	var torrents []struct {
		Hash     string  `json:"hash"`
		Name     string  `json:"name"`
		State    string  `json:"state"`
		Progress float64 `json:"progress"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&torrents); err != nil {
		return TorrentInfo{}, false, fmt.Errorf("invalid qBittorrent torrent status response: %w", err)
	}
	for _, torrent := range torrents {
		if strings.EqualFold(torrent.Hash, hash) {
			return TorrentInfo{Hash: strings.ToLower(torrent.Hash), Name: torrent.Name, State: torrent.State, Progress: torrent.Progress}, true, nil
		}
	}
	return TorrentInfo{}, false, nil
}

func (m *Manager) ForceStart(ctx context.Context, hash string) error {
	return m.ForceStartMany(ctx, []string{hash})
}

func (m *Manager) ForceStartMany(ctx context.Context, hashes []string) error {
	normalized := make([]string, 0, len(hashes))
	seen := map[string]bool{}
	for _, hash := range hashes {
		hash = strings.ToLower(strings.TrimSpace(hash))
		if !validInfoHash(hash) {
			return errors.New("invalid torrent info hash")
		}
		if !seen[hash] {
			seen[hash] = true
			normalized = append(normalized, hash)
		}
	}
	if len(normalized) == 0 {
		return errors.New("at least one torrent info hash is required")
	}
	if err := m.loginQB(ctx); err != nil {
		return err
	}
	form := url.Values{"hashes": {strings.Join(normalized, "|")}, "value": {"true"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, m.QBHost+"/api/v2/torrents/setForceStart", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", m.QBHost)
	res, err := m.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("qBittorrent rejected force start")
	}
	return nil
}

func (m *Manager) downloadTorrent(ctx context.Context, rawURL string) ([]byte, string, error) {
	target, err := url.Parse(rawURL)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		return nil, "", fmt.Errorf("%w: invalid Jackett download URL", ErrTorrentDownload)
	}
	jackett, err := url.Parse(m.JackettHost)
	if err != nil || jackett.Host == "" || !strings.EqualFold(target.Host, jackett.Host) {
		return nil, "", fmt.Errorf("%w: download URL is not served by Jackett", ErrTorrentDownload)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	req.Header.Set("Accept", "application/x-bittorrent, application/octet-stream")
	res, err := m.HTTP.Do(req)
	if err != nil {
		// Do not wrap the transport error: it commonly contains Jackett's API key.
		return nil, "", fmt.Errorf("%w: Jackett request failed", ErrTorrentDownload)
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return nil, "", fmt.Errorf("%w: Jackett returned HTTP %d", ErrTorrentDownload, res.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, maxTorrentFileSize+1))
	if err != nil {
		return nil, "", fmt.Errorf("%w: response could not be read", ErrTorrentDownload)
	}
	if len(b) > maxTorrentFileSize {
		return nil, "", fmt.Errorf("%w: torrent file is too large", ErrTorrentDownload)
	}
	hash, err := torrentInfoHash(b)
	if err != nil {
		return nil, "", fmt.Errorf("%w: Jackett did not return a valid torrent file", ErrTorrentDownload)
	}
	return b, hash, nil
}

func torrentInfoHash(data []byte) (string, error) {
	if len(data) < 2 || data[0] != 'd' {
		return "", errors.New("torrent is not a bencoded dictionary")
	}
	for pos := 1; pos < len(data) && data[pos] != 'e'; {
		key, next, err := bencodedString(data, pos)
		if err != nil {
			return "", err
		}
		start := next
		end, err := skipBencoded(data, start, 0)
		if err != nil {
			return "", err
		}
		if key == "info" {
			sum := sha1.Sum(data[start:end])
			return hex.EncodeToString(sum[:]), nil
		}
		pos = end
	}
	return "", errors.New("torrent has no info dictionary")
}

func bencodedString(data []byte, pos int) (string, int, error) {
	colon := pos
	for colon < len(data) && data[colon] >= '0' && data[colon] <= '9' {
		colon++
	}
	if colon == pos || colon >= len(data) || data[colon] != ':' {
		return "", 0, errors.New("invalid bencoded string")
	}
	length, err := strconv.Atoi(string(data[pos:colon]))
	start := colon + 1
	if err != nil || length < 0 || length > len(data)-start {
		return "", 0, errors.New("invalid bencoded string length")
	}
	return string(data[start : start+length]), start + length, nil
}

func skipBencoded(data []byte, pos, depth int) (int, error) {
	if pos >= len(data) || depth > 100 {
		return 0, errors.New("invalid bencoded value")
	}
	switch data[pos] {
	case 'i':
		end := bytes.IndexByte(data[pos+1:], 'e')
		if end < 1 {
			return 0, errors.New("invalid bencoded integer")
		}
		return pos + 1 + end + 1, nil
	case 'l', 'd':
		pos++
		for pos < len(data) && data[pos] != 'e' {
			var err error
			pos, err = skipBencoded(data, pos, depth+1)
			if err != nil {
				return 0, err
			}
		}
		if pos >= len(data) {
			return 0, errors.New("unterminated bencoded collection")
		}
		return pos + 1, nil
	default:
		_, next, err := bencodedString(data, pos)
		return next, err
	}
}

func extractInfoHash(v string) string {
	u, err := url.Parse(v)
	if err != nil {
		return ""
	}
	xt := u.Query().Get("xt")
	if strings.HasPrefix(strings.ToLower(xt), "urn:btih:") {
		return strings.TrimPrefix(xt, "urn:btih:")
	}
	return ""
}
func JSON(v any) []byte { b, _ := json.Marshal(v); return b }
