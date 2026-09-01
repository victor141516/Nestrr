package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var reserved = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\..*)?$`)
var invalidControls = regexp.MustCompile(`[\x00-\x1f\x80-\x9f]`)
var separators = regexp.MustCompile(`[:/\\]+`)
var invalid = regexp.MustCompile(`[<>"|?*]`)
var whitespace = regexp.MustCompile(`\s+`)
var trailing = regexp.MustCompile(`[. ]+$`)
var supported = map[string]bool{".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".webm": true, ".flv": true, ".m4v": true, ".mkv": true, ".vob": true, ".ts": true, ".3gp": true, ".asf": true, ".divx": true}

// SanitizeTitle intentionally mirrors Chiprr's MediaPathBuilder contract.
func SanitizeTitle(value string) (string, error) {
	v := invalidControls.ReplaceAllString(value, " ")
	v = separators.ReplaceAllString(v, " - ")
	v = invalid.ReplaceAllString(v, " ")
	v = whitespace.ReplaceAllString(v, " ")
	v = strings.TrimSpace(v)
	v = trailing.ReplaceAllString(v, "")
	if reserved.MatchString(v) {
		v += "_"
	}
	if v == "" || v == "." || v == ".." {
		return "", errors.New("media title cannot be represented as a filename")
	}
	return v, nil
}
func IsVideo(path string) bool { return supported[strings.ToLower(filepath.Ext(path))] }

type Resolver struct{ MoviesRoot, ShowsRoot string }

func (r Resolver) Movie(title string, year int) (string, bool, error) {
	title, err := SanitizeTitle(title)
	if err != nil {
		return "", false, err
	}
	stem := title
	if year > 0 {
		stem += fmt.Sprintf(" (%d)", year)
	}
	return choose(r.MoviesRoot, stem)
}
func (r Resolver) Episode(show string, season, episode int) (string, bool, error) {
	if season <= 0 || episode <= 0 {
		return "", false, errors.New("season and episode must be positive")
	}
	show, err := SanitizeTitle(show)
	if err != nil {
		return "", false, err
	}
	dir := filepath.Join(r.ShowsRoot, show, fmt.Sprintf("Season %d", season))
	stem := fmt.Sprintf("%s S%02dE%02d", show, season, episode)
	return choose(dir, stem)
}
func (r Resolver) Show(show string) (string, bool, error) {
	show, err := SanitizeTitle(show)
	if err != nil {
		return "", false, err
	}
	dir := filepath.Join(r.ShowsRoot, show)
	available := false
	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() && IsVideo(path) {
			available = true
			return filepath.SkipAll
		}
		return nil
	})
	return dir, available, err
}
func choose(dir, stem string) (string, bool, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return filepath.Join(dir, stem), false, nil
	}
	if err != nil {
		return "", false, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), stem) && IsVideo(e.Name()) {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		return paths[0], true, nil
	}
	return filepath.Join(dir, stem), false, nil
}

type LibraryEntry struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Title   string `json:"title"`
	Year    int    `json:"year,omitempty"`
	Show    string `json:"show,omitempty"`
	Season  int    `json:"season,omitempty"`
	Episode int    `json:"episode,omitempty"`
}

var movieName = regexp.MustCompile(`^(.*?)(?: \((\d{4})\))?\.[^.]+$`)
var episodeName = regexp.MustCompile(`^(.*?) S(\d{2,})E(\d{2,})\.[^.]+$`)

func (r Resolver) Enumerate() ([]LibraryEntry, error) {
	var out []LibraryEntry
	movies, err := os.ReadDir(r.MoviesRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, e := range movies {
		if e.IsDir() || !IsVideo(e.Name()) {
			continue
		}
		m := movieName.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		year := 0
		fmt.Sscanf(m[2], "%d", &year)
		out = append(out, LibraryEntry{Kind: "movie", Path: filepath.ToSlash(filepath.Join("Movies", e.Name())), Title: m[1], Year: year})
	}
	err = filepath.WalkDir(r.ShowsRoot, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() || !IsVideo(path) {
			return nil
		}
		m := episodeName.FindStringSubmatch(d.Name())
		if m == nil {
			return nil
		}
		var s, ep int
		fmt.Sscanf(m[2], "%d", &s)
		fmt.Sscanf(m[3], "%d", &ep)
		rel, _ := filepath.Rel(filepath.Dir(r.ShowsRoot), path)
		out = append(out, LibraryEntry{Kind: "episode", Path: filepath.ToSlash(rel), Title: m[1], Show: m[1], Season: s, Episode: ep})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}
