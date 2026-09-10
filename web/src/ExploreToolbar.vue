<script setup lang="ts">
import { computed } from "vue";
import BooleanSwitch from "./BooleanSwitch.vue";
import type { ExploreSort } from "./downloadedExplore";
import ExploreFiltersControl from "./ExploreFiltersControl.vue";
import type { TranslationKey } from "./i18n";
import MediaTypeSwitch from "./MediaTypeSwitch.vue";
import type { CountryOption, FilterOption, MovieFilters } from "./movieFilters";
import SearchControl from "./SearchControl.vue";
import SortControl from "./SortControl.vue";

const props = defineProps<{
  searchType: "movie" | "tv";
  downloadedOnly: boolean;
  busy: boolean;
  filterPanelOpen: boolean;
  movieFilterCount: number;
  movieFilterDraft: MovieFilters;
  movieGenres: FilterOption[];
  movieCountries: CountryOption[];
  filterOptionsLoading: boolean;
  rottenTomatoesEnabled: boolean;
  translate: (key: TranslationKey) => string;
}>();

const query = defineModel<string>("query", { required: true });
const sort = defineModel<ExploreSort>("sort", { required: true });
const sortOptions = computed(() => [
  { value: "popularity" as const, label: props.translate("popularity") },
  { value: "release" as const, label: props.translate("releaseDate") },
  { value: "rating" as const, label: props.translate("rating") },
  { value: "title" as const, label: props.translate("titleLabel") },
]);

const emit = defineEmits<{
  submit: [];
  settle: [];
  selectType: [type: "movie" | "tv"];
  toggleFilters: [];
  changeSort: [sort: ExploreSort];
  setDownloadedOnly: [value: boolean];
  applyFilters: [filters: MovieFilters];
  resetFilters: [filters: MovieFilters];
}>();
</script>

<template>
  <div class="explore-toolbar" :aria-busy="busy">
    <SearchControl
      v-model="query"
      class="explore-search-control"
      :placeholder="translate('searchPlaceholder')"
      :action-label="translate('search')"
      @submit="emit('submit')"
      @settle="emit('settle')"
    />
    <MediaTypeSwitch
      class="explore-media-type"
      :model-value="searchType"
      :label="translate('contentType')"
      :translate="translate"
      @change="emit('selectType', $event)"
    />
    <SortControl
      v-model="sort"
      class="explore-sort-control"
      :label="translate('sortBy')"
      :options="sortOptions"
      @change="emit('changeSort', $event)"
    />
    <BooleanSwitch
      class="explore-downloaded-control"
      :model-value="downloadedOnly"
      :label="translate('downloadedOnly')"
      @change="emit('setDownloadedOnly', $event)"
    />
    <ExploreFiltersControl
      :open="filterPanelOpen"
      :active-count="movieFilterCount"
      :filters="movieFilterDraft"
      :genres="movieGenres"
      :countries="movieCountries"
      :loading="filterOptionsLoading"
      :media-type="searchType"
      :rotten-tomatoes-enabled="rottenTomatoesEnabled"
      :translate="translate"
      @toggle="emit('toggleFilters')"
      @apply="emit('applyFilters', $event)"
      @reset="emit('resetFilters', $event)"
    />
  </div>
</template>

<style scoped>
.explore-toolbar {
  display: grid;
  grid-template-columns: minmax(320px, 1fr) auto auto auto auto;
  grid-template-areas:
    "search type filters sort downloaded"
    "panel panel panel panel panel";
  align-items: start;
  gap: 0.7rem;
}
.explore-search-control {
  grid-area: search;
}
.explore-media-type {
  grid-area: type;
}
.explore-sort-control {
  grid-area: sort;
}
.explore-downloaded-control {
  grid-area: downloaded;
}
@media (max-width: 1180px) {
  .explore-toolbar {
    grid-template-columns: 1fr auto auto auto;
    grid-template-areas:
      "search search search search"
      "type filters sort downloaded"
      "panel panel panel panel";
  }
}
@media (max-width: 760px) {
  .explore-toolbar {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    grid-template-areas:
      "search search"
      "type type"
      "filters sort"
      "downloaded downloaded"
      "panel panel";
  }
}
@media (max-width: 420px) {
  .explore-toolbar {
    grid-template-columns: 1fr;
    grid-template-areas: "search" "type" "filters" "sort" "downloaded" "panel";
  }
}
</style>
