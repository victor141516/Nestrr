<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import type { ListDirection, ListSort } from "./listViewState";
import MediaTypeSwitch from "./MediaTypeSwitch.vue";
import type { CatalogMediaType } from "./mediaType";
import SortControl from "./SortControl.vue";

const props = defineProps<{
  mediaType: CatalogMediaType;
  sort: ListSort;
  direction: ListDirection;
  translate: (key: TranslationKey) => string;
}>();
const emit = defineEmits<{
  selectMediaType: [value: CatalogMediaType];
  selectSort: [value: ListSort];
  toggleDirection: [];
}>();

const sortOptions = computed(() => [
  { value: "addedAt" as const, label: props.translate("dateAdded") },
  { value: "title" as const, label: props.translate("titleLabel") },
  { value: "releaseDate" as const, label: props.translate("releaseDate") },
  { value: "rating" as const, label: props.translate("rating") },
  { value: "popularity" as const, label: props.translate("popularity") },
  { value: "runtime" as const, label: props.translate("runtime") },
]);
</script>

<template>
  <div class="list-toolbar">
    <MediaTypeSwitch
      :model-value="mediaType"
      :label="translate('contentType')"
      :translate="translate"
      @change="emit('selectMediaType', $event)"
    />
    <SortControl
      :model-value="sort"
      :label="translate('sortBy')"
      :options="sortOptions"
      @change="emit('selectSort', $event)"
    />
    <button
      class="sort-direction"
      :aria-label="translate('sortDirection')"
      :title="translate('sortDirection')"
      @click="emit('toggleDirection')"
    >
      <AppIcon name="arrowUp" :class="{ descending: direction === 'desc' }" />
    </button>
  </div>
</template>

<style scoped>
.list-toolbar {
  display: flex;
  align-items: center;
  gap: 0.7rem;
}
.sort-direction {
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  padding: 0;
}
.sort-direction :deep(.descending) {
  transform: rotate(180deg);
}
@media (max-width: 680px) {
  .list-toolbar {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 44px;
  }
  .list-toolbar > :first-child {
    grid-column: 1 / -1;
  }
}
</style>
