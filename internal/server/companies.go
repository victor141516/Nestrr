package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
)

const companySearchTTL = 30 * 24 * time.Hour

type tmdbCompany struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	LogoPath      string `json:"logo_path"`
	OriginCountry string `json:"origin_country"`
}

func storedCompany(company tmdbCompany) store.Company {
	return store.Company{
		TMDBID:        company.ID,
		Name:          strings.TrimSpace(company.Name),
		LogoPath:      strings.TrimSpace(company.LogoPath),
		OriginCountry: strings.TrimSpace(company.OriginCountry),
	}
}

func parseMovieCompanies(payload []byte) ([]store.Company, error) {
	var document struct {
		ProductionCompanies []tmdbCompany `json:"production_companies"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		return nil, err
	}
	companies := make([]store.Company, 0, len(document.ProductionCompanies))
	for _, source := range document.ProductionCompanies {
		company := storedCompany(source)
		if company.TMDBID > 0 && company.Name != "" {
			companies = append(companies, company)
		}
	}
	return companies, nil
}

func (s *Server) storeMovieCompanies(r *http.Request, movieID int, payload []byte) {
	companies, err := parseMovieCompanies(payload)
	if err == nil {
		err = s.Store.ReplaceMediaCompanies(r.Context(), "movie", movieID, companies)
	}
	if err != nil && s.Log != nil {
		s.Log.Warn("TMDB companies could not be indexed", "operation", "companies.index", "tmdbId", movieID, "error", err)
	}
}

func (s *Server) companySearch(w http.ResponseWriter, r *http.Request) {
	if rawIDs := strings.TrimSpace(r.URL.Query().Get("ids")); rawIDs != "" {
		ids, err := parsePositiveIDs(rawIDs, 50)
		if err != nil {
			problem(w, http.StatusBadRequest, "invalid_company_ids", err.Error())
			return
		}
		companies, err := s.Store.CompaniesByIDs(r.Context(), ids)
		if err != nil {
			internal(w, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": companies})
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(query)) < 2 {
		problem(w, http.StatusBadRequest, "company_query_too_short", "Enter at least two characters.")
		return
	}
	limit := 12
	if parsed, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && parsed > 0 {
		limit = min(parsed, 30)
	}
	phase := r.URL.Query().Get("phase")
	if phase == "" || phase == "local" {
		companies, err := s.Store.SearchCompanies(r.Context(), query, limit)
		if err != nil {
			internal(w, err)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": companies, "phase": "local"})
		return
	}
	if phase != "remote" {
		problem(w, http.StatusBadRequest, "invalid_company_phase", "Company search phase must be local or remote.")
		return
	}

	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	if cached, fresh, err := s.Store.CachedCompanySearch(r.Context(), query, user.Language, companySearchTTL); err != nil {
		internal(w, err)
		return
	} else if fresh {
		if len(cached) > limit {
			cached = cached[:limit]
		}
		jsonResponse(w, http.StatusOK, map[string]any{"results": cached, "phase": "remote", "cached": true})
		return
	}
	response, err := s.TMDB.Get(r.Context(), "/search/company", url.Values{
		"query":    {query},
		"language": {user.Language},
		"page":     {"1"},
	}, false)
	if err != nil {
		problem(w, http.StatusBadGateway, "company_search_failed", "Companies could not be searched.")
		return
	}
	var document struct {
		Results []tmdbCompany `json:"results"`
	}
	if err := json.Unmarshal(response.Body, &document); err != nil {
		internal(w, err)
		return
	}
	companies := make([]store.Company, 0, len(document.Results))
	for _, source := range document.Results {
		company := storedCompany(source)
		if company.TMDBID > 0 && company.Name != "" {
			companies = append(companies, company)
		}
	}
	if err := s.Store.PutCompanySearch(r.Context(), query, user.Language, companies); err != nil {
		internal(w, err)
		return
	}
	if len(companies) > limit {
		companies = companies[:limit]
	}
	jsonResponse(w, http.StatusOK, map[string]any{"results": companies, "phase": "remote", "cached": false})
}

func (s *Server) localMediaByCompanies(w http.ResponseWriter, r *http.Request) {
	companyIDs, err := parsePositiveIDs(r.URL.Query().Get("companies"), 8)
	if err != nil {
		problem(w, http.StatusBadRequest, "invalid_company_filter", "Company filters must contain positive identifiers.")
		return
	}
	if len(companyIDs) == 0 {
		jsonResponse(w, http.StatusOK, map[string]any{"results": []any{}})
		return
	}
	user, err := s.Store.UserByID(r.Context(), claims(r).UserID)
	if err != nil {
		internal(w, err)
		return
	}
	results, err := s.Store.LocalMediaForCompanies(r.Context(), user.ID, user.Language, "movie", companyIDs)
	if err != nil {
		internal(w, err)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"results": results})
}
