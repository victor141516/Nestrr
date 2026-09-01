<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import MediaKeywordsMenu from "./MediaKeywordsMenu.vue";
import MediaMetadataGroup from "./MediaMetadataGroup.vue";
import TorrentDownloadProgress from "./TorrentDownloadProgress.vue";
import {
  companyMetadataItems,
  countryMetadataItems,
  genreExploreRoute,
  genreMetadataItems,
  keywordMetadataItems,
  type TMDBCountry,
  type TMDBNamedItem,
} from "./mediaMetadata";
import { useNestrrApplicationContext } from "./useNestrrApplication";

type MediaDetail = {
  id: number;
  backdrop_path?: string;
  poster_path?: string;
  overview?: string;
  runtime?: number;
  vote_average?: number;
  rotten_tomatoes_score?: number;
  genres?: TMDBNamedItem[];
  production_companies?: TMDBNamedItem[];
  production_countries?: TMDBCountry[];
  _availabilityState?: string;
  _keywords?: TMDBNamedItem[];
  [key: string]: unknown;
};

const props = defineProps<{ detail: MediaDetail }>();
const {
  artwork,
  availability,
  currentAvailability,
  currentKind,
  detailProgress,
  fmtSeconds,
  forceTorrent,
  forcingTorrents,
  inSystemList,
  integrations,
  listEditor,
  mediaLists,
  membershipsFor,
  openTorrent,
  playDetail,
  refreshDetail,
  rottenTomatoesEnabled,
  renditionURL,
  t,
  title,
  toggleCustomList,
  toggleSystemList,
  year,
} = useNestrrApplicationContext();

const genres = computed(() =>
  genreMetadataItems(currentKind.value, props.detail.genres),
);
const genreGroupRoute = computed(() =>
  genreExploreRoute(
    currentKind.value,
    genres.value.map((genre) => Number(genre.id)),
  ),
);
const companies = computed(() =>
  companyMetadataItems(currentKind.value, props.detail.production_companies),
);
const countries = computed(() =>
  countryMetadataItems(currentKind.value, props.detail.production_countries),
);
const keywords = computed(() =>
  keywordMetadataItems(currentKind.value, props.detail._keywords),
);
const rottenTomatoesScore = computed(() => {
  const score = Number(props.detail.rotten_tomatoes_score);
  return Number.isFinite(score) && score >= 0 && score <= 100
    ? Math.round(score)
    : null;
});
</script>

