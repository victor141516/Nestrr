package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/victor141516/Nestrr/internal/auth"
	"github.com/victor141516/Nestrr/internal/store"
)

func TestCompletedPlaybackMovesTitleFromWatchLaterToWatched(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	user, err := db.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	metadata := []byte(`{"id":550,"title":"El club de la lucha","original_title":"Fight Club","poster_path":"/poster.jpg","release_date":"1999-10-15","vote_average":8.4,"vote_count":30000,"popularity":80,"runtime":139}`)
	if err := db.PutMetadata(ctx, "movie", 550, "es-ES", metadata); err != nil {
		t.Fatal(err)
	}
	item := store.MediaListItem{MediaType: "movie", TMDBID: 550, Title: "El club de la lucha"}
	if err := db.SetSystemListMembership(ctx, user.ID, "watch-later", item, true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetSystemListMembership(ctx, user.ID, "favorites", item, true); err != nil {
		t.Fatal(err)
	}

	s := &Server{Store: db}
	s.syncCompletedTitle(ctx, user.ID, "movie", 550)
	memberships, err := db.ListMemberships(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, membership := range memberships {
		if membership.MediaType == "movie" && membership.TMDBID == 550 {
			keys[membership.SystemKey] = true
		}
	}
	if !keys["watched"] || !keys["favorites"] || keys["watch-later"] {
		t.Fatalf("unexpected memberships after completion: %#v", keys)
	}
}

func TestAddListItemRejectsInvalidRestoreDate(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	user, _ := db.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	list, _ := db.SystemMediaList(ctx, user.ID, "favorites")

	req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/lists/%d/items", list.ID), bytes.NewBufferString(`{"mediaType":"movie","tmdbId":550,"title":"Fight Club","addedAt":"yesterday"}`))
	route := chi.NewRouteContext()
	route.URLParams.Add("listID", fmt.Sprint(list.ID))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
	req = req.WithContext(context.WithValue(req.Context(), claimsKey, auth.Claims{UserID: user.ID}))
	response := httptest.NewRecorder()

	(&Server{Store: db}).addMediaListItem(response, req)
	if response.Code != 400 {
		t.Fatalf("invalid inclusion date returned %d: %s", response.Code, response.Body.String())
	}
}

func TestListPresentationFollowsUserLanguageFromSharedMetadata(t *testing.T) {
	ctx := context.Background()
	db, err := store.Open(filepath.Join(t.TempDir(), "nestrr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	user, err := db.CreateFirstAdmin(ctx, "admin", "hash", "es-ES")
	if err != nil {
		t.Fatal(err)
	}
	list, _ := db.SystemMediaList(ctx, user.ID, "favorites")
	if err := db.AddMediaListItem(ctx, user.ID, list.ID, store.MediaListItem{MediaType: "movie", TMDBID: 7, Title: "Stale snapshot"}); err != nil {
		t.Fatal(err)
	}
	if err := db.PutMetadata(ctx, "movie", 7, "es-ES", []byte(`{"id":7,"title":"Título español","overview":"ES"}`)); err != nil {
		t.Fatal(err)
	}
	if err := db.PutMetadata(ctx, "movie", 7, "en-US", []byte(`{"id":7,"title":"English title","overview":"EN"}`)); err != nil {
		t.Fatal(err)
	}
	server := &Server{Store: db}
	requestItems := func() string {
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/lists/%d/items", list.ID), nil)
		route := chi.NewRouteContext()
		route.URLParams.Add("listID", fmt.Sprint(list.ID))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
		req = req.WithContext(context.WithValue(req.Context(), claimsKey, auth.Claims{UserID: user.ID}))
		response := httptest.NewRecorder()
		server.listMediaListItems(response, req)
		if response.Code != 200 {
			t.Fatalf("list response %d: %s", response.Code, response.Body.String())
		}
		var body struct {
			Items []store.MediaListItem `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.Items) != 1 {
			t.Fatalf("invalid list response: %s %v", response.Body.String(), err)
		}
		return body.Items[0].Title
	}
	if title := requestItems(); title != "Título español" {
		t.Fatalf("Spanish shared translation not used: %q", title)
	}
	if _, err := db.SetPreferences(ctx, user.ID, "en-US", "#ff624f"); err != nil {
		t.Fatal(err)
	}
	if title := requestItems(); title != "English title" {
		t.Fatalf("language change did not select shared translation: %q", title)
	}
}
