<script setup lang="ts">
import type { RouteLocationRaw } from "vue-router";
import AppIcon from "./AppIcon.vue";
import type { MediaMetadataItem } from "./mediaMetadata";

defineProps<{
  icon: string;
  label: string;
  iconLabel?: string;
  iconTo?: RouteLocationRaw;
  items: MediaMetadataItem[];
}>();
</script>

<template>
  <div class="detail-metadata-group" role="group" :aria-label="label">
    <RouterLink
      v-if="iconTo"
      class="detail-metadata-icon actionable"
      :to="iconTo"
      :aria-label="iconLabel || label"
      :title="iconLabel || label"
    >
      <AppIcon :name="icon" :size="15" />
    </RouterLink>
    <span v-else class="detail-metadata-icon" aria-hidden="true">
      <AppIcon :name="icon" :size="15" />
    </span>
    <span class="detail-metadata-items">
      <template v-for="item in items" :key="item.id">
        <RouterLink
          v-if="item.to"
          class="detail-metadata-item actionable"
          :to="item.to"
        >
          {{ item.label }}
        </RouterLink>
        <span v-else class="detail-metadata-item">{{ item.label }}</span>
      </template>
    </span>
  </div>
</template>

<style scoped>
.detail-metadata-group {
  display: inline-flex;
  max-width: 100%;
  align-items: stretch;
  overflow: hidden;
  border: 0;
  border-radius: 6px;
  background: rgba(15, 23, 34, 0.78);
  color: #c4cbd4;
  font-size: 0.8rem;
}

.detail-metadata-icon {
  display: inline-grid;
  width: auto;
  min-width: 0;
  min-height: 0;
  padding: 0.25rem 0.52rem;
  place-items: center;
  border-inline-end: 1px solid rgba(255, 255, 255, 0.12);
  color: var(--accent);
}

.detail-metadata-items {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  overflow-x: auto;
  scrollbar-width: none;
}

.detail-metadata-items::-webkit-scrollbar {
  display: none;
}

.detail-metadata-item {
  display: inline-flex;
  flex: 0 0 auto;
  min-height: 0;
  align-items: center;
  padding: 0.25rem 0.52rem;
  white-space: nowrap;
}

.detail-metadata-item + .detail-metadata-item {
  border-inline-start: 1px solid rgba(255, 255, 255, 0.12);
}

.actionable {
  cursor: pointer;
}

.actionable:focus-visible {
  position: relative;
  z-index: 1;
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}
</style>
