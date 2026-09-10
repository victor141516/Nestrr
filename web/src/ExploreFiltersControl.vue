<script setup lang="ts">
import AdvancedMovieFilters from "./AdvancedMovieFilters.vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import type { CountryOption, FilterOption, MovieFilters } from "./movieFilters";

defineProps<{
  open: boolean;
  activeCount: number;
  filters: MovieFilters;
  genres: FilterOption[];
  countries: CountryOption[];
  loading: boolean;
  mediaType: "movie" | "tv";
  rottenTomatoesEnabled: boolean;
  translate: (key: TranslationKey) => string;
}>();

const emit = defineEmits<{
  toggle: [];
  apply: [filters: MovieFilters];
  reset: [filters: MovieFilters];
}>();
</script>

<template>
  <button
    type="button"
    :class="['explore-filter-trigger', { active: open || activeCount }]"
    :aria-expanded="open"
    @click="emit('toggle')"
  >
    <AppIcon name="filter" />{{ translate("filters") }}
    <span v-if="activeCount">{{ activeCount }}</span>
  </button>
  <Transition name="filter-panel-reveal">
    <div v-if="open" class="explore-filter-panel-reveal">
      <div class="filter-panel-reveal-clip">
        <AdvancedMovieFilters
          :filters="filters"
          :genres="genres"
          :countries="countries"
          :loading="loading"
          :media-type="mediaType"
          :rotten-tomatoes-enabled="rottenTomatoesEnabled"
          @apply="emit('apply', $event)"
          @reset="emit('reset', $event)"
        />
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.explore-filter-trigger {
  grid-area: filters;
  display: inline-flex;
  height: 44px;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border-color: var(--line);
  background: var(--control);
  font-size: 0.84rem;
}
.explore-filter-trigger.active {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent);
}
.explore-filter-trigger > span {
  display: grid;
  min-width: 20px;
  min-height: 20px;
  padding: 0 0.35rem;
  place-items: center;
  border-radius: 999px;
  background: var(--accent);
  color: var(--accent-ink);
  font-size: 0.72rem;
  font-weight: 700;
}
.explore-filter-panel-reveal {
  grid-area: panel;
  min-width: 0;
  overflow: hidden;
}
.filter-panel-reveal-enter-active,
.filter-panel-reveal-leave-active {
  display: grid;
  grid-template-rows: 1fr;
  transition:
    grid-template-rows 0.22s ease,
    opacity 0.16s ease;
}
.filter-panel-reveal-enter-from,
.filter-panel-reveal-leave-to {
  grid-template-rows: 0fr;
  opacity: 0;
}
.filter-panel-reveal-clip {
  min-height: 0;
  overflow: hidden;
}
@media (prefers-reduced-motion: reduce) {
  .filter-panel-reveal-enter-active,
  .filter-panel-reveal-leave-active {
    transition-duration: 0s;
  }
}
</style>
