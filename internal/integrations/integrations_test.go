package integrations

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSubmitLogsInAndPreservesCategory(t *testing.T) {
	var added string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.Header().Add("Set-Cookie", "SID=test; Path=/")
			_, _ = w.Write([]byte("Ok."))
		case "/api/v2/torrents/add":
			if !strings.Contains(r.Header.Get("Cookie"), "SID=test") {
				t.Fatal("authenticated qBittorrent cookie was not retained")
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			added = r.FormValue("urls") + "|" + r.FormValue("category") + "|" + r.FormValue("autoTMM")
			_, _ = w.Write([]byte("Ok."))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	m := New("", "", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	hash, err := m.Submit(context.Background(), SubmitRequest{URL: "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567", Category: "Movies"})
	if err != nil {
		t.Fatal(err)
	}
	if added != "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567|Movies|true" {
		t.Fatalf("unexpected qBittorrent form: %s", added)
	}
	if hash != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("unexpected info hash: %s", hash)
	}
}

func TestSubmitAcceptsQBittorrent52Responses(t *testing.T) {
	torrent := []byte("d4:infod4:name4:testee")
	var uploaded bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/test.torrent":
			w.Header().Set("Content-Type", "application/x-bittorrent")
			_, _ = w.Write(torrent)
		case "/api/v2/auth/login":
			w.Header().Add("Set-Cookie", "SID=test; Path=/")
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/add":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatal(err)
			}
			file, _, err := r.FormFile("torrents")
			if err != nil {
				t.Fatalf("torrent file was not uploaded: %v", err)
			}
			defer file.Close()
			got, _ := io.ReadAll(file)
			uploaded = string(got) == string(torrent) && r.FormValue("urls") == ""
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"added_torrent_ids":["aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"],"failure_count":0,"pending_count":0,"success_count":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m := New(server.URL, "key", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	hash, err := m.Submit(context.Background(), SubmitRequest{URL: server.URL + "/download/test.torrent", Category: "Movies"})
	if err != nil {
		t.Fatal(err)
	}
	if !uploaded {
		t.Fatal("qBittorrent did not receive the downloaded .torrent file")
	}
	if hash != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("unexpected info hash: %s", hash)
	}
}

func TestSubmitRejectsApplicationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/add":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"added_torrent_ids":[],"failure_count":1,"pending_count":0,"success_count":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m := New("", "", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := m.Submit(context.Background(), SubmitRequest{URL: "magnet:?xt=urn:btih:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Category: "Movies"}); err == nil {
		t.Fatal("qBittorrent application failure was accepted")
	}
}

func TestSubmitRejectsPendingRemoteTorrent(t *testing.T) {
	torrent := []byte("d4:infod4:name4:testee")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download/remote.torrent":
			_, _ = w.Write(torrent)
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/add":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"added_torrent_ids":[],"failure_count":0,"pending_count":1,"success_count":0}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	m := New(server.URL, "key", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := m.Submit(context.Background(), SubmitRequest{URL: server.URL + "/download/remote.torrent", Category: "Movies"}); err == nil {
		t.Fatal("pending remote torrent was accepted as a successful add")
	}
}

func TestSubmitReportsJackettDownloadFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	m := New(server.URL, "key", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	_, err := m.Submit(context.Background(), SubmitRequest{URL: server.URL + "/download/broken.torrent", Category: "Shows"})
	if !errors.Is(err, ErrTorrentDownload) {
		t.Fatalf("expected ErrTorrentDownload, got %v", err)
	}
}

func TestTorrentReturnsExactQBittorrentStatus(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef01234567"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.Header().Add("Set-Cookie", "SID=test; Path=/")
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/info":
			if r.URL.Query().Get("hashes") != hash {
				t.Fatalf("unexpected hash filter: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"hash":"0123456789abcdef0123456789abcdef01234567","name":"Movie.Release.mkv","state":"stalledDL","progress":0.42}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	m := New("", "", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	status, found, err := m.Torrent(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if !found || status.State != "stalledDL" || status.Progress != 0.42 || status.Name != "Movie.Release.mkv" {
		t.Fatalf("unexpected torrent status: %#v found=%v", status, found)
	}
}

func TestForceStartUsesQBittorrentForceStartAction(t *testing.T) {
	const hash = "0123456789abcdef0123456789abcdef01234567"
	var received url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/setForceStart":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			received = r.Form
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	m := New("", "", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := m.ForceStart(context.Background(), hash); err != nil {
		t.Fatal(err)
	}
	if received.Get("hashes") != hash || received.Get("value") != "true" {
		t.Fatalf("unexpected force start form: %#v", received)
	}
}

func TestForceStartManyUsesOneQBittorrentActionForEveryHash(t *testing.T) {
	hashes := []string{
		"0123456789abcdef0123456789abcdef01234567",
		"89abcdef0123456789abcdef0123456789abcdef",
	}
	var received url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/torrents/setForceStart":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			received = r.Form
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	m := New("", "", server.URL, "admin", "secret", "Shows", "Movies", slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := m.ForceStartMany(context.Background(), hashes); err != nil {
		t.Fatal(err)
	}
	if received.Get("hashes") != strings.Join(hashes, "|") || received.Get("value") != "true" {
		t.Fatalf("unexpected force start form: %#v", received)
	}
}

func TestTorrentInfoHashUsesRawInfoDictionary(t *testing.T) {
	data := []byte("d8:announce13:https://track4:infod4:name4:testee")
	got, err := torrentInfoHash(data)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes := sha1.Sum([]byte("d4:name4:teste"))
	want := hex.EncodeToString(wantBytes[:])
	if got != want {
		t.Fatalf("unexpected info hash: got %s want %s", got, want)
	}
}
