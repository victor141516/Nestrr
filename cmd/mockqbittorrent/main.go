// mockqbittorrent accepts Nestrr's qBittorrent calls and creates a real video.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const infoHash = "4242424242424242424242424242424242424242"
const releaseFile = "Temporal.Horizon.2026.1080p.WEB-DL.VP9-E2E.webm"

type serviceState struct {
	sync.Mutex
	Accepted  int     `json:"accepted"`
	Generated bool    `json:"generated"`
	Category  string  `json:"category"`
	URL       string  `json:"url"`
	Progress  float64 `json:"progress"`
	State     string  `json:"state"`
	Forced    bool    `json:"forced"`
	Error     string  `json:"error,omitempty"`
}

var current serviceState
var once sync.Once

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/api/v2/auth/login", login)
	mux.HandleFunc("/api/v2/app/version", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("v5.2.0-mock")) })
	mux.HandleFunc("/api/v2/torrents/add", addTorrent)
	mux.HandleFunc("/api/v2/torrents/info", torrentInfo)
	mux.HandleFunc("/api/v2/torrents/setForceStart", forceStart)
	mux.HandleFunc("/mock/state", mockState)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func mockState(w http.ResponseWriter, r *http.Request) {
	current.Lock()
	defer current.Unlock()
	if r.Method == http.MethodPost {
		var update struct {
			State    string  `json:"state"`
			Progress float64 `json:"progress"`
		}
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "invalid state", http.StatusBadRequest)
			return
		}
		current.State = update.State
		current.Progress = update.Progress
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(current)
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "SID", Value: "e2e-session", Path: "/", HttpOnly: true})
	w.WriteHeader(http.StatusNoContent)
}

func addTorrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	current.Lock()
	current.Accepted++
	current.Category = r.FormValue("category")
	current.URL = r.FormValue("urls")
	current.Progress = 0.05
	current.State = env("MOCK_QB_INITIAL_STATE", "downloading")
	current.Unlock()
	once.Do(func() { go materializeVideo() })
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"added_torrent_ids": []string{infoHash}, "failure_count": 0, "pending_count": 0, "success_count": 1})
}

func torrentInfo(w http.ResponseWriter, _ *http.Request) {
	current.Lock()
	defer current.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if current.Accepted == 0 {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}
	_ = json.NewEncoder(w).Encode([]map[string]any{{"hash": infoHash, "name": releaseFile, "state": current.State, "progress": current.Progress}})
}

func forceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil || r.FormValue("hashes") != infoHash || r.FormValue("value") != "true" {
		http.Error(w, "invalid force start", http.StatusBadRequest)
		return
	}
	current.Lock()
	current.State = "forcedDL"
	current.Forced = true
	current.Unlock()
	w.WriteHeader(http.StatusOK)
}

func materializeVideo() {
	delay, _ := strconv.Atoi(env("MOCK_QB_DELAY_SECONDS", "2"))
	time.Sleep(time.Duration(delay) * time.Second)
	root := env("MOCK_QB_DOWNLOAD_ROOT", "/data/downloads")
	incomplete, completed := filepath.Join(root, "incomplete"), filepath.Join(root, "completed", "Movies")
	if err := os.MkdirAll(incomplete, 0o755); err != nil {
		setError(err)
		return
	}
	if err := os.MkdirAll(completed, 0o755); err != nil {
		setError(err)
		return
	}
	partial, final := filepath.Join(incomplete, releaseFile), filepath.Join(completed, releaseFile)
	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "testsrc2=size=320x180:rate=24", "-f", "lavfi", "-i", "sine=frequency=880:sample_rate=48000", "-t", "8", "-c:v", "libvpx-vp9", "-deadline", "realtime", "-cpu-used", "8", "-b:v", "500k", "-pix_fmt", "yuv420p", "-c:a", "libopus", partial)
	if output, err := cmd.CombinedOutput(); err != nil {
		setError(fmt.Errorf("ffmpeg: %w: %s", err, output))
		return
	}
	if err := os.Rename(partial, final); err != nil {
		setError(err)
		return
	}
	current.Lock()
	current.Generated = true
	current.Progress = 1
	current.State = "uploading"
	current.Unlock()
	log.Printf("mock download completed: %s", final)
}
func setError(err error) {
	current.Lock()
	current.Error = err.Error()
	current.Unlock()
	log.Printf("mock download failed: %v", err)
}
func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
