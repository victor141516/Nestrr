package recommendations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/victor141516/Nestrr/internal/store"
	"github.com/victor141516/Nestrr/internal/tmdb"
)

const (
	weeklyInterval   = 7 * 24 * time.Hour
	hydrationWorkers = 6
)

type SignalCounts struct {
	Genres    int `json:"genres"`
	Keywords  int `json:"keywords"`
	Actors    int `json:"actors"`
	Directors int `json:"directors"`
	Companies int `json:"companies"`
}

type WorkProgress struct {
	Total  int `json:"total"`
	Done   int `json:"done"`
	Found  int `json:"found,omitempty"`
	Added  int `json:"added,omitempty"`
	Failed int `json:"failed"`
}

type RunStatus struct {
	ID           int64        `json:"id,omitempty"`
	State        string       `json:"state"`
	Phase        string       `json:"phase"`
	StartedAt    string       `json:"startedAt,omitempty"`
	UpdatedAt    string       `json:"updatedAt,omitempty"`
	FinishedAt   string       `json:"finishedAt,omitempty"`
	UsersScanned int          `json:"usersScanned"`
	Signals      SignalCounts `json:"signals"`
	Discovery    WorkProgress `json:"discovery"`
	Hydration    WorkProgress `json:"hydration"`
	CatalogCount int          `json:"catalogCount"`
	Pending      int          `json:"pending"`
	LastError    string       `json:"lastError,omitempty"`
}

type Service struct {
	Store *store.Store
	TMDB  *tmdb.Client
	Log   *slog.Logger

	mu                sync.RWMutex
	persist           sync.Mutex
	running           bool
	status            RunStatus
	root              context.Context
	limiter           *time.Ticker
	lastStatusPersist time.Time
}

func New(database *store.Store, client *tmdb.Client, log *slog.Logger) *Service {
	return &Service{Store: database, TMDB: client, Log: log}
}

func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	s.root = ctx
	if s.limiter == nil {
		s.limiter = time.NewTicker(60 * time.Millisecond)
	}
	s.mu.Unlock()
	go func() {
		delay := time.NewTimer(30 * time.Second)
		defer delay.Stop()
		select {
		case <-ctx.Done():
			return
		case <-delay.C:
			latest, err := s.Store.LatestRecommendationRun(ctx)
			if errors.Is(err, sql.ErrNoRows) || (err == nil && runIsDue(latest)) {
				_, _, _ = s.RunAsync()
			}
		}
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.mu.Lock()
				if s.limiter != nil {
					s.limiter.Stop()
					s.limiter = nil
				}
				s.mu.Unlock()
				return
			case <-ticker.C:
				latest, err := s.Store.LatestRecommendationRun(ctx)
				if errors.Is(err, sql.ErrNoRows) || (err == nil && runIsDue(latest)) {
					_, _, _ = s.RunAsync()
				}
			}
		}
	}()
}

func runIsDue(run store.RecommendationRunRecord) bool {
	if run.State == "running" {
		return true
	}
	stamp := run.FinishedAt
	if stamp == "" {
		stamp = run.UpdatedAt
	}
	parsed, err := time.Parse(time.RFC3339Nano, stamp)
	return err != nil || time.Since(parsed) >= weeklyInterval
}

func (s *Service) RunAsync() (RunStatus, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return s.status, false, nil
	}
	ctx := s.root
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	status := RunStatus{State: "running", Phase: "signals", StartedAt: now, UpdatedAt: now}
	payload, _ := json.Marshal(status)
	id, err := s.Store.CreateRecommendationRun(ctx, status.State, payload)
	if err != nil {
		return RunStatus{}, false, err
	}
	status.ID = id
	s.status = status
	s.running = true
	go s.run(ctx, id)
	return status, true, nil
}

func (s *Service) Status(ctx context.Context) (RunStatus, error) {
	s.mu.RLock()
	if s.running {
		status := s.status
		s.mu.RUnlock()
		return status, nil
	}
	s.mu.RUnlock()
	record, err := s.Store.LatestRecommendationRun(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return RunStatus{}, err
	}
	status := RunStatus{State: "idle", Phase: "idle"}
	if err == nil {
		if decodeErr := json.Unmarshal(record.Payload, &status); decodeErr != nil {
			return RunStatus{}, decodeErr
		}
	}
	status.CatalogCount, _ = s.Store.CatalogMediaCount(ctx)
	counts, _ := s.Store.RecommendationCandidateCounts(ctx)
	status.Pending = counts["pending"] + counts["processing"] + counts["failed"]
	return status, nil
}

