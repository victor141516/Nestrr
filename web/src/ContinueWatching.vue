<script setup lang="ts">
import type { TranslationKey } from "./i18n";
import type { ContinueItem } from "./useHomeLibrary";
import AppIcon from "./AppIcon.vue";

defineProps<{
  items: ContinueItem[];
  artwork: (path: string, size?: string) => string;
  percent: (progress: unknown) => number;
  translate: (key: TranslationKey) => string;
}>();
</script>

<template>
  <section v-if="items.length" class="content section-block continue-section">
    <div class="section-heading">
      <h1>{{ translate("continue") }}</h1>
    </div>
    <div class="continue-row">
      <RouterLink
        v-for="item in items"
        :key="item.progress.mediaType + item.progress.tmdbId"
        class="progress-card"
        :to="item.to"
      >
        <div
          class="progress-visual"
          :style="
            item.imagePath
              ? {
                  backgroundImage: `url(${artwork(item.imagePath, 'w780')})`,
                }
              : undefined
          "
        >
          <span><AppIcon name="play" /></span>
        </div>
        <div class="progress-copy">
          <div>
            <b>{{ item.title }}</b>
            <small v-if="item.progress.mediaType === 'episode'">
              {{ translate("season") }} {{ item.progress.seasonNumber }} ·
              {{ translate("episode") }} {{ item.progress.episodeNumber
              }}<template v-if="item.episodeTitle">
                · {{ item.episodeTitle }}</template
              >
            </small>
          </div>
          <small>{{ Math.round(percent(item.progress)) }}%</small>
        </div>
        <div class="bar">
          <i :style="{ width: `${percent(item.progress)}%` }" />
        </div>
      </RouterLink>
    </div>
  </section>
</template>
