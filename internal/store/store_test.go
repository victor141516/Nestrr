package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func TestSingleAdministratorBootstrap(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, err := s.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil || !u.IsAdmin {
		t.Fatalf("bootstrap: %v %#v", err, u)
	}
	if _, err = s.CreateFirstAdmin(ctx, "other", "hash", "en-US"); err == nil {
		t.Fatal("second administrator was accepted")
	}
}

func TestUserPreferencesPersistAccentColor(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, err := s.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	if u.AccentColor != "#ff624f" {
		t.Fatalf("unexpected default accent: %q", u.AccentColor)
	}
	u, err = s.SetPreferences(ctx, u.ID, "en-US", "#38bdf8")
	if err != nil {
		t.Fatal(err)
	}
	if u.Language != "en-US" || u.AccentColor != "#38bdf8" {
		t.Fatalf("preferences were not persisted: %#v", u)
	}
}
func TestPlaybackCompletionRulesAndResume(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	p := Progress{MediaType: "movie", TMDBID: 10, PositionSeconds: 96, DurationSeconds: 100, LastEventID: "one", LastSessionID: "s"}
	got, err := s.UpsertProgress(ctx, u.ID, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Watched {
		t.Fatal("unqualified progress above 95% marked watched")
	}
	p.LastEventID = "two"
	got, err = s.UpsertProgress(ctx, u.ID, p, true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Watched || got.ResumeSeconds != 91 {
		t.Fatalf("qualified completion rule failed: %#v", got)
	}
}

func TestInitialProgressRequiresQualificationThenUpdatesImmediately(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	p := Progress{MediaType: "movie", TMDBID: 20, PositionSeconds: 40, DurationSeconds: 100, LastEventID: "gate-one", LastSessionID: "s"}
	got, err := s.UpsertProgress(ctx, u.ID, p, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Started || got.PositionSeconds != 0 {
		t.Fatalf("unqualified first playback became started: %#v", got)
	}
	items, _ := s.ListProgress(ctx, u.ID)
	if len(items) != 0 {
		t.Fatalf("pre-threshold row leaked into progress: %#v", items)
	}
	p.LastEventID = "gate-two"
	got, err = s.UpsertProgress(ctx, u.ID, p, true)
	if err != nil || !got.Started || got.PositionSeconds != 40 {
		t.Fatalf("qualified playback was not stored: %#v %v", got, err)
	}
	p.PositionSeconds, p.LastEventID = 96, "gate-three"
	got, err = s.UpsertProgress(ctx, u.ID, p, false)
	if err != nil || !got.Watched || got.PositionSeconds != 96 {
		t.Fatalf("existing progress did not update immediately: %#v %v", got, err)
	}
}

func TestMediaListsArePerUserAndSystemListsCannotBeDeleted(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	admin, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	member, _ := s.CreateUser(ctx, admin.ID, "member", "hash", "en-US")
	lists, err := s.MediaLists(ctx, member.ID)
	if err != nil || len(lists) != 3 {
		t.Fatalf("default lists: %#v %v", lists, err)
	}
	watched, _ := s.SystemMediaList(ctx, member.ID, "watched")
	item := MediaListItem{MediaType: "movie", TMDBID: 42, Title: "A Movie"}
	if err := s.AddMediaListItem(ctx, member.ID, watched.ID, item); err != nil {
		t.Fatal(err)
	}
	memberships, _ := s.ListMemberships(ctx, member.ID)
	if len(memberships) != 1 || memberships[0].SystemKey != "watched" {
		t.Fatalf("unexpected memberships: %#v", memberships)
	}
	adminMemberships, _ := s.ListMemberships(ctx, admin.ID)
	if len(adminMemberships) != 0 {
		t.Fatalf("member list leaked to administrator: %#v", adminMemberships)
	}
	if err := s.DeleteMediaList(ctx, member.ID, watched.ID); err == nil {
		t.Fatal("system list was deletable")
	}
}

func TestMediaListItemRestoreKeepsOriginalDateAndOverview(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	list, _ := s.SystemMediaList(ctx, u.ID, "favorites")
	addedAt := "2025-03-14T09:26:53.123456789Z"
	item := MediaListItem{
		MediaType: "movie",
		TMDBID:    550,
		Title:     "Fight Club",
		Overview:  "An insomniac meets a soap maker.",
		AddedAt:   addedAt,
	}
	if err := s.AddMediaListItem(ctx, u.ID, list.ID, item); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveMediaListItem(ctx, u.ID, list.ID, "movie", 550); err != nil {
		t.Fatal(err)
	}
	if err := s.AddMediaListItem(ctx, u.ID, list.ID, item); err != nil {
		t.Fatal(err)
	}
	items, err := s.MediaListItems(ctx, u.ID, list.ID, "", "addedAt", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].AddedAt != addedAt || items[0].Overview != item.Overview {
		t.Fatalf("restored item lost its snapshot: %#v", items)
	}
}

func TestSeriesCompletionIncludesReleasedSpecials(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	showID := 77
	_ = s.PutMetadata(ctx, "tv", showID, "en-US", []byte(`{"id":77,"seasons":[{"season_number":0},{"season_number":1}]}`))
	_ = s.PutMetadata(ctx, "season", showID*1000, "en-US", []byte(`{"episodes":[{"id":700,"air_date":"2020-01-01"}]}`))
	_ = s.PutMetadata(ctx, "season", showID*1000+1, "en-US", []byte(`{"episodes":[{"id":701,"air_date":"2020-01-02"}]}`))
	if _, err := s.SetEpisodeWatched(ctx, u.ID, Progress{TMDBID: 701, ShowTMDBID: showID, SeasonNumber: 1, EpisodeNumber: 1}, true); err != nil {
		t.Fatal(err)
	}
	complete, err := s.ShowFullyWatched(ctx, u.ID, showID)
	if err != nil || complete {
		t.Fatalf("show completed without its special: %v %v", complete, err)
	}
	if _, err := s.SetEpisodeWatched(ctx, u.ID, Progress{TMDBID: 700, ShowTMDBID: showID, EpisodeNumber: 1}, true); err != nil {
		t.Fatal(err)
	}
	complete, err = s.ShowFullyWatched(ctx, u.ID, showID)
	if err != nil || !complete {
		t.Fatalf("show did not complete after every released episode: %v %v", complete, err)
	}
}

func TestEpisodeProgressKeepsSeriesContext(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	p := Progress{MediaType: "episode", TMDBID: 501, ShowTMDBID: 42, SeasonNumber: 3, EpisodeNumber: 7, PositionSeconds: 20, DurationSeconds: 100, LastEventID: "episode-one", LastSessionID: "s"}
	got, err := s.UpsertProgress(ctx, u.ID, p, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ShowTMDBID != 42 || got.SeasonNumber != 3 || got.EpisodeNumber != 7 {
		t.Fatalf("episode context was not persisted: %#v", got)
	}
}

func TestListProgressBackfillsLegacyEpisodeContextFromMetadata(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	p := Progress{MediaType: "episode", TMDBID: 987, PositionSeconds: 20, DurationSeconds: 100, LastEventID: "legacy-episode", LastSessionID: "s"}
	if _, err := s.UpsertProgress(ctx, u.ID, p, true); err != nil {
		t.Fatal(err)
	}
	cacheID := 42*1000 + 3
	payload := []byte(fmt.Sprintf(`{"season_number":3,"episodes":[{"id":%d,"episode_number":7}]}`, p.TMDBID))
	if err := s.PutMetadata(ctx, "season", cacheID, "en-US", payload); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListProgress(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ShowTMDBID != 42 || items[0].SeasonNumber != 3 || items[0].EpisodeNumber != 7 {
		t.Fatalf("legacy episode context was not recovered: %#v", items)
	}
}

func TestLatestTorrentAssociationSupportsLegacyTVKind(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	if err := s.AssociateTorrent(ctx, u.ID, "tv", 42, "0123456789abcdef0123456789abcdef01234567", "Shows"); err != nil {
		t.Fatal(err)
	}
	association, err := s.LatestTorrentAssociation(ctx, u.ID, "show", 42)
	if err != nil {
		t.Fatal(err)
	}
	if association.InfoHash != "0123456789abcdef0123456789abcdef01234567" || association.Category != "Shows" {
		t.Fatalf("unexpected association: %#v", association)
	}
}

func TestTorrentAssociationsReturnsEveryAssociatedTorrent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	u, _ := s.CreateFirstAdmin(ctx, "admin", "hash", "en-US")
	hashes := []string{
		"0123456789abcdef0123456789abcdef01234567",
		"89abcdef0123456789abcdef0123456789abcdef",
	}
	for _, hash := range hashes {
		if err := s.AssociateTorrent(ctx, u.ID, "movie", 99, hash, "Movies"); err != nil {
			t.Fatal(err)
		}
	}
	associations, err := s.TorrentAssociations(ctx, u.ID, "movie", 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(associations) != 2 {
		t.Fatalf("got %d associations, want 2: %#v", len(associations), associations)
	}
	got := map[string]bool{}
	for _, association := range associations {
		got[association.InfoHash] = true
	}
	for _, hash := range hashes {
		if !got[hash] {
			t.Fatalf("association %s is missing: %#v", hash, associations)
		}
	}
}
