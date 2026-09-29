<script setup lang="ts">
import AppIcon from "./AppIcon.vue";
import TorrentDownloadProgress from "./TorrentDownloadProgress.vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";

defineProps<{ detail: any }>();
const {
  artwork,
  availability,
  episodeDisplayTitle,
  episodeImagePath,
  forceTorrent,
  forcingTorrents,
  integrations,
  loadSeasonEpisodes,
  onSeasonToggle,
  openTorrent,
  percent,
  playEpisode,
  createWatchParty,
  progressFor,
  setEpisodeWatched,
  setSeasonWatched,
  t,
} = useNestrrApplicationContext();
</script>

<template>
  <section v-if="detail.seasons" class="season-section">
    <div class="section-heading">
      <h2>{{ t("seasons") }}</h2>
    </div>
    <details
      v-for="season in detail.seasons.filter(
        (item: any) => item.season_number >= 0,
      )"
      :key="season.id"
      :open="Boolean(season._open)"
      @toggle="onSeasonToggle(season, $event)"
    >
      <summary>
        <div>
          <h3>{{ season.name }}</h3>
          <p>
            {{ season.episode_count }} {{ t("episodes") }} ·
            {{ season.air_date?.slice(0, 4) }}
          </p>
        </div>
        <div class="season-summary-meta">
          <span
            v-if="season._loaded"
            :class="['state-pill', season._availabilityState]"
          >
            <AppIcon
              :name="
                season._availabilityState === 'complete'
                  ? 'checked'
                  : season._availabilityState === 'partial'
                    ? 'status'
                    : season._availabilityState === 'checking'
                      ? 'loader'
                      : 'download'
              "
              :size="15"
            />
            {{
              season._availabilityState === "complete"
                ? t("complete")
                : season._availabilityState === "partial"
                  ? `${season._availableCount}/${season._episodes?.length} ${t("available")}`
                  : season._availabilityState === "checking"
                    ? t("checking")
                    : t("unavailable")
            }}
          </span>
          <AppIcon name="chevronDown" class="season-chevron" :size="19" />
        </div>
      </summary>
      <template v-if="season._open">
        <div class="season-actions">
          <button
            class="secondary"
            :disabled="season._loading && !season._loaded"
            @click="setSeasonWatched(season)"
          >
            <AppIcon name="checks" />{{ t("markSeasonWatched") }}
          </button>
        </div>
        <div
          v-if="season._loading && !season._loaded"
          class="season-inline-state"
          aria-busy="true"
        >
          <AppIcon name="loader" />{{ t("loading") }}
        </div>
        <div
          v-else-if="season._loadError"
          class="season-inline-state error"
          role="alert"
        >
          <span>{{ season._loadError }}</span>
          <button class="secondary" @click="loadSeasonEpisodes(season)">
            {{ t("retry") }}
          </button>
        </div>
        <div v-else-if="season._loaded" class="episodes">
          <article v-for="episode in season._episodes" :key="episode.id">
            <img
              v-if="episodeImagePath(episode, detail)"
              :src="artwork(episodeImagePath(episode, detail))"
              alt=""
            />
            <div v-else class="episode-image-placeholder" aria-hidden="true">
              <AppIcon name="tv" />
            </div>
            <div class="episode-copy">
              <div class="episode-title">
                <b>{{
                  episodeDisplayTitle(
                    t("episode"),
                    episode.episode_number,
                    episode.name,
                  )
                }}</b>
                <span v-if="progressFor('episode', episode.id)?.watched"
                  ><AppIcon name="checked" :size="14" />{{ t("watched") }}</span
                >
              </div>
              <p>{{ episode.overview }}</p>
              <div class="episode-state">
                <small
                  :class="[
                    availability[`episode:${episode.id}`]?.available
                      ? 'yes'
                      : 'no',
                    {
                      downloading:
                        availability[`episode:${episode.id}`]?.torrent &&
                        !availability[`episode:${episode.id}`]?.available,
                    },
                  ]"
                >
                  <AppIcon
                    :name="
                      availability[`episode:${episode.id}`]?.available
                        ? 'checked'
                        : availability[`episode:${episode.id}`]?.torrent
                          ? 'loader'
                          : 'download'
                    "
                    :size="14"
                  />
                  {{
                    availability[`episode:${episode.id}`]?.available
                      ? t("downloaded")
                      : availability[`episode:${episode.id}`]?.torrent
                        ? t("downloading")
                        : t("unavailable")
                  }}
                </small>
                <div
                  v-if="progressFor('episode', episode.id)?.durationSeconds"
                  class="episode-progress"
                >
                  <i
                    :style="{
                      width: `${percent(progressFor('episode', episode.id))}%`,
                    }"
                  />
                  <span
                    >{{
                      Math.round(percent(progressFor("episode", episode.id)))
                    }}%</span
                  >
                </div>
              </div>
            </div>
            <div class="episode-actions">
              <button
                class="icon-action episode-watched-action"
                :class="{ active: progressFor('episode', episode.id)?.watched }"
                :aria-pressed="
                  Boolean(progressFor('episode', episode.id)?.watched)
                "
                :aria-label="
                  progressFor('episode', episode.id)?.watched
                    ? t('markUnwatched')
                    : t('markWatched')
                "
                :title="
                  progressFor('episode', episode.id)?.watched
                    ? t('markUnwatched')
                    : t('markWatched')
                "
                @click="
                  setEpisodeWatched(
                    episode,
                    !progressFor('episode', episode.id)?.watched,
                  )
                "
              >
                <AppIcon name="checks" />
              </button>
              <TorrentDownloadProgress
                v-if="
                  availability[`episode:${episode.id}`]?.torrent &&
                  !availability[`episode:${episode.id}`]?.available
                "
                :torrent="availability[`episode:${episode.id}`]?.torrent!"
                :completed-label="t('downloadFinishedPendingOrganization')"
                :force-label="t('forceStart')"
                :forcing="forcingTorrents[`episode:${episode.id}`]"
                @force-start="forceTorrent(episode, 'episode')"
              />
              <template v-else>
                <button
                  v-if="
                    integrations.plarr?.healthy &&
                    availability[`episode:${episode.id}`]?.available
                  "
                  class="primary icon-action"
                  :aria-label="
                    progressFor('episode', episode.id)?.resumeSeconds
                      ? t('resume')
                      : t('play')
                  "
                  :title="
                    progressFor('episode', episode.id)?.resumeSeconds
                      ? t('resume')
                      : t('play')
                  "
                  @click="playEpisode(episode)"
                >
                  <AppIcon name="play" />
                </button>
                <button
                  v-if="integrations.plarr?.healthy && availability[`episode:${episode.id}`]?.available"
                  class="secondary icon-action"
                  :aria-label="t('watchParty')"
                  :title="t('watchParty')"
                  @click="createWatchParty({ ...episode, show: detail.name, showId: detail.id, showOriginalName: detail.original_name, showOriginalLanguage: detail.original_language }, 'episode')"
                ><AppIcon name="users" /></button>
                <button
                  v-if="
                    integrations.jackett?.healthy &&
                    integrations.qbittorrent?.healthy &&
                    !availability[`episode:${episode.id}`]?.available
                  "
                  class="icon-action"
                  :aria-label="t('searchTorrent')"
                  :title="t('searchTorrent')"
                  @click="
                    openTorrent({ ...episode, show: detail.name }, 'episode')
                  "
                >
                  <AppIcon name="downloadFile" />
                </button>
              </template>
            </div>
          </article>
        </div>
      </template>
    </details>
  </section>
</template>
