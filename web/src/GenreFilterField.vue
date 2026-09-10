<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import {
  cycleGenre,
  genreState,
  type FilterOption,
  type MovieFilters,
} from "./movieFilters";

const props = defineProps<{
  filters: MovieFilters;
  genres: FilterOption[];
  translate: (key: TranslationKey) => string;
}>();
const emit = defineEmits<{ update: [filters: MovieFilters] }>();

const sortedGenres = computed(() =>
  [...props.genres].sort((a, b) => a.name.localeCompare(b.name)),
);

function genreLabel(genre: FilterOption): string {
  const state = genreState(props.filters, genre.id);
  const stateLabel =
    state === "include"
      ? props.translate("included")
      : state === "exclude"
        ? props.translate("excluded")
        : props.translate("neutral");
  return `${stateLabel}: ${genre.name}`;
}
</script>

<template>
  <fieldset class="filter-card genre-card compact-genre-card">
    <legend>
      <AppIcon name="film" /><b>{{ translate("genres") }}</b>
    </legend>
    <div class="genre-legend">
      <span><i class="include" />{{ translate("include") }}</span>
      <span><i class="exclude" />{{ translate("exclude") }}</span>
      <span><i />{{ translate("neutral") }}</span>
    </div>
    <div class="genre-grid">
      <button
        v-for="genre in sortedGenres"
        :key="genre.id"
        type="button"
        :class="['genre-toggle', genreState(filters, genre.id)]"
        :aria-label="genreLabel(genre)"
        @click="emit('update', cycleGenre(filters, genre.id))"
      >
        <AppIcon
          :name="
            genreState(filters, genre.id) === 'include'
              ? 'plus'
              : genreState(filters, genre.id) === 'exclude'
                ? 'x'
                : 'circle'
          "
          :size="15"
        />{{ genre.name }}
      </button>
    </div>
  </fieldset>
</template>
