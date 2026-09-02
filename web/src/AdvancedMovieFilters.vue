<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import AppIcon from "./AppIcon.vue";
import CountryMultiSelect from "./CountryMultiSelect.vue";
import CompanyTagInput from "./CompanyTagInput.vue";
import DualRangeSlider from "./DualRangeSlider.vue";
import KeywordTagInput from "./KeywordTagInput.vue";
import PersonTagInput from "./PersonTagInput.vue";
import { session } from "./api";
import { tr } from "./i18n";
import {
  cloneMovieFilters,
  createMovieFilters,
  cycleGenre,
  genreState,
  normalizeMovieFilters,
  productionCountryOptions,
  type CountryOption,
  type FilterOption,
  type MovieFilters,
} from "./movieFilters";

const props = defineProps<{
  filters: MovieFilters;
  genres: FilterOption[];
  countries: CountryOption[];
  loading: boolean;
  mediaType: "movie" | "tv";
  rottenTomatoesEnabled: boolean;
}>();
const emit = defineEmits<{
  apply: [filters: MovieFilters];
  reset: [filters: MovieFilters];
}>();
const t = (key: any) => tr(session.user?.language, key);
const currentYear = new Date().getFullYear();
const draft = reactive<MovieFilters>(cloneMovieFilters(props.filters));
watch(
  () => props.filters,
  (value) => Object.assign(draft, cloneMovieFilters(value)),
  { deep: true },
);
const sortedGenres = computed(() =>
  [...props.genres].sort((a, b) => a.name.localeCompare(b.name)),
);
const countryOptions = computed(() =>
  productionCountryOptions(props.countries),
);
function toggleGenre(id: number) {
  Object.assign(draft, cycleGenre(draft, id));
}
function setIncludedKeywords(value: MovieFilters["includeKeywords"]) {
  draft.includeKeywords = value;
  const selected = new Set(value.map((keyword) => keyword.id));
  draft.excludeKeywords = draft.excludeKeywords.filter(
    (keyword) => !selected.has(keyword.id),
  );
}
function setExcludedKeywords(value: MovieFilters["excludeKeywords"]) {
  draft.excludeKeywords = value;
  const selected = new Set(value.map((keyword) => keyword.id));
  draft.includeKeywords = draft.includeKeywords.filter(
    (keyword) => !selected.has(keyword.id),
  );
}
function setCast(value: MovieFilters["cast"]) {
  draft.cast = value;
}
function setDirectors(value: MovieFilters["directors"]) {
  draft.directors = value;
}
function setCompanies(value: MovieFilters["includeCompanies"]) {
  draft.includeCompanies = value;
}
function genreLabel(genre: FilterOption) {
  const state = genreState(draft, genre.id);
  return state === "include"
    ? `${t("included")}: ${genre.name}`
    : state === "exclude"
      ? `${t("excluded")}: ${genre.name}`
      : `${t("neutral")}: ${genre.name}`;
}
function reset() {
  const fresh = createMovieFilters(session.user?.language);
  Object.assign(draft, fresh);
  emit("reset", fresh);
}
function apply() {
  emit("apply", normalizeMovieFilters(draft));
}
</script>

