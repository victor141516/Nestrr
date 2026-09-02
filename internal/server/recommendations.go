package server

import (
	"net/http"
	"strconv"
)

func (s *Server) recommendations(w http.ResponseWriter, r *http.Request) {
	if s.Recommendations == nil {
		problem(w, http.StatusServiceUnavailable, "recommendations_unavailable", "Recommendations are unavailable.")
		return
	}
	mediaType := r.URL.Query().Get("kind")
	if mediaType != "tv" {
		mediaType = "movie"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	items, err := s.Recommendations.Recommend(r.Context(), user.ID, user.Language, mediaType, limit)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"results": items, "total_results": len(items), "media_type": mediaType})
}

func (s *Server) recommendationCatalogStatus(w http.ResponseWriter, r *http.Request) {
	if s.Recommendations == nil {
		problem(w, http.StatusServiceUnavailable, "recommendations_unavailable", "Recommendations are unavailable.")
		return
	}
	status, err := s.Recommendations.Status(r.Context())
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, status)
}

func (s *Server) runRecommendationCatalog(w http.ResponseWriter, r *http.Request) {
	if s.Recommendations == nil {
		problem(w, http.StatusServiceUnavailable, "recommendations_unavailable", "Recommendations are unavailable.")
		return
	}
	status, started, err := s.Recommendations.RunAsync()
	if err != nil {
		internal(w, err)
		return
	}
	code := http.StatusAccepted
	if !started {
		code = http.StatusOK
	}
	jsonResponse(w, code, status)
}
