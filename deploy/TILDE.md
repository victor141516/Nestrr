# Tilde deployment

`tilde.compose.yml` runs Nestrr and Plarr beside the existing Jellyfin,
Chiprr, Jackett and qBittorrent services. Copy it to
`~/services/nestrr/docker-compose.yml`, with sibling `Nestrr` and `Plarr`
source checkouts. Set `RENDER_GID` in that directory's `compose.env` from
`stat -c %g /dev/dri/renderD128`. Keep secrets outside the repositories.

```sh
docker compose --env-file compose.env up -d --build
```

The existing Chiprr env file supplies the TMDB token. The Nestrr env file
supplies a random JWT secret and the existing Jackett API key. Initialize
the administrator through Nestrr's bootstrap API before publishing it.
The deployed administrator is `admin`; its generated password is stored
only in `~/secrets/nestrr/admin-credentials` (mode 0600).

Both applications read the same organized Movies and Shows directories as
Jellyfin. qBittorrent retains its original files for seeding. The movie
migration created hard links, preserved additional versions under
`Versions`, and preserved the Blu-ray directory tree. Do not replace source
downloads or delete directories by matching title alone.

Migration backups and verification reports are under
`~/maintenance-backups/nestrr-migration-20260913`. They include the original
Compose and Caddy files, an online Jellyfin database backup, source-to-link
mapping, TMDB detail snapshots, and Jellyfin user progress. Intermediate
migration directories are not library roots.

## Playback authentication

Insert `tilde.Caddyfile` inside the existing `(services)` snippet, validate
the complete configuration, then reload Caddy. No additional host ports or
DNS changes are required by this Compose file.

Set the same random `PLARR_API_KEY` for Nestrr and Plarr. Nestrr uses the
Docker-network Plarr origin to exchange this secret for an opaque grant bound
to one resolved file. Caddy proxies Plarr directly; there is no `forward_auth`
or shared browser cookie. The player propagates the grant to browser and Cast
manifest, segment, subtitle, preview, download, and session requests. Leave
the key empty only when unrestricted standalone Plarr access is intended.

## AMD and verification

Tilde uses VAAPI for decoding, scaling and H.264/HEVC encoding. HDR output
preserves ten-bit HEVC and PQ metadata. HDR-to-SDR uses CPU tone mapping
between hardware decode/resize and hardware encode. Vulkan interop was
excluded after black-frame failures and an `amdgpu_hmm_invalidate_gfx`
kernel panic during userptr memory compaction on Tilde's kernel.

The Plarr verification script creates eight-second flash/tone fixtures and
checks direct-play byte identity, stream copying, audio-only transcoding,
audio/video transcoding, HDR preservation, HDR-to-SDR, and non-keyframe
seeking. It measures synchronization and drift from the decoded output,
checks nonblank images and HDR metadata, and requires hardware execution
in the backend diagnostics. Run against an isolated fixture directory.
See `Plarr/scripts/verify_playback.py --help` for host and mount arguments.