func (s *Service) run(ctx context.Context, id int64) {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()
	if err := s.Store.ResetRecommendationCandidates(ctx); err != nil {
		s.finish(ctx, id, err)
		return
	}
	seeds, err := s.Store.RecommendationSeeds(ctx, 0)
	if err != nil {
		s.finish(ctx, id, err)
		return
	}
	users, seedKeys := distinctSeeds(seeds)
	catalogCount, _ := s.Store.CatalogMediaCount(ctx)
	s.update(ctx, id, func(status *RunStatus) {
		status.UsersScanned = len(users)
		status.CatalogCount = catalogCount
	})
	missing, err := s.Store.MissingCatalogKeys(ctx, seedKeys)
	if err != nil {
		s.finish(ctx, id, err)
		return
	}
	if _, err = s.Store.AddRecommendationCandidates(ctx, missing); err != nil {
		s.finish(ctx, id, err)
		return
	}
	s.setPhase(ctx, id, "hydration")
	if err = s.hydratePending(ctx, id); err != nil {
		s.finish(ctx, id, err)
		return
	}

	seedFeatures, err := s.Store.CatalogFeaturesForKeys(ctx, seedKeys)
	if err != nil {
		s.finish(ctx, id, err)
		return
	}
	signals := signalsForSeeds(seedFeatures, seeds)
	tasks := discoveryTasks(signals)
	s.update(ctx, id, func(status *RunStatus) {
		status.Phase = "discovery"
		status.Signals = signals.counts()
		status.Discovery = WorkProgress{Total: len(tasks)}
	})
	for _, task := range tasks {
		keys, discoverErr := s.discoverTask(ctx, task)
		if discoverErr != nil {
			s.Log.Warn("recommendation discovery failed", "mediaType", task.MediaType, "signal", task.SignalType, "signalId", task.SignalID, "error", discoverErr)
			s.update(ctx, id, func(status *RunStatus) { status.Discovery.Done++; status.Discovery.Failed++ })
			continue
		}
		added, addErr := s.Store.AddRecommendationCandidates(ctx, keys)
		if addErr != nil {
			s.finish(ctx, id, addErr)
			return
		}
		s.update(ctx, id, func(status *RunStatus) {
			status.Discovery.Done++
			status.Discovery.Found += len(keys)
			status.Discovery.Added += added
		})
	}
	s.setPhase(ctx, id, "hydration")
	if err = s.hydratePending(ctx, id); err != nil {
		s.finish(ctx, id, err)
		return
	}
	s.finish(ctx, id, nil)
}

func distinctSeeds(seeds []store.RecommendationSeed) (map[int64]bool, []store.CatalogKey) {
	users := map[int64]bool{}
	seen := map[store.CatalogKey]bool{}
	keys := []store.CatalogKey{}
	for _, seed := range seeds {
		users[seed.UserID] = true
		if !seen[seed.CatalogKey] {
			seen[seed.CatalogKey] = true
			keys = append(keys, seed.CatalogKey)
		}
	}
	return users, keys
}

type signalSet struct {
	Genres, Keywords, Actors, Directors, Companies map[int]bool
}

func (s signalSet) counts() SignalCounts {
	return SignalCounts{Genres: len(s.Genres), Keywords: len(s.Keywords), Actors: len(s.Actors), Directors: len(s.Directors), Companies: len(s.Companies)}
}

func signalsForSeeds(featuresByKey map[store.CatalogKey]store.CatalogFeatures, seeds []store.RecommendationSeed) signalSet {
	result := signalSet{Genres: map[int]bool{}, Keywords: map[int]bool{}, Actors: map[int]bool{}, Directors: map[int]bool{}, Companies: map[int]bool{}}
	for _, seed := range seeds {
		features, ok := featuresByKey[seed.CatalogKey]
		if !ok {
			continue
		}
		addIDs(result.Genres, features.Genres)
		addIDs(result.Keywords, features.Keywords)
		addIDs(result.Actors, features.Cast)
		addIDs(result.Directors, features.Directors)
		addIDs(result.Companies, features.Companies)
	}
	return result
}

func addIDs(target map[int]bool, values []int) {
	for _, value := range values {
		if value > 0 {
			target[value] = true
		}
	}
}

type discoveryTask struct {
	MediaType, SignalType string
	SignalID              int
}

func discoveryTasks(signals signalSet) []discoveryTask {
	tasks := []discoveryTask{}
	for _, mediaType := range []string{"movie", "tv"} {
		for _, definition := range []struct {
			name string
			ids  map[int]bool
		}{{"genre", signals.Genres}, {"keyword", signals.Keywords}, {"company", signals.Companies}, {"actor", signals.Actors}, {"director", signals.Directors}} {
			ids := make([]int, 0, len(definition.ids))
			for id := range definition.ids {
				ids = append(ids, id)
			}
			sort.Ints(ids)
			for _, id := range ids {
				tasks = append(tasks, discoveryTask{MediaType: mediaType, SignalType: definition.name, SignalID: id})
			}
		}
	}
	return tasks
}

