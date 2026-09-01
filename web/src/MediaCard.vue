<script setup lang="ts">
import { computed } from "vue";
import { artwork, session } from "./api";
import { tr } from "./i18n";
import AppIcon from "./AppIcon.vue";

const props = withDefaults(
  defineProps<{
    m: any;
    availability?: any;
    to: string;
    memberships?: string[];
    started?: boolean;
  }>(),
  { memberships: () => [], started: false },
);

const indicators = computed(() => [
  ...(props.memberships.includes("favorites")
    ? [
        {
          key: "favorites",
          icon: "heart",
          label: tr(session.user?.language, "favorites"),
        },
      ]
    : []),
  ...(props.memberships.includes("watch-later")
    ? [
        {
          key: "watch-later",
          icon: "bookmark",
          label: tr(session.user?.language, "watchLater"),
        },
      ]
    : []),
  ...(props.memberships.includes("watched")
    ? [
        {
          key: "watched",
          icon: "checks",
          label: tr(session.user?.language, "watched"),
        },
      ]
    : []),
  ...(props.started
    ? [
        {
          key: "started",
          icon: "started",
          label: tr(session.user?.language, "started"),
        },
      ]
    : []),
]);

const state = computed(() =>
  props.availability?.state === "partial"
    ? "partial"
    : props.availability?.available
      ? "available"
      : "remote",
);
const stateLabel = computed(() =>
  state.value === "available"
    ? tr(session.user?.language, "downloaded")
    : state.value === "partial"
      ? tr(session.user?.language, "partial")
      : tr(session.user?.language, "unavailable"),
);
const tmdbScore = computed(() => Number(props.m.vote_average || 0).toFixed(1));
const rottenTomatoesScore = computed(() => {
  const rawScore = props.m.rotten_tomatoes_score;
  if (rawScore === null || rawScore === undefined || rawScore === "")
    return null;
  const score = Number(rawScore);
  return Number.isFinite(score) && score >= 0 && score <= 100
    ? Math.round(score)
    : null;
});
const rottenTomatoesLabel = computed(() =>
  tr(session.user?.language, "rottenTomatoes"),
);
</script>

<template>
  <RouterLink class="media-card" :to="to">
    <div class="poster">
      <img
        v-if="m.poster_path"
        :src="artwork(m.poster_path)"
        loading="lazy"
        :alt="m.title || m.name || ''"
      />
      <div v-else class="poster-placeholder">
        <AppIcon name="film" :size="28" />
      </div>
      <span
        :class="['card-status-indicator', state]"
        :aria-label="stateLabel"
        :title="stateLabel"
        ><AppIcon
          :name="
            state === 'available'
              ? 'checked'
              : state === 'partial'
                ? 'status'
                : 'download'
          "
          :size="15"
      /></span>
      <span v-if="indicators.length" class="card-list-indicators">
        <span
          v-for="indicator in indicators"
          :key="indicator.key"
          :class="['card-list-indicator', indicator.key]"
          :aria-label="indicator.label"
          :title="indicator.label"
          ><AppIcon :name="indicator.icon" :size="14"
        /></span>
      </span>
      <span class="card-play"><AppIcon name="play" :size="18" /></span>
    </div>
    <b>{{ m.title || m.name }}</b>
    <small>
      <span>{{ (m.release_date || m.first_air_date || "").slice(0, 4) }}</span>
      <span class="card-ratings">
        <i
          class="card-rating card-rating-tmdb"
          :aria-label="`TMDB: ${tmdbScore}`"
          :title="`TMDB: ${tmdbScore}`"
          ><AppIcon name="star" :size="13" />{{ tmdbScore }}</i
        >
        <i
          v-if="rottenTomatoesScore !== null"
          class="card-rating card-rating-rotten"
          :aria-label="`${rottenTomatoesLabel}: ${rottenTomatoesScore}%`"
          :title="rottenTomatoesLabel"
          ><AppIcon name="tomato" :size="13" />{{ rottenTomatoesScore }}%</i
        >
      </span>
    </small>
  </RouterLink>
</template>
