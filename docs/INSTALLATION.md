# Install Nestrr

This guide gets a single-host Nestrr installation running with Docker Compose. The included stack starts Nestrr, [Plarr](https://github.com/victor141516/Plarr), Jackett, FlareSolverr, qBittorrent, [Chiprr](https://github.com/victor141516/chiprr), and [Strastorr](https://github.com/victor141516/Strastorr). You can use the catalog and player without configuring torrent providers.

## 1. What you need

- Docker Engine with the Compose plugin, or Docker Desktop with WSL 2 on Windows.
- Enough disk space for your media, downloads, and temporary playback data.
- A [TMDB API Read Access Token](https://www.themoviedb.org/settings/api). Use the **API Read Access Token**, not the shorter v3 API key.
- Git to clone the four projects.

Clone the repositories as siblings. The directory names matter because the Compose file builds the other services from relative paths:

```sh
mkdir nestrr-stack
cd nestrr-stack
git clone https://github.com/victor141516/Nestrr.git
git clone https://github.com/victor141516/Plarr.git
git clone https://github.com/victor141516/chiprr.git
git clone https://github.com/victor141516/Strastorr.git
cd Nestrr
```

## 2. Set your credentials

Copy the example environment file:

```sh
cp .env.example .env
```

In PowerShell, use `Copy-Item .env.example .env` instead. Edit `.env` and set at least:

| Setting | What to enter |
| --- | --- |
| `TMDB_TOKEN` | Your TMDB API Read Access Token. |
| `NESTRR_JWT_SECRET` | A fresh random secret of at least 32 characters. For example, generate one with `openssl rand -hex 32`. |

Keep `.env` private. Do not paste your TMDB token, secret, or passwords into a public issue or commit them to Git.

For playback access from other devices, also set `PLARR_API_KEY` to a random value. The Compose file passes the same key to Nestrr and Plarr, so Nestrr can create file-scoped playback grants without exposing the key to the browser.

## 3. Start the stack

Choose **one** command and reuse it for future rebuilds:

```sh
# CPU-only, or start without GPU acceleration
docker compose up -d --build

# NVIDIA, including Docker Desktop with WSL 2 on Windows
docker compose -f docker-compose.yml -f docker-compose.nvidia.yml up -d --build
```

For Intel or AMD GPU acceleration on native Linux, find the render device's group ID and use the VAAPI override:

```sh
export RENDER_GID=$(stat -c %g /dev/dri/renderD128)
docker compose -f docker-compose.yml -f docker-compose.vaapi.yml up -d --build
```

On NVIDIA systems, **always include the NVIDIA override when recreating containers**. Starting again with only the base file removes Plarr's GPU device request. GPU support is optional; Plarr can fall back to CPU when an accelerated path is unavailable.

Wait for the services to start with `docker compose ps`. Then open [http://localhost:8080](http://localhost:8080) and create your account. **The first successfully created account becomes the administrator. Claim it before exposing a fresh instance outside your machine.**

## 4. Add media

Place existing videos in:

```text
Nestrr/.runtime/data/library/Movies/
Nestrr/.runtime/data/library/Shows/
```

Nestrr and Plarr read these directories; you own the files. Chiprr can populate them from completed downloads using hard links. Keep `.runtime/data/downloads/completed` and `.runtime/data/library` on the same filesystem if you use this workflow—hard links cannot cross filesystems. Nestrr's SQLite database is stored in the Docker volume `nestrr-data`, not alongside the videos. On Windows/WSL, keep that database on Docker's Linux filesystem rather than a Windows bind mount.

## 5. Enable torrent search and downloads (optional)

The stack already starts Jackett, FlareSolverr, and qBittorrent, but they need first-time setup:

1. Open Jackett at [http://localhost:19117](http://localhost:19117). Add the indexers you want to use. In its server settings, set **FlareSolverr API URL** to `http://flaresolverr:8191` and **FlareSolverr Max Timeout** to `60000`.
2. Copy Jackett's API key into `JACKETT_API_KEY` in Nestrr's `.env`.
3. Open qBittorrent at [http://localhost:8085](http://localhost:8085). Its first-run temporary password is in `docker compose logs qbittorrent`. Set a permanent password and enter the username and password as `QBITTORRENT_USERNAME` and `QBITTORRENT_PASSWORD` in `.env`.
4. Create qBittorrent categories named `Movies` and `Shows`, with save paths `/downloads/completed/Movies` and `/downloads/completed/Shows`. Keep incomplete downloads in `/downloads/incomplete`.
5. Recreate the stack with the same Compose command chosen in step 3 so Nestrr receives the new credentials.

When a torrent finishes in the matching completed directory, Strastorr can extract newly added archives and Chiprr can organize recognized videos into the library. Start these watchers before downloading: Strastorr does not automatically process archives that were already present when it started. Do not feed untrusted archives to Strastorr; it does not limit recursive extraction or protect against archive bombs.

## 6. Watch from another device (optional)

Both the Nestrr page and the Plarr player need to be reachable from the viewer's device. Set `NESTRR_PUBLIC_ORIGIN` and `PLARR_PUBLIC_ORIGIN` in `.env` to their exact browser-facing origins, then recreate the stack. For example, a Tailscale Serve setup can expose Nestrr on HTTPS port 443 and Plarr on 8443. Use HTTPS for remote access; do not expose a new instance until you have claimed the administrator account.

See the [deployment guide](../DEPLOYMENT.md#topology-and-origins) for an example and reverse-proxy details. If the page loads but playback does not, check that **both** origins are reachable from that device.

## Check or troubleshoot

```sh
docker compose ps
docker compose logs --tail=100 nestrr plarr
docker compose logs --tail=100 jackett qbittorrent chiprr strastorr
```

For configuration, GPU passthrough, storage, backup, and recovery details, use the [deployment reference](../DEPLOYMENT.md). Return to the [Nestrr overview](../README.md).
