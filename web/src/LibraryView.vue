<script setup lang="ts">
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AppIcon from "./AppIcon.vue";
import MediaCard from "./MediaCard.vue";

const { artwork, availability, library, loadLibrary, t, title } =
  useNestrrApplicationContext();
</script>

<template>
  <section class="content page library-page">
    <div class="library-toolbar">
      <button
        :aria-label="t('rescan')"
        :title="t('rescan')"
        @click="loadLibrary"
      >
        <AppIcon name="refresh" />
      </button>
    </div>
    <div v-if="library.length" class="library-list">
      <RouterLink
        v-for="f in library"
        :key="f.path"
        :to="f.tmdbId ? `/${f.mediaType}/${f.tmdbId}` : '/library'"
        :class="{ actionable: f.tmdbId, disabled: !f.tmdbId }"
        @click="!f.tmdbId && $event.preventDefault()"
        ><img
          v-if="f.metadata?.poster_path"
          :src="artwork(f.metadata.poster_path)"
          alt="" />
        <div v-else class="file-placeholder"><AppIcon name="film" /></div>
        <div class="library-main">
          <div class="library-title">
            <b>{{ f.metadata?.title || f.metadata?.name || f.title }}</b
            ><span>{{
              f.kind === "movie"
                ? t("movie")
                : `S${String(f.season).padStart(2, "0")}E${String(f.episode).padStart(2, "0")}`
            }}</span>
          </div>
          <span class="availability-line yes"
            ><AppIcon name="checked" :size="15" />{{ t("downloaded") }}</span
          ><small>{{ f.path }}</small>
        </div>
        <AppIcon v-if="f.tmdbId" name="chevronRight"
      /></RouterLink>
    </div>
    <div v-else class="empty-state">
      <AppIcon name="download" :size="32" />
      <h2>{{ t("libraryEmpty") }}</h2>
      <p>{{ t("libraryEmptyHelp") }}</p>
    </div>
  </section>
</template>
