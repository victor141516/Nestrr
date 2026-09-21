# Versioned API contracts

All Nestrr client operations live under `/api/v1`. Except setup/bootstrap/login/refresh and artwork caching, endpoints require `Authorization: Bearer <access-token>`. Errors use `{"error":{"code":"stable_code","message":"safe message"}}`.

## Nestrr

- `GET /setup` exposes `rottenTomatoesEnabled` when MDBList or OMDb is configured; `POST /auth/bootstrap`; `POST /auth/login`; `POST /auth/refresh`; `POST /auth/logout`.
- `GET /me`; `PATCH /me/settings` accepts the account-scoped `language` and `accentColor` preferences; `DELETE /me/metadata`.
- `GET /catalog/{operation}` proxies versioned search, discover, genre, movie, collection, show, season, episode, and recommendation operations while preserving TMDB JSON. Movie and series objects gain `rotten_tomatoes_score` when the optional permanently cached rating lookup succeeds; `refresh=true` forces both metadata and rating refresh.
- `POST /availability/check` accepts localized display names plus TMDB identity fields such as `originalTitle`, `originalShow`, `originalLanguage`, and `showTmdbId`. Filesystem resolution always derives the canonical English TMDB name; localized names are never used as paths. `GET /availability/stream?items=<base64url-json>` emits one `availability` SSE event per item.
- `GET /library` enumerates canonical files and attaches resolved TMDB metadata when available.
- `GET /progress`; `GET /progress/{movie|episode}/{tmdbId}`; `POST /progress/{movie|tv|episode}/{id}/watched`; `POST /progress/episodes/watched`; `POST /playback/events`; `POST /playback/navigation`; `POST /playback/grant` resolves a catalog identity to a local file and creates a Plarr grant when configured.
- Lists: `GET/POST /lists`, `PATCH/DELETE /lists/{id}`, `GET/POST /lists/{id}/items`, `DELETE /lists/{id}/items/{movie|tv}/{tmdbId}`, and `GET /list-memberships`. List item snapshots include `overview` and `addedAt`; a valid `addedAt` sent when restoring an item preserves its original inclusion date. Every user receives immutable Favorites, Watch Later, and Watched lists; custom lists use the same item model but can be renamed or deleted.
- `GET /integrations/status`; `GET /torrents/search?q=...`; `POST /torrents/submit` with `URL`, `Category`, `MediaType`, and `TMDBID`.
- Administrator: `GET/POST /admin/users`, `PATCH/DELETE /admin/users/{id}`, `PUT /admin/users/{id}/password`, `POST /admin/users/{id}/impersonate`, `POST /admin/impersonation/stop`, and `GET /admin/state`.

Access tokens are JWTs with a two-hour default lifetime. Browser refresh tokens are rotating HttpOnly, SameSite=Strict cookies and refresh enforces the configured origin. Native clients may request/return the refresh token in JSON.

Plarr events use `source: "plarr"`, `version: 1`, UUID-like `eventId`/`sessionId`, absolute position/duration, media type and TMDB ID. Event IDs are idempotent. `qualified` indicates that the initial 30 seconds of continuous playback since the latest seek have elapsed. Before qualification Nestrr may retain track preferences, but does not expose resumable progress. Once progress exists, subsequent positions are stored immediately. At 95%, playback completion adds the movie or fully completed series to Watched and removes it from Watch Later; manual Watched-list changes do not alter playback progress.

## Plarr

Plarr's public API is `/api/v1`. When `PLARR_API_KEY` is configured, all media and playback-session endpoints require a file-scoped `playback_token`; health, configuration, capabilities, static assets, CORS preflight, and grant issuance remain public routes:

- `GET /config`, `/capabilities`.
- `GET /media/{base64url-path}/probe`, `/original`, `/download/original`, `/download/{quality}`.
- `GET /media/{id}/hls/master.m3u8`, `/hls/{quality}/index.m3u8`, segments, audio playlists/segments, and `/subtitles/{stream}.vtt`.
- `POST /auth/playback-grants` accepts the shared API key and a media id server-to-server. Grants activate within one minute and expire after one hour without a valid request.

Every media request decodes the relative path, rejects absolute/drive/traversal input, resolves symlinks/realpaths, enforces containment under `PLARR_MEDIA_ROOT`, and accepts only supported video extensions.

The iframe protocol is `postMessage` envelope version 1. Types are `ready`, `opened`, `play`, `pause`, `seek`, `heartbeat`, `ended`, `closed`, `page-hidden`, audio/subtitle changes, automatic/manual rendition changes, next/previous requests, authorization expiration, and playback errors. The child posts only to an exact configured parent origin; Nestrr validates source window, origin, protocol, and envelope before ingestion.