<template>
  <div
    class="detail-hero"
    :style="{
      backgroundImage: `linear-gradient(90deg,#060a12 2%,rgba(6,10,18,.86) 52%,rgba(6,10,18,.2)),linear-gradient(0deg,#060a12 0%,transparent 35%),url(${artwork(detail.backdrop_path, 'original')})`,
    }"
  >
    <img class="detail-poster" :src="artwork(detail.poster_path)" alt="" />
    <div class="detail-copy">
      <p class="eyebrow">
        {{
          currentKind === "movie"
            ? t("movie")
            : currentKind === "tv"
              ? t("show")
              : t("collection")
        }}
        · {{ year(detail) }}
      </p>
      <h1>{{ title(detail) }}</h1>
      <div class="facts">
        <span
          v-if="currentKind !== 'collection'"
          :class="[
            'availability-fact',
            currentAvailability?.torrent && !currentAvailability.available
              ? 'downloading'
              : currentAvailability?.available ||
                  (currentKind === 'tv' &&
                    detail._availabilityState &&
                    detail._availabilityState !== 'none')
                ? 'available'
                : 'remote',
          ]"
        >
          <AppIcon
            :name="
              currentAvailability?.torrent && !currentAvailability.available
                ? 'loader'
                : currentAvailability?.available ||
                    (currentKind === 'tv' &&
                      detail._availabilityState &&
                      detail._availabilityState !== 'none')
                  ? 'checked'
                  : 'download'
            "
            :size="15"
          />
          {{
            currentAvailability?.torrent && !currentAvailability.available
              ? t("downloading")
              : currentKind === "tv" && detail._availabilityState === "partial"
                ? t("partial")
                : currentAvailability?.available
                  ? t("downloaded")
                  : t("unavailable")
          }}
        </span>
        <span v-if="detail.vote_average"
          ><AppIcon name="star" :size="15" />{{
            detail.vote_average.toFixed(1)
          }}</span
        >
        <span
          v-if="rottenTomatoesEnabled && rottenTomatoesScore !== null"
          :title="t('rottenTomatoes')"
          ><AppIcon name="tomato" :size="15" />{{ rottenTomatoesScore }}%</span
        >
        <span v-if="detail.runtime"
          ><AppIcon name="clock" :size="15" />{{ detail.runtime }} min</span
        >
        <span v-if="detailProgress?.positionSeconds"
          ><AppIcon name="clock" :size="15" />{{
            fmtSeconds(detailProgress.positionSeconds)
          }}</span
        >
      </div>
      <div
        v-if="
          genres.length ||
          companies.length ||
          countries.length ||
          keywords.length
        "
        class="detail-metadata"
      >
        <MediaMetadataGroup
          v-if="genres.length"
          icon="tags"
          :label="t('genres')"
          :icon-label="t('exploreAllGenres')"
          :icon-to="genreGroupRoute"
          :items="genres"
        />
        <MediaMetadataGroup
          v-if="companies.length"
          icon="building"
          :label="t('productionCompanies')"
          :items="companies"
        />
        <MediaMetadataGroup
          v-if="countries.length"
          icon="globe"
          :label="t('productionCountry')"
          :items="countries"
        />
        <MediaKeywordsMenu
          v-if="keywords.length"
          :items="keywords"
          :label="t('keywords')"
          :search-label="t('searchKeywords')"
          :no-results-label="t('noResults')"
        />
      </div>
      <div class="overview-wrap">
        <p class="overview">{{ detail.overview }}</p>
        <button
          class="metadata-refresh"
          :aria-label="t('refresh')"
          :title="t('refresh')"
          @click="refreshDetail"
        >
          <AppIcon name="refresh" />
        </button>
      </div>
      <div
        v-if="currentKind === 'movie' || currentKind === 'tv'"
        class="title-list-actions"
        :aria-label="t('listActions')"
      >
        <button
          v-for="entry in [
            { key: 'favorites', icon: 'heart', label: t('favorites') },
            { key: 'watch-later', icon: 'bookmark', label: t('watchLater') },
            { key: 'watched', icon: 'checks', label: t('watched') },
          ]"
          :key="entry.key"
          :class="{ active: inSystemList(entry.key, currentKind, detail.id) }"
          :aria-pressed="inSystemList(entry.key, currentKind, detail.id)"
          :title="entry.label"
          @click="toggleSystemList(entry.key, detail, currentKind)"
        >
          <AppIcon :name="entry.icon" /><span>{{ entry.label }}</span>
        </button>
        <details class="custom-list-menu">
          <summary :title="t('customLists')">
            <AppIcon name="list" /><span>{{ t("customLists") }}</span>
          </summary>
          <div>
            <button
              v-for="list in mediaLists.filter((item) => !item.systemKey)"
              :key="list.id"
              :class="{
                selected: membershipsFor(currentKind, detail.id).some(
                  (membership) => membership.listId === list.id,
                ),
              }"
              @click="toggleCustomList(list, detail, currentKind)"
            >
              <AppIcon
                :name="
                  membershipsFor(currentKind, detail.id).some(
                    (membership) => membership.listId === list.id,
                  )
                    ? 'check'
                    : 'plus'
                "
              />
              {{ list.name }}
            </button>
            <button
              class="create-inline-list"
              @click="listEditor = { mode: 'create', name: '' }"
            >
              <AppIcon name="plus" />{{ t("newList") }}
            </button>
          </div>
        </details>
      </div>
      <div class="actions detail-actions">
        <TorrentDownloadProgress
          v-if="currentAvailability?.torrent && !currentAvailability.available"
          :torrent="currentAvailability.torrent"
          :completed-label="t('downloadFinishedPendingOrganization')"
          :force-label="t('forceStart')"
          :forcing="forcingTorrents[`${currentKind}:${detail.id}`]"
          @force-start="forceTorrent(detail, currentKind)"
        />
        <template v-else>
          <button
            v-if="integrations.plarr?.healthy && currentAvailability?.playerUrl"
            class="primary icon-action"
            :aria-label="
              detailProgress?.resumeSeconds ? t('resume') : t('play')
            "
            :title="detailProgress?.resumeSeconds ? t('resume') : t('play')"
            @click="playDetail"
          >
            <AppIcon name="play" />
          </button>
          <details
            v-if="
              integrations.plarr?.healthy &&
              currentAvailability?.originalDownloadUrl
            "
            class="download-menu icon-action"
          >
            <summary :aria-label="t('download')" :title="t('download')">
              <AppIcon name="downloadFile" />
            </summary>
            <a :href="currentAvailability.originalDownloadUrl">Original</a>
            <a
              v-for="resolution in ['1080p', '720p', '480p', '360p', '240p']"
              :key="resolution"
              :href="renditionURL(currentAvailability, resolution)"
            >
              {{ resolution }} MP4
            </a>
          </details>
          <button
            v-if="
              currentKind !== 'collection' &&
              integrations.jackett?.healthy &&
              integrations.qbittorrent?.healthy &&
              !currentAvailability?.available
            "
            class="torrent-search-action"
            :aria-label="t('searchTorrent')"
            :title="t('searchTorrent')"
            @click="openTorrent(detail, currentKind)"
          >
            <AppIcon name="search" /><span>{{ t("searchTorrent") }}</span>
          </button>
        </template>
      </div>
      <p
        v-if="
          !integrations.plarr?.healthy &&
          availability[`${currentKind}:${detail.id}`]?.available
        "
        class="inline-message warning"
      >
        <AppIcon name="alert" />{{ t("plarrUnavailable") }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.detail-metadata {
  display: flex;
  max-width: 800px;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin: -0.35rem 0 1.2rem;
}

@media (max-width: 760px) {
  .detail-metadata {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
