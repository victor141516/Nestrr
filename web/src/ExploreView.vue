<script setup lang="ts">
import { shallowRef, useTemplateRef, watch } from "vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AdvancedMovieFilters from "./AdvancedMovieFilters.vue";
import AppIcon from "./AppIcon.vue";
import ContinueWatching from "./ContinueWatching.vue";
import ExploreToolbar from "./ExploreToolbar.vue";
import MediaCard from "./MediaCard.vue";

const {
  appliedFilterChips,
  appliedMovieFilters,
  applyMovieFilters,
  artwork,
  availability,
  busy,
  changeExploreSort,
  continueItems,
  doSearch,
  downloadedOnly,
  error,
  exploreSort,
  filterOptionsLoading,
  filterPanelOpen,
  hasMoreSearchResults,
  loadMoreSearchResults,
  loadingMore,
  movieCountries,
  movieFilterCount,
  movieFilterDraft,
  movieGenres,
  openMovieFilters,
  percent,
  query,
  removeMovieFilter,
  resetMovieFilters,
  resultsTransitioning,
  rottenTomatoesEnabled,
  searchMode,
  searchStarted,
  searchTotalResults,
  searchType,
  selectSearchType,
  setDownloadedOnly,
  startedFor,
  systemMembershipsFor,
  t,
  title,
  visibleItems,
} = useNestrrApplicationContext();

const resultsShell = useTemplateRef<HTMLElement>("resultsShell");
const preservedResultsHeight = shallowRef<number | null>(null);

watch(
  resultsTransitioning,
  (transitioning, _previous, onCleanup) => {
    if (transitioning) {
      const height = resultsShell.value?.getBoundingClientRect().height || 0;
      if (height > 0) preservedResultsHeight.value = height;
      return;
    }
    const release = window.setTimeout(() => {
      preservedResultsHeight.value = null;
    }, 140);
    onCleanup(() => window.clearTimeout(release));
  },
  { flush: "sync" },
);
</script>

<template>
  <ContinueWatching
    :items="continueItems"
    :artwork="artwork"
    :percent="percent"
    :translate="t"
  />

  <section
    class="content section-block search-page explore-page home-discover"
    :class="{ 'after-continue': continueItems.length }"
  >
    <div class="section-heading">
      <component :is="continueItems.length ? 'h2' : 'h1'">{{
        t("discover")
      }}</component>
    </div>
    <ExploreToolbar
      v-model:query="query"
      v-model:sort="exploreSort"
      :search-type="searchType"
      :downloaded-only="downloadedOnly"
      :busy="busy"
      :filter-panel-open="filterPanelOpen"
      :movie-filter-count="movieFilterCount"
      :translate="t"
      @submit="doSearch"
      @select-type="selectSearchType"
      @toggle-filters="
        filterPanelOpen ? (filterPanelOpen = false) : openMovieFilters()
      "
      @change-sort="changeExploreSort"
      @set-downloaded-only="setDownloadedOnly"
    />
    <Transition name="filter-panel-reveal">
      <div v-if="filterPanelOpen" class="filter-panel-reveal">
        <div class="filter-panel-reveal-clip">
          <AdvancedMovieFilters
            :filters="movieFilterDraft"
            :genres="movieGenres"
            :countries="movieCountries"
            :loading="filterOptionsLoading"
            :media-type="searchType"
            :rotten-tomatoes-enabled="rottenTomatoesEnabled"
            @apply="applyMovieFilters"
            @reset="resetMovieFilters"
          />
        </div>
      </div>
    </Transition>
    <div v-if="appliedFilterChips.length" class="active-filter-bar">
      <div>
        <button
          v-for="chip in appliedFilterChips"
          :key="chip.group"
          class="filter-chip"
          :aria-label="`${t('removeFilter')} ${chip.label}`"
          @click="removeMovieFilter(chip.group)"
        >
          {{ chip.label }} <AppIcon name="x" :size="13" />
        </button>
      </div>
      <button class="clear-filter-button" @click="resetMovieFilters()">
        {{ t("clearAll") }}
      </button>
    </div>
    <div
      ref="resultsShell"
      :class="[
        'explore-results-shell',
        { 'is-transitioning': resultsTransitioning },
      ]"
      :style="{
        minHeight:
          preservedResultsHeight === null
            ? undefined
            : `${preservedResultsHeight}px`,
      }"
    >
      <div v-if="searchStarted && !busy" class="results-summary">
        <b>{{ searchTotalResults.toLocaleString() }} {{ t("results") }}</b>
      </div>
      <p v-if="error" class="inline-message error" role="alert">
        <AppIcon name="alert" />{{ error }}
        <button
          @click="
            searchMode === 'filters' && appliedMovieFilters
              ? applyMovieFilters(appliedMovieFilters)
              : doSearch()
          "
        >
          {{ t("retry") }}
        </button>
      </p>
      <div v-if="busy && !visibleItems.length" class="checking-line">
        <span class="mini-spinner" />{{ t("searching") }}
      </div>
      <div v-else-if="visibleItems.length" class="media-grid search-results">
        <MediaCard
          v-for="m in visibleItems"
          :key="`${m.media_type}:${m.id}`"
          :m="m"
          :to="`/${m.media_type}/${m.id}`"
          :availability="availability[`${m.media_type}:${m.id}`]"
          :memberships="
            systemMembershipsFor(
              m.media_type || (m.title ? 'movie' : 'tv'),
              m.id,
            )
          "
          :started="
            startedFor(m.media_type || (m.title ? 'movie' : 'tv'), m.id)
          "
        />
      </div>
      <div v-else-if="searchStarted && !busy" class="empty-state">
        <AppIcon name="search" :size="32" />
        <h2>{{ t("noResults") }}</h2>
      </div>
      <div v-if="hasMoreSearchResults && !busy" class="load-more">
        <button :disabled="loadingMore" @click="loadMoreSearchResults">
          <AppIcon v-if="loadingMore" class="spin" name="loader" /><span>{{
            loadingMore ? t("loading") : t("loadMore")
          }}</span>
        </button>
      </div>
    </div>
  </section>
</template>