func (s *Service) discoverTask(ctx context.Context, task discoveryTask) ([]store.CatalogKey, error) {
	if task.MediaType == "tv" && (task.SignalType == "actor" || task.SignalType == "director") {
		return s.discoverPersonTV(ctx, task)
	}
	cursor, err := s.Store.RecommendationDiscoveryPage(ctx, task.MediaType, task.SignalType, task.SignalID, "popularity.desc")
	if err != nil {
		return nil, err
	}
	pages := []int{1}
	if cursor > 1 {
		pages = append(pages, cursor)
	}
	seen := map[store.CatalogKey]bool{}
	keys := []store.CatalogKey{}
	lastTotalPages := 500
	for _, page := range pages {
		query := url.Values{"sort_by": {"popularity.desc"}, "include_adult": {"false"}, "page": {strconv.Itoa(page)}}
		parameter := map[string]string{"genre": "with_genres", "keyword": "with_keywords", "company": "with_companies", "actor": "with_cast", "director": "with_crew"}[task.SignalType]
		query.Set(parameter, strconv.Itoa(task.SignalID))
		response, requestErr := s.tmdbGet(ctx, "/discover/"+task.MediaType, query)
		if requestErr != nil {
			return nil, requestErr
		}
		var listing struct {
			Page       int `json:"page"`
			TotalPages int `json:"total_pages"`
			Results    []struct {
				ID int `json:"id"`
			} `json:"results"`
		}
		if err := json.Unmarshal(response, &listing); err != nil {
			return nil, err
		}
		if listing.TotalPages > 0 {
			lastTotalPages = listing.TotalPages
		}
		for _, result := range listing.Results {
			key := store.CatalogKey{MediaType: task.MediaType, TMDBID: result.ID}
			if result.ID > 0 && !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	next := cursor + 1
	if cursor == 1 {
		next = 2
	}
	if next > lastTotalPages || next > 500 {
		next = 1
	}
	if err := s.Store.SetRecommendationDiscoveryPage(ctx, task.MediaType, task.SignalType, task.SignalID, "popularity.desc", next); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *Service) discoverPersonTV(ctx context.Context, task discoveryTask) ([]store.CatalogKey, error) {
	response, err := s.tmdbGet(ctx, fmt.Sprintf("/person/%d/tv_credits", task.SignalID), url.Values{"language": {"en-US"}})
	if err != nil {
		return nil, err
	}
	var credits struct {
		Cast []struct {
			ID int `json:"id"`
		} `json:"cast"`
		Crew []struct {
			ID  int    `json:"id"`
			Job string `json:"job"`
		} `json:"crew"`
	}
	if err := json.Unmarshal(response, &credits); err != nil {
		return nil, err
	}
	keys := []store.CatalogKey{}
	seen := map[int]bool{}
	if task.SignalType == "actor" {
		for _, item := range credits.Cast {
			if item.ID > 0 && !seen[item.ID] {
				seen[item.ID] = true
				keys = append(keys, store.CatalogKey{MediaType: "tv", TMDBID: item.ID})
			}
		}
	} else {
		for _, item := range credits.Crew {
			if item.ID > 0 && !seen[item.ID] {
				seen[item.ID] = true
				keys = append(keys, store.CatalogKey{MediaType: "tv", TMDBID: item.ID})
			}
		}
	}
	return keys, nil
}

func (s *Service) hydratePending(ctx context.Context, runID int64) error {
	keys, err := s.Store.PendingRecommendationCandidates(ctx, 0)
	if err != nil {
		return err
	}
	s.update(ctx, runID, func(status *RunStatus) {
		status.Hydration.Total += len(keys)
		status.Pending += len(keys)
	})
	jobs := make(chan store.CatalogKey)
	var workers sync.WaitGroup
	for range hydrationWorkers {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for key := range jobs {
				if startErr := s.Store.StartRecommendationCandidate(ctx, key); startErr != nil {
					s.recordHydration(ctx, runID, true)
					continue
				}
				if hydrateErr := s.hydrate(ctx, key); hydrateErr != nil {
					_ = s.Store.FailRecommendationCandidate(ctx, key, hydrateErr.Error())
					s.Log.Warn("recommendation candidate hydration failed", "mediaType", key.MediaType, "tmdbId", key.TMDBID, "error", hydrateErr)
					s.recordHydration(ctx, runID, true)
					continue
				}
				s.recordHydration(ctx, runID, false)
			}
		}()
	}
	for _, key := range keys {
		select {
		case <-ctx.Done():
			close(jobs)
			workers.Wait()
			return ctx.Err()
		case jobs <- key:
		}
	}
	close(jobs)
	workers.Wait()
	return ctx.Err()
}

func (s *Service) recordHydration(ctx context.Context, runID int64, failed bool) {
	refreshCatalog := false
	s.update(ctx, runID, func(status *RunStatus) {
		status.Hydration.Done++
		if failed {
			status.Hydration.Failed++
		}
		if status.Pending > 0 {
			status.Pending--
		}
		refreshCatalog = status.Hydration.Done%25 == 0
	})
	if !refreshCatalog {
		return
	}
	count, err := s.Store.CatalogMediaCount(ctx)
	if err != nil {
		return
	}
	s.update(ctx, runID, func(status *RunStatus) { status.CatalogCount = count })
}

func (s *Service) hydrate(ctx context.Context, key store.CatalogKey) error {
	appendResources := "credits,keywords,translations"
	if key.MediaType == "movie" {
		appendResources += ",release_dates"
	} else {
		appendResources += ",content_ratings"
	}
	payload, err := s.tmdbGet(ctx, fmt.Sprintf("/%s/%d", key.MediaType, key.TMDBID), url.Values{"language": {"en-US"}, "append_to_response": {appendResources}})
	if err != nil {
		return err
	}
	parsed, err := parseCompleteMedia(key.MediaType, key.TMDBID, payload)
	if err != nil {
		return err
	}
	if err = s.Store.ReplaceMediaCredits(ctx, key.MediaType, key.TMDBID, parsed.People, parsed.Credits); err != nil {
		return err
	}
	if err = s.Store.ReplaceMediaCompanies(ctx, key.MediaType, key.TMDBID, parsed.Companies); err != nil {
		return err
	}
	if err = s.Store.ReplaceMediaGenres(ctx, key.MediaType, key.TMDBID, "en-US", parsed.Genres); err != nil {
		return err
	}
	if err = s.Store.ReplaceMediaKeywords(ctx, key.MediaType, key.TMDBID, parsed.Keywords); err != nil {
		return err
	}
	return s.Store.PutCatalogMedia(ctx, parsed.Catalog)
}

func (s *Service) tmdbGet(ctx context.Context, path string, query url.Values) ([]byte, error) {
	s.mu.RLock()
	limiter := s.limiter
	s.mu.RUnlock()
	if limiter != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-limiter.C:
		}
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		response, err := s.TMDB.Get(ctx, path, query, false)
		if err == nil {
			return response.Body, nil
		}
		last = err
		if response.Status != 429 && response.Status < 500 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * time.Second):
		}
	}
	return nil, last
}

