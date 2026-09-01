// mocktmdb is a deterministic integration-test fixture, never used in production.
package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

var requests atomic.Int64
var movies = []map[string]any{
	{"id": 424242, "title": "Temporal Horizon", "original_title": "Temporal Horizon", "release_date": "2026-02-20", "overview": "A test pilot crosses a temporal storm and races home before the horizon closes.", "vote_average": 8.4, "genre_ids": []int{12, 878}, "poster_path": "/temporal.png", "backdrop_path": "/temporal-bg.png"},
	{"id": 329, "title": "Jurassic Park", "original_title": "Jurassic Park", "release_date": "1993-06-11", "overview": "A paleontologist visits a theme park populated by living dinosaurs.", "vote_average": 7.9, "genre_ids": []int{12, 878}, "poster_path": "/jurassic.png", "backdrop_path": "/jurassic-bg.png"},
	{"id": 348, "title": "Alien", "original_title": "Alien", "release_date": "1979-05-25", "overview": "A deep-space crew encounters an unknown lifeform.", "vote_average": 8.2, "genre_ids": []int{878}, "poster_path": "/alien.png", "backdrop_path": "/alien-bg.png"},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/stats", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"requests": requests.Load(), "service": "mock-tmdb"})
	})
	mux.HandleFunc("/image/", func(w http.ResponseWriter, _ *http.Request) {
		b, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/3/", fixture)
	_ = http.ListenAndServe(":9090", mux)
}

func fixture(w http.ResponseWriter, r *http.Request) {
	requests.Add(1)
	path := strings.TrimPrefix(r.URL.Path, "/3")
	var v any
	switch path {
	case "/discover/movie":
		v = page(asAny(movies))
	case "/search/movie":
		query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query")))
		results := make([]map[string]any, 0)
		for _, movie := range movies {
			title := strings.ToLower(movie["title"].(string))
			if query == "" || strings.Contains(title, query) || strings.Contains(query, title) {
				results = append(results, movie)
			}
		}
		v = page(asAny(results))
	case "/discover/tv", "/search/tv":
		v = page([]any{show()})
	case "/genre/movie/list":
		v = map[string]any{"genres": []any{map[string]any{"id": 12, "name": "Adventure"}, map[string]any{"id": 878, "name": "Science Fiction"}}}
	case "/genre/tv/list":
		v = map[string]any{"genres": []any{map[string]any{"id": 18, "name": "Drama"}}}
	case "/configuration/countries":
		v = []any{
			map[string]any{"iso_3166_1": "ES", "english_name": "Spain", "native_name": "España"},
			map[string]any{"iso_3166_1": "FR", "english_name": "France", "native_name": "France"},
			map[string]any{"iso_3166_1": "GB", "english_name": "United Kingdom", "native_name": "United Kingdom"},
			map[string]any{"iso_3166_1": "HK", "english_name": "Hong Kong", "native_name": "香港"},
			map[string]any{"iso_3166_1": "KP", "english_name": "North Korea", "native_name": "조선민주주의인민공화국"},
			map[string]any{"iso_3166_1": "MN", "english_name": "Mongolia", "native_name": "Монгол Улс"},
			map[string]any{"iso_3166_1": "MO", "english_name": "Macao", "native_name": "澳門"},
			map[string]any{"iso_3166_1": "TW", "english_name": "Taiwan", "native_name": "台灣"},
			map[string]any{"iso_3166_1": "US", "english_name": "United States of America", "native_name": "United States"},
		}
	case "/certification/movie/list":
		v = map[string]any{"certifications": map[string]any{
			"ES": []any{map[string]any{"certification": "A", "meaning": "All audiences", "order": 1}, map[string]any{"certification": "7", "meaning": "Not recommended under 7", "order": 2}, map[string]any{"certification": "12", "meaning": "Not recommended under 12", "order": 3}, map[string]any{"certification": "16", "meaning": "Not recommended under 16", "order": 4}, map[string]any{"certification": "18", "meaning": "Adults only", "order": 5}},
			"US": []any{map[string]any{"certification": "G", "meaning": "General audiences", "order": 1}, map[string]any{"certification": "PG", "meaning": "Parental guidance suggested", "order": 2}, map[string]any{"certification": "PG-13", "meaning": "Parents strongly cautioned", "order": 3}, map[string]any{"certification": "R", "meaning": "Restricted", "order": 4}, map[string]any{"certification": "NC-17", "meaning": "Adults only", "order": 5}},
		}}
	case "/tv/1396":
		v = showDetail()
		if strings.Contains(r.URL.Query().Get("append_to_response"), "content_ratings") {
			v.(map[string]any)["content_ratings"] = map[string]any{"results": []any{map[string]any{"iso_3166_1": "US", "rating": "TV-MA"}}}
		}
	case "/tv/1396/recommendations":
		v = page([]any{})
	case "/tv/1396/season/1":
		v = season()
	case "/tv/1396/season/1/episode/1":
		v = episode(1, "Pilot")
	default:
		if id, suffix, ok := moviePath(path); ok {
			movie := findMovie(id)
			if movie != nil {
				switch suffix {
				case "":
					v = movieDetail(movie)
					if strings.Contains(r.URL.Query().Get("append_to_response"), "release_dates") {
						releases := []any{}
						if id != 329 { // Jurassic Park intentionally exercises the unrated-in-US path.
							certification := "PG-13"
							if id == 348 {
								certification = "R"
							}
							releases = append(releases, map[string]any{"iso_3166_1": "US", "release_dates": []any{map[string]any{"certification": certification}}})
						}
						v.(map[string]any)["release_dates"] = map[string]any{"results": releases}
					}
				case "/recommendations":
					v = page([]any{})
				case "/translations":
					v = map[string]any{"id": id, "translations": []any{map[string]any{"iso_3166_1": "US", "iso_639_1": "en", "english_name": "English", "data": map[string]any{"title": movie["title"]}}}}
				}
			}
		}
		if v == nil {
			http.Error(w, `{"status_message":"fixture not found"}`, http.StatusNotFound)
			return
		}
	}
	writeJSON(w, v)
}

func moviePath(path string) (int, string, bool) {
	if !strings.HasPrefix(path, "/movie/") {
		return 0, "", false
	}
	parts := strings.Split(strings.TrimPrefix(path, "/movie/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}
	if len(parts) > 1 {
		return id, "/" + strings.Join(parts[1:], "/"), true
	}
	return id, "", true
}
func findMovie(id int) map[string]any {
	for _, movie := range movies {
		if movie["id"].(int) == id {
			return movie
		}
	}
	return nil
}
func asAny(items []map[string]any) []any {
	out := make([]any, len(items))
	for i := range items {
		out[i] = items[i]
	}
	return out
}
func page(items []any) map[string]any {
	return map[string]any{"page": 1, "total_pages": 1, "total_results": len(items), "results": items}
}
func movieDetail(movie map[string]any) map[string]any {
	out := make(map[string]any, len(movie)+2)
	for k, v := range movie {
		out[k] = v
	}
	out["runtime"] = 1
	out["genres"] = []any{map[string]any{"id": 12, "name": "Adventure"}, map[string]any{"id": 878, "name": "Science Fiction"}}
	return out
}
func show() map[string]any {
	return map[string]any{"id": 1396, "name": "Breaking Bad", "original_name": "Breaking Bad", "first_air_date": "2008-01-20", "overview": "A chemistry teacher turns to manufacturing after a diagnosis.", "vote_average": 8.9, "poster_path": "/breaking.png", "backdrop_path": "/breaking-bg.png"}
}
func showDetail() map[string]any {
	s := show()
	s["number_of_seasons"] = 1
	s["genres"] = []any{map[string]any{"id": 18, "name": "Drama"}}
	s["seasons"] = []any{map[string]any{"id": 3572, "name": "Season 1", "season_number": 1, "episode_count": 2, "air_date": "2008-01-20", "poster_path": "/breaking-s1.png"}}
	return s
}
func season() map[string]any {
	return map[string]any{"id": 3572, "name": "Season 1", "season_number": 1, "episodes": []any{episode(1, "Pilot"), episode(2, "Cat's in the Bag...")}}
}
func episode(n int, name string) map[string]any {
	return map[string]any{"id": 62085 + n, "name": name, "season_number": 1, "episode_number": n, "air_date": "2008-01-20", "overview": "Synthetic metadata for an integration test.", "vote_average": 8.1, "still_path": "/breaking-e.png"}
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
