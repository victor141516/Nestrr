# Deployment guide

## Topology and origins

`docker-compose.yml` runs Nestrr, Plarr, Jackett, FlareSolverr, qBittorrent, Chiprr, and Strastorr. Clone `Strastorr` alongside `Nestrr`, `chiprr`, and `Plarr` before building. Only Nestrr and Plarr need public browser origins. FlareSolverr has no published host port and is reachable only as `http://flaresolverr:8191` on the Compose network. Jackett and qBittorrent also stay private to the Compose network; their mapped ports are for local administration and may be removed behind another management route.

Nestrr is authenticated. Plarr can optionally require file-scoped playback grants: set the same random `PLARR_API_KEY` value for both services. Nestrr exchanges it server-to-server for a short-lived playback token only when a user starts playback or a download. Leave it empty only when unrestricted standalone Plarr access is intentional. Use HTTPS for both browser origins. `NESTRR_PUBLIC_ORIGIN` is also the refresh-cookie/CSRF origin, while `PLARR_PUBLIC_ORIGIN` is embedded in player links. `PLARR_ALLOWED_PARENT_ORIGINS` must contain the exact Nestrr origin.

The local Tailscale example uses:

```powershell
tailscale serve --bg --https=443 http://127.0.0.1:8080
tailscale serve --bg --https=8443 http://127.0.0.1:8090
tailscale serve --bg --https=8444 http://127.0.0.1:8085
```

Set `NESTRR_PUBLIC_ORIGIN=https://host.tailnet.ts.net` and `PLARR_PUBLIC_ORIGIN=https://host.tailnet.ts.net:8443` before recreating the containers. qBittorrent is then available to tailnet devices at `https://host.tailnet.ts.net:8444`; its WebUI port is bound only to host loopback. Keep qBittorrent's Host header and CSRF protections enabled; when restricting its server-domain list from `*`, allow both `host.tailnet.ts.net` and the Compose service name `qbittorrent`. This deployment bypasses qBittorrent's own login and relies on Tailscale as the access boundary, so keep the loopback bind, keep Funnel disabled, and do not expose port `8085` to the LAN or Internet.

## First start

1. Copy `.env.example` to `.env`.
2. Set a TMDB API Read Access Token and a random JWT secret of at least 32 characters.
3. Start FlareSolverr, Jackett, and qBittorrent. In Jackett's server settings, set **FlareSolverr API URL** to `http://flaresolverr:8191` and **FlareSolverr Max Timeout** to `60000`, then apply the settings.
4. Copy Jackett's API key and set a permanent qBittorrent password.
5. Create qBittorrent categories `Shows` and `Movies` with save paths `/downloads/completed/Shows` and `/downloads/completed/Movies`; keep incomplete data at `/downloads/incomplete`.
6. Run `docker compose up -d --build` on a CPU-only host. On every NVIDIA host, run `docker compose -f docker-compose.yml -f docker-compose.nvidia.yml up -d --build` instead.
7. Immediately open Nestrr and create the administrator.

There is deliberately no bootstrap token. The first successful account claim becomes the only administrator, so do not expose a fresh instance before completing setup.

## Environment reference

Nestrr variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `NESTRR_LISTEN_ADDR` | `:8080` | HTTP listen address |
| `NESTRR_DATABASE_PATH` | `./data/nestrr.db` | SQLite database |
| `NESTRR_IMAGE_CACHE_DIR` | `./data/images` | cached TMDB artwork |
| `NESTRR_MOVIE_LIBRARY_ROOT` | `./media/Movies` | canonical movie root |
| `NESTRR_SHOW_LIBRARY_ROOT` | `./media/Shows` | canonical show root |
| `NESTRR_TMDB_BEARER_TOKEN` | none | required TMDB read token |
| `NESTRR_TMDB_BASE_URL` | TMDB `/3` | upstream/API-test override |
| `NESTRR_TMDB_IMAGE_BASE_URL` | TMDB image origin | artwork upstream |
| `NESTRR_TMDB_CACHE_TTL` | `1h` | raw request-cache TTL |
| `NESTRR_MDBLIST_API_KEY` | none | optional primary Rotten Tomatoes rating provider |
| `NESTRR_MDBLIST_BASE_URL` | MDBList API | upstream/API-test override |
| `NESTRR_OMDB_API_KEY` | none | optional IMDb-ID fallback rating provider |
| `NESTRR_OMDB_BASE_URL` | OMDb API | upstream/API-test override |
| `NESTRR_PLARR_HOST` | none | public player origin |
| `NESTRR_PLARR_INTERNAL_HOST` | public host | private health-check and grant-issuance origin |
| `NESTRR_PLARR_API_KEY` | none | shared Plarr grant secret; normally sourced from `PLARR_API_KEY` |
| `NESTRR_JACKETT_HOST`, `NESTRR_JACKETT_API_KEY` | none | optional Jackett connection |
| `NESTRR_QBITTORRENT_HOST`, `NESTRR_QBITTORRENT_USERNAME`, `NESTRR_QBITTORRENT_PASSWORD` | none | optional qBittorrent connection |
| `NESTRR_QBITTORRENT_SHOW_CATEGORY` | `Shows` | default show category |
| `NESTRR_QBITTORRENT_MOVIE_CATEGORY` | `Movies` | default movie category |
| `NESTRR_JWT_SECRET` | none | required signing secret |
| `NESTRR_ACCESS_TOKEN_TTL` | `15m` | short-lived bearer token |
| `NESTRR_REFRESH_TOKEN_TTL` | `720h` | rotating refresh session |
| `NESTRR_PUBLIC_ORIGIN` | `http://localhost:8080` | exact catalog origin |
| `NESTRR_TRUSTED_PROXY` | `false` | trust proxy-forwarded client IP |
| `NESTRR_LOG_LEVEL` | `INFO` | structured log threshold |