<template>
  <aside class="movie-filter-panel" :aria-label="t('filters')">
    <div v-if="loading" class="filter-loading" aria-busy="true">
      <AppIcon class="spin" name="loader" /><span>{{
        t("loadingFilters")
      }}</span>
    </div>
    <form v-else @submit.prevent="apply">
      <div
        :class="[
          'filter-grid',
          'compact-filter-grid',
          { 'has-rotten-tomatoes': rottenTomatoesEnabled },
        ]"
      >
        <fieldset class="filter-card range-card">
          <legend>
            <AppIcon name="timer" /><b>{{ t("releaseYear") }}</b>
          </legend>
          <DualRangeSlider
            :min-value="draft.yearMin"
            :max-value="draft.yearMax"
            :floor="1870"
            :ceiling="currentYear + 5"
            :min-label="`${t('releaseYear')} · ${t('minimum')}`"
            :max-label="`${t('releaseYear')} · ${t('maximum')}`"
            @update:min-value="draft.yearMin = $event"
            @update:max-value="draft.yearMax = $event"
          />
        </fieldset>
        <fieldset class="filter-card range-card">
          <legend><AppIcon name="star" /><b>TMDB</b></legend>
          <DualRangeSlider
            :min-value="draft.ratingMin"
            :max-value="draft.ratingMax"
            :floor="0"
            :ceiling="10"
            :step="0.5"
            :min-label="`${t('rating')} · ${t('minimum')}`"
            :max-label="`${t('rating')} · ${t('maximum')}`"
            @update:min-value="draft.ratingMin = $event"
            @update:max-value="draft.ratingMax = $event"
          />
        </fieldset>
        <fieldset v-if="rottenTomatoesEnabled" class="filter-card range-card">
          <legend>
            <AppIcon name="tomato" /><b>{{ t("rottenTomatoes") }}</b>
          </legend>
          <DualRangeSlider
            :min-value="draft.rottenTomatoesMin"
            :max-value="draft.rottenTomatoesMax"
            :floor="0"
            :ceiling="100"
            :min-label="`${t('rottenTomatoes')} · ${t('minimum')}`"
            :max-label="`${t('rottenTomatoes')} · ${t('maximum')}`"
            :format-value="(value) => `${value}%`"
            @update:min-value="draft.rottenTomatoesMin = $event"
            @update:max-value="draft.rottenTomatoesMax = $event"
          />
        </fieldset>
        <fieldset v-if="mediaType === 'movie'" class="filter-card range-card">
          <legend>
            <AppIcon name="clock" /><b>{{ t("runtime") }}</b>
          </legend>
          <DualRangeSlider
            :min-value="draft.runtimeMin"
            :max-value="draft.runtimeMax"
            :floor="0"
            :ceiling="600"
            :step="5"
            :min-label="`${t('runtime')} · ${t('minimum')}`"
            :max-label="`${t('runtime')} · ${t('maximum')}`"
            :format-value="(value) => `${value} min`"
            @update:min-value="draft.runtimeMin = $event"
            @update:max-value="draft.runtimeMax = $event"
          />
        </fieldset>
        <fieldset class="filter-card range-card">
          <legend>
            <AppIcon name="shield" /><b>{{ t("ageRating") }}</b>
          </legend>
          <DualRangeSlider
            :min-value="draft.ageMin"
            :max-value="draft.ageMax"
            :floor="0"
            :ceiling="18"
            :min-label="`${t('ageRating')} · ${t('minimum')}`"
            :max-label="`${t('ageRating')} · ${t('maximum')}`"
            :format-value="
              (value) =>
                `${value} ${session.user?.language?.startsWith('es') ? 'años' : 'years'}`
            "
            @update:min-value="draft.ageMin = $event"
            @update:max-value="draft.ageMax = $event"
          />
        </fieldset>
        <fieldset class="filter-card country-card compact-country-card">
          <legend>
            <AppIcon name="globe" /><b>{{ t("productionCountry") }}</b>
          </legend>
          <CountryMultiSelect
            v-model="draft.productionCountries"
            :options="countryOptions"
            :countries="countries"
          />
        </fieldset>
        <fieldset class="filter-card genre-card compact-genre-card">
          <legend>
            <AppIcon name="film" /><b>{{ t("genres") }}</b>
          </legend>
          <div class="genre-legend">
            <span><i class="include" />{{ t("include") }}</span
            ><span><i class="exclude" />{{ t("exclude") }}</span
            ><span><i />{{ t("neutral") }}</span>
          </div>
          <div class="genre-grid">
            <button
              v-for="genre in sortedGenres"
              :key="genre.id"
              type="button"
              :class="['genre-toggle', genreState(draft, genre.id)]"
              :aria-label="genreLabel(genre)"
              @click="toggleGenre(genre.id)"
            >
              <AppIcon
                :name="
                  genreState(draft, genre.id) === 'include'
                    ? 'plus'
                    : genreState(draft, genre.id) === 'exclude'
                      ? 'x'
                      : 'circle'
                "
                :size="15"
              />{{ genre.name }}
            </button>
          </div>
        </fieldset>
        <fieldset class="filter-card keyword-card compact-keyword-card">
          <legend>
            <AppIcon name="sparkles" /><b>{{ t("keywords") }}</b>
          </legend>
          <div class="keyword-filter-fields">
            <KeywordTagInput
              id="include-keywords"
              :model-value="draft.includeKeywords"
              :label="t('includeKeywords')"
              :placeholder="t('searchKeywords')"
              @update:model-value="setIncludedKeywords"
            />
            <KeywordTagInput
              id="exclude-keywords"
              :model-value="draft.excludeKeywords"
              :label="t('excludeKeywords')"
              :placeholder="t('searchKeywords')"
              @update:model-value="setExcludedKeywords"
            />
          </div>
        </fieldset>
        <fieldset
          v-if="mediaType === 'movie'"
          class="filter-card keyword-card compact-keyword-card people-filter-card"
        >
          <legend>
            <AppIcon name="users" /><b>{{ t("people") }}</b>
          </legend>
          <div class="keyword-filter-fields">
            <PersonTagInput
              id="cast"
              :model-value="draft.cast"
              :label="t('cast')"
              :placeholder="t('searchPeople')"
              @update:model-value="setCast"
            />
            <PersonTagInput
              id="directors"
              :model-value="draft.directors"
              :label="t('directors')"
              :placeholder="t('searchPeople')"
              @update:model-value="setDirectors"
            />
          </div>
        </fieldset>
        <fieldset
          v-if="mediaType === 'movie'"
          class="filter-card keyword-card compact-keyword-card company-filter-card"
        >
          <legend>
            <AppIcon name="building" /><b>{{ t("productionCompanies") }}</b>
          </legend>
          <div class="keyword-filter-fields">
            <CompanyTagInput
              id="companies"
              :model-value="draft.includeCompanies"
              :label="t('productionCompany')"
              :placeholder="t('searchCompanies')"
              @update:model-value="setCompanies"
            />
          </div>
        </fieldset>
      </div>
      <footer class="filter-actions">
        <div>
          <button type="button" @click="reset">
            <AppIcon name="reset" />{{ t("resetFilters") }}
          </button>
          <button class="primary" type="submit">
            <AppIcon name="search" />{{ t("showResults") }}
          </button>
        </div>
      </footer>
    </form>
  </aside>
</template>
