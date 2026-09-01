<script setup lang="ts">
import type { TranslationKey } from "./i18n";
import type { ExploreSort } from "./downloadedExplore";
import AppIcon from "./AppIcon.vue";

const props = defineProps<{
  searchType: "movie" | "tv";
  downloadedOnly: boolean;
  busy: boolean;
  filterPanelOpen: boolean;
  movieFilterCount: number;
  translate: (key: TranslationKey) => string;
}>();

const query = defineModel<string>("query", { required: true });
const sort = defineModel<ExploreSort>("sort", { required: true });

const emit = defineEmits<{
  submit: [];
  selectType: [type: "movie" | "tv"];
  toggleFilters: [];
  changeSort: [];
  setDownloadedOnly: [value: boolean];
}>();

function updateDownloadedOnly(event: Event): void {
  emit("setDownloadedOnly", (event.target as HTMLInputElement).checked);
}
</script>

<template>
  <form
    class="search-form explore-toolbar"
    :aria-busy="busy"
    @submit.prevent="emit('submit')"
  >
    <div class="search-input">
      <AppIcon name="search" />
      <input v-model="query" :placeholder="translate('searchPlaceholder')" />
    </div>
    <button class="primary explore-control" :disabled="!query.trim()">
      <AppIcon name="search" />
      {{ translate("search") }}
    </button>
    <div
      :class="['segmented', { 'selected-tv': searchType === 'tv' }]"
      role="group"
      :aria-label="translate('contentType')"
    >
      <i class="segmented-selection" aria-hidden="true" />
      <button
        type="button"
        :aria-pressed="searchType === 'movie'"
        @click="emit('selectType', 'movie')"
      >
        <AppIcon name="film" />{{ translate("movies") }}
      </button>
      <button
        type="button"
        :aria-pressed="searchType === 'tv'"
        @click="emit('selectType', 'tv')"
      >
        <AppIcon name="tv" />{{ translate("shows") }}
      </button>
    </div>
    <button
      type="button"
      :class="[
        'filter-trigger',
        'explore-control',
        { active: filterPanelOpen || movieFilterCount },
      ]"
      :aria-expanded="filterPanelOpen"
      @click="emit('toggleFilters')"
    >
      <AppIcon name="filter" />{{ translate("filters") }}
      <span v-if="movieFilterCount">{{ movieFilterCount }}</span>
    </button>
    <label class="explore-sort-control explore-control">
      <span>{{ translate("sortBy") }}</span>
      <select v-model="sort" @change="emit('changeSort')">
        <option value="popularity">{{ translate("popularity") }}</option>
        <option value="release">{{ translate("releaseDate") }}</option>
        <option value="rating">{{ translate("rating") }}</option>
        <option value="title">{{ translate("titleLabel") }}</option>
      </select>
    </label>
    <label class="switch-label explore-control">
      <span>{{ translate("downloadedOnly") }}</span>
      <input
        :checked="downloadedOnly"
        type="checkbox"
        @change="updateDownloadedOnly"
      />
      <i />
    </label>
  </form>
</template>
