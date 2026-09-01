package server

import (
	"testing"

	"github.com/victor141516/Nestrr/internal/integrations"
)

func TestCanForceStartTorrent(t *testing.T) {
	for _, state := range []string{"stalledDL", "queuedDL"} {
		if !canForceStartTorrent(state) {
			t.Fatalf("state %q should be forceable", state)
		}
	}
	for _, state := range []string{"downloading", "forcedDL", "pausedDL"} {
		if canForceStartTorrent(state) {
			t.Fatalf("state %q should not be forceable", state)
		}
	}
}

func TestTorrentWithMostProgress(t *testing.T) {
	torrents := []integrations.TorrentInfo{
		{Hash: "first", Progress: 0.25},
		{Hash: "second", Progress: 0.8},
		{Hash: "third", Progress: 0.55},
	}
	var selected *integrations.TorrentInfo
	for _, torrent := range torrents {
		selected = torrentWithMostProgress(selected, torrent)
	}
	if selected == nil || selected.Hash != "second" {
		t.Fatalf("selected torrent = %#v, want second", selected)
	}
}

func TestTorrentWithMostProgressKeepsFirstOnTie(t *testing.T) {
	selected := torrentWithMostProgress(nil, integrations.TorrentInfo{Hash: "newest", Progress: 0.8})
	selected = torrentWithMostProgress(selected, integrations.TorrentInfo{Hash: "older", Progress: 0.8})
	if selected == nil || selected.Hash != "newest" {
		t.Fatalf("selected torrent = %#v, want newest", selected)
	}
}