Plarr variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PLARR_LISTEN_ADDR` | `:8090` | HTTP listen address |
| `PLARR_MEDIA_ROOT` | `./media` | realpath-contained media root |
| `PLARR_PUBLIC_ORIGIN` | `http://localhost:8090` | public player/API origin |
| `PLARR_ALLOWED_PARENT_ORIGINS` | empty | comma-separated postMessage parents |
| `PLARR_API_KEY` | none | optional shared secret enabling file-scoped playback grants |
| `PLARR_CACHE_DIR` | `./cache` | disposable segment/download cache |
| `PLARR_CACHE_LIMIT` | `10GB` | cold-file eviction ceiling |
| `PLARR_FFMPEG_PATH`, `PLARR_FFPROBE_PATH` | command names | media binaries |
| `PLARR_HARDWARE_ACCELERATION` | `auto` | `auto`, `nvidia`, `intel`, `amd`, `all`, or CPU-only value |
| `PLARR_CPU_LIMIT` | `4.0` | Compose CPU ceiling for software fallbacks; hardware encoding is unaffected |
| `PLARR_RENDITION_LADDER` | built-in 2160p–240p JSON | custom rendition array |
| `PLARR_LOG_LEVEL` | `INFO` | structured log threshold |

FlareSolverr variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `FLARESOLVERR_LOG_LEVEL` | `info` | FlareSolverr request and challenge log detail |

FlareSolverr is an internal helper for Jackett, not a browser-facing Nestrr dependency. Jackett waits for its healthcheck before starting. To verify both network access and challenge handling, test an indexer that requires FlareSolverr in the Jackett dashboard and inspect `docker compose logs flaresolverr`.

## GPU passthrough

CPU H.264 is always the fallback. The base Compose file does not require a GPU.

For NVIDIA under Docker Desktop/WSL, always include `docker-compose.nvidia.yml` when creating or recreating the stack. It requests `gpus: all` and `NVIDIA_DRIVER_CAPABILITIES=compute,video,utility`. Verify `nvidia-smi` inside WSL, verify that the Plarr container has a GPU device request, and check Plarr logs for an NVIDIA backend. Running the base Compose file alone removes GPU access from the recreated container.

For native Linux NVIDIA, install the NVIDIA Container Toolkit and use the same override. For Intel/AMD VAAPI, expose `/dev/dri`:

```yaml
services:
  plarr:
    devices:
      - /dev/dri:/dev/dri
    group_add:
      - video
```

Use `PLARR_HARDWARE_ACCELERATION=intel` for QSV/VAAPI or `amd` for VAAPI. If initialization fails, Plarr logs the failed backend and retries with CPU.

## qBittorrent and Chiprr filesystem contract

Strastorr watches the same completed Movies/Shows roots and extracts newly added archives in place. It preserves originals and volumes, rejects conflicting destinations, and processes one extraction at a time. It has no ports or API integration. Start the watchers before adding downloads; pre-existing archives are ignored, including after restart. `STRASTORR_POLL_INTERVAL` defaults to two seconds and `STRASTORR_LOG_LEVEL` to `INFO`. Match `PUID`/`PGID` to writable host directories. No archive-bomb limits are implemented: recursive expansion can exhaust resources. See Strastorr's README for its full safety contract.

Incomplete downloads must stay outside Chiprr's watched completed directories. qBittorrent moves finished data into the selected category path. Chiprr watches the two completed roots and creates canonical hard links:

- `Movies/Title (YYYY).ext`
- `Shows/Show/Season N/Show SxxEyy.ext`

The completed directories and organized library are mounted through one `./.runtime/data:/data` filesystem in Chiprr. Hard links cannot cross filesystems; separate Docker volumes or host filesystems break this flow. Chiprr watch mode handles new completions. Its manual execute/recovery mode scans pre-existing completed files after downtime.

Title sanitization replaces control characters with spaces, `:`, `/`, and `\` with a spaced dash, other cross-platform invalid characters with spaces, collapses whitespace, trims trailing dots/spaces, preserves Unicode, and protects Windows device names.

The current Jackett setup includes MejorTorrent through FlareSolverr. The Compose service name is deliberately used instead of a container IP so the setting remains valid when containers are recreated.

## Persistence, backup, and recovery

Back up `.runtime/nestrr` (SQLite and WAL files), `.runtime/images`, Jackett/qBittorrent configuration, and Chiprr's TMDB cache. Quiesce Nestrr or use SQLite's online backup tooling for a consistent database copy. The Plarr cache is disposable, cleared on every start, and bounded by its configured limit. Media and completed downloads remain operator-owned and need their own backup policy.

Recovery is: restore persistent configuration/database/media, start services, then run Chiprr's manual scan if completed files arrived while its watcher was offline. Nestrr revalidates filesystem availability lazily and its Downloaded page enumerates new canonical files without a global rescan.

## Initial limitations

Single video files only; no multi-episode/anime/special numbering or sidecar subtitles. Firefox is best effort. Safari/iPhone use native HLS when available. When playback grants are disabled, direct Plarr paths remain unrestricted bearer-style capabilities.

For the versioned contracts, see [docs/API.md](docs/API.md).