func (s *Service) setPhase(ctx context.Context, id int64, phase string) {
	s.update(ctx, id, func(status *RunStatus) { status.Phase = phase })
	s.persistCurrentStatus(ctx, id)
}

func (s *Service) update(ctx context.Context, id int64, mutate func(*RunStatus)) {
	s.mu.Lock()
	mutate(&s.status)
	s.status.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	shouldPersist := time.Since(s.lastStatusPersist) >= time.Second
	if shouldPersist {
		s.lastStatusPersist = time.Now()
	}
	s.mu.Unlock()
	if shouldPersist {
		s.persistCurrentStatus(ctx, id)
	}
}

func (s *Service) persistCurrentStatus(ctx context.Context, id int64) {
	s.persist.Lock()
	defer s.persist.Unlock()
	s.mu.RLock()
	status := s.status
	s.mu.RUnlock()
	payload, _ := json.Marshal(status)
	if err := s.Store.UpdateRecommendationRun(ctx, id, status.State, payload, false); err != nil {
		s.Log.Warn("recommendation run status persistence failed", "runId", id, "error", err)
	}
}

func (s *Service) finish(ctx context.Context, id int64, runErr error) {
	s.mu.Lock()
	if runErr != nil {
		s.status.State = "failed"
		s.status.LastError = runErr.Error()
	} else {
		s.status.State = "completed"
		s.status.Phase = "completed"
	}
	s.status.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	s.status.UpdatedAt = s.status.FinishedAt
	s.status.CatalogCount, _ = s.Store.CatalogMediaCount(ctx)
	counts, _ := s.Store.RecommendationCandidateCounts(ctx)
	s.status.Pending = counts["pending"] + counts["processing"] + counts["failed"]
	status := s.status
	s.mu.Unlock()
	s.persist.Lock()
	defer s.persist.Unlock()
	payload, _ := json.Marshal(status)
	if err := s.Store.UpdateRecommendationRun(ctx, id, status.State, payload, true); err != nil {
		s.Log.Warn("recommendation run final status persistence failed", "runId", id, "error", err)
	}
}
