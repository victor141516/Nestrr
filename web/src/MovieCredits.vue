<script setup lang="ts">
import AppIcon from "./AppIcon.vue";
import type { RouteLocationRaw } from "vue-router";
import type { MovieCastMember, MovieCreditPerson } from "./mediaCredits";

defineProps<{
  directors: MovieCreditPerson[];
  cast: MovieCastMember[];
  directorLabel: string;
  castLabel: string;
  artwork: (path?: string | null, size?: string) => string;
  directorTo: (person: MovieCreditPerson) => RouteLocationRaw;
  castTo: (person: MovieCastMember) => RouteLocationRaw;
}>();
</script>

<template>
  <section v-if="directors.length || cast.length" class="movie-credits">
    <div
      v-if="directors.length"
      class="movie-directors"
      :aria-label="directorLabel"
    >
      <span class="movie-credit-heading">
        <AppIcon name="film" :size="15" />
        {{ directorLabel }}
      </span>
      <span class="movie-director-list">
        <RouterLink
          v-for="director in directors"
          :key="director.id"
          :data-person-id="director.id"
          :to="directorTo(director)"
        >
          {{ director.name }}
        </RouterLink>
      </span>
    </div>

    <div v-if="cast.length" class="movie-cast" :aria-label="castLabel">
      <h2>
        <AppIcon name="users" :size="15" />
        {{ castLabel }}
      </h2>
      <div class="movie-cast-list">
        <RouterLink
          v-for="member in cast"
          :key="member.id"
          class="movie-cast-member"
          :data-person-id="member.id"
          :to="castTo(member)"
        >
          <div
            class="movie-cast-portrait"
            :class="{
              'movie-cast-portrait--missing': !member.profilePath,
            }"
          >
            <img
              v-if="member.profilePath"
              :src="artwork(member.profilePath, 'w185')"
              alt=""
              loading="lazy"
            />
            <AppIcon v-else name="user" :size="24" :stroke-width="1.5" />
          </div>
          <strong>{{ member.name }}</strong>
          <small v-if="member.character">{{ member.character }}</small>
        </RouterLink>
      </div>
    </div>
  </section>
</template>

<style scoped>
.movie-credits {
  display: grid;
  max-width: 800px;
  gap: 0.85rem;
  margin: 1.2rem 0;
}

.movie-directors {
  display: inline-flex;
  width: fit-content;
  max-width: 100%;
  overflow: hidden;
  align-items: stretch;
  border-radius: 6px;
  background: rgba(15, 23, 34, 0.78);
  color: #c4cbd4;
  font-size: 0.8rem;
}

.movie-credit-heading {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.38rem;
  padding: 0.25rem 0.52rem;
  border-inline-end: 1px solid rgba(255, 255, 255, 0.12);
  color: var(--accent);
}

.movie-director-list {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  overflow-x: auto;
  scrollbar-width: none;
}

.movie-director-list::-webkit-scrollbar {
  display: none;
}

.movie-director-list > a {
  display: inline-flex;
  flex: 0 0 auto;
  min-height: 0;
  align-items: center;
  padding: 0.25rem 0.52rem;
  white-space: nowrap;
}

.movie-director-list > a + a {
  border-inline-start: 1px solid rgba(255, 255, 255, 0.12);
}

.movie-cast h2 {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  margin: 0 0 0.5rem;
  color: rgba(244, 247, 251, 0.82);
  font-size: 0.8rem;
  font-weight: 650;
  letter-spacing: 0.01em;
}

.movie-cast h2 svg {
  color: var(--accent);
}

.movie-cast-list {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 0.65rem;
}

.movie-cast-member {
  position: relative;
  min-width: 0;
  border-radius: 7px;
}

.movie-director-list > a,
.movie-cast-member {
  cursor: pointer;
  text-decoration: none;
}

.movie-director-list > a:hover,
.movie-director-list > a:focus-visible {
  background: rgba(255, 255, 255, 0.08);
}

.movie-director-list > a:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.movie-cast-member:focus-visible {
  outline: none;
}

.movie-cast-member:focus-visible .movie-cast-portrait--missing {
  box-shadow: 0 0 0 2px var(--accent);
}

.movie-cast-portrait {
  position: relative;
  display: grid;
  aspect-ratio: 1;
  margin-bottom: 0.42rem;
  overflow: visible;
  place-items: center;
  border-radius: 7px;
  background: rgba(8, 13, 21, 0.78);
  color: rgba(215, 222, 232, 0.48);
}

.movie-cast-portrait img {
  position: absolute;
  inset-inline: 0;
  bottom: 0;
  width: 100%;
  height: 100%;
  border-radius: inherit;
  object-fit: cover;
  transition:
    height 220ms cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 220ms ease;
  will-change: height;
}

@media (hover: hover) and (pointer: fine) {
  .movie-cast-member:hover,
  .movie-cast-member:focus-visible {
    z-index: 3;
  }

  .movie-cast-member:hover .movie-cast-portrait img,
  .movie-cast-member:focus-visible .movie-cast-portrait img {
    height: 150%;
    box-shadow:
      0 0 0 2px var(--accent),
      0 10px 28px rgba(0, 0, 0, 0.42);
  }

  .movie-cast-member:hover .movie-cast-portrait--missing {
    box-shadow: 0 0 0 2px var(--accent);
  }
}

@media (prefers-reduced-motion: reduce) {
  .movie-cast-portrait img {
    transition: none;
  }
}

.movie-cast-member strong,
.movie-cast-member small {
  display: -webkit-box;
  overflow: hidden;
  -webkit-box-orient: vertical;
}

.movie-cast-member strong {
  -webkit-line-clamp: 2;
  color: #f4f7fb;
  font-size: 0.76rem;
  font-weight: 650;
  line-height: 1.2;
}

.movie-cast-member small {
  margin-top: 0.16rem;
  -webkit-line-clamp: 1;
  color: rgba(222, 228, 236, 0.62);
  font-size: 0.68rem;
  line-height: 1.25;
}

@media (max-width: 760px) {
  .movie-cast-list {
    display: flex;
    width: calc(100vw - 2rem);
    gap: 0.65rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
    scrollbar-width: thin;
  }

  .movie-cast-member {
    flex: 0 0 108px;
  }
}
</style>
