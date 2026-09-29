package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) createPlaybackRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var in availabilityInput
	if !decode(w, r, &in) {
		return
	}
	availability := s.resolveAvailability(r.Context(), in)
	if !availability.Available || availability.RelativePath == "" || in.Kind == "show" {
		problem(w, http.StatusNotFound, "media_unavailable", "The requested media is not available for playback.")
		return
	}
	mediaID := base64.RawURLEncoding.EncodeToString([]byte(availability.RelativePath))
	body, err := json.Marshal(map[string]string{"apiKey": s.Config.PlarrAPIKey, "mediaId": mediaID})
	if err != nil {
		internal(w, err)
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, s.Config.PlarrInternalHost+"/api/v1/watch-parties", bytes.NewReader(body))
	if err != nil {
		internal(w, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	client := s.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		problem(w, http.StatusBadGateway, "plarr_unavailable", "Watch party creation is unavailable.")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		problem(w, http.StatusBadGateway, "plarr_room_failed", "The watch party could not be created.")
		return
	}
	var room struct {
		RoomID    string `json:"roomId"`
		PlayerURL string `json:"playerUrl"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&room) != nil || room.RoomID == "" || room.PlayerURL == "" {
		problem(w, http.StatusBadGateway, "plarr_room_invalid", "The watch party response is invalid.")
		return
	}
	jsonResponse(w, http.StatusCreated, room)
}
