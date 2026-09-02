<script setup lang="ts">
import MediaCard from "./MediaCard.vue";
import type { RecommendationItem } from "./useRecommendations";

defineProps<{
  items: RecommendationItem[];
  availability: Record<string, unknown>;
  membershipsFor: (mediaType: string, tmdbId: number) => string[];
  startedFor: (mediaType: string, tmdbId: number) => boolean;
}>();
</script>

<template>
  <div class="media-grid recommendation-results">
    <MediaCard
      v-for="item in items"
      :key="`${item.media_type}:${item.id}`"
      :m="item"
      :to="`/${item.media_type}/${item.id}`"
      :availability="availability[`${item.media_type}:${item.id}`]"
      :memberships="membershipsFor(item.media_type, item.id)"
      :started="startedFor(item.media_type, item.id)"
    />
  </div>
</template>
