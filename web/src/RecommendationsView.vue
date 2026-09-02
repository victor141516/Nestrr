<script setup lang="ts">
import { watch } from "vue";
import AppIcon from "./AppIcon.vue";
import RecommendationGrid from "./RecommendationGrid.vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";
import { useRecommendations } from "./useRecommendations";

const { availability, checkCards, startedFor, systemMembershipsFor, t } =
  useNestrrApplicationContext();
const { error, items, load, loading, mediaType } = useRecommendations();

watch(items, (nextItems) => void checkCards(nextItems));
</script>

<template>
  <section class="content section-block recommendations-page">
    <div class="recommendations-heading">
      <h1>{{ t("recommendations") }}</h1>
      <div class="segmented" role="group" :aria-label="t('mediaType')">
        <button
          type="button"
          :class="{ active: mediaType === 'movie' }"
          :aria-pressed="mediaType === 'movie'"
          @click="mediaType = 'movie'"
        >
          <AppIcon name="film" />{{ t("movies") }}
        </button>
        <button
          type="button"
          :class="{ active: mediaType === 'tv' }"
          :aria-pressed="mediaType === 'tv'"
          @click="mediaType = 'tv'"
        >
          <AppIcon name="tv" />{{ t("shows") }}
        </button>
      </div>
    </div>
    <p v-if="error" class="inline-message error" role="alert">
      <AppIcon name="alert" />{{ error }}
      <button type="button" @click="load">{{ t("retry") }}</button>
    </p>
    <div v-if="loading && !items.length" class="checking-line" aria-busy="true">
      <span class="mini-spinner" />{{ t("loading") }}
    </div>
    <RecommendationGrid
      v-else-if="items.length"
      :items="items"
      :availability="availability"
      :memberships-for="systemMembershipsFor"
      :started-for="startedFor"
    />
    <div v-else-if="!loading && !error" class="empty-state">
      <AppIcon name="sparkles" :size="32" />
      <h2>{{ t("noRecommendations") }}</h2>
    </div>
  </section>
</template>

<style scoped>
.recommendations-page {
  min-height: 55vh;
}

.recommendations-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.recommendations-heading h1 {
  margin: 0;
}

.segmented button {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
}

@media (max-width: 640px) {
  .recommendations-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .segmented button {
    flex: 1;
    justify-content: center;
  }
}
</style>
