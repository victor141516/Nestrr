<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import type { CatalogMediaType } from "./mediaType";

const props = defineProps<{
  label: string;
  translate: (key: TranslationKey) => string;
}>();
const model = defineModel<CatalogMediaType>({ required: true });
const emit = defineEmits<{ change: [value: CatalogMediaType] }>();

const options = computed(() => [
  { value: "movie" as const, label: props.translate("movies"), icon: "film" },
  { value: "tv" as const, label: props.translate("shows"), icon: "tv" },
]);
const selectedIndex = computed(() =>
  Math.max(
    0,
    options.value.findIndex((option) => option.value === model.value),
  ),
);
const trackStyle = computed(() => ({
  "--segment-count": String(options.value.length),
  "--segment-index": String(selectedIndex.value),
}));

function select(value: CatalogMediaType): void {
  model.value = value;
  emit("change", value);
}
</script>

<template>
  <div
    class="media-type-switch"
    :style="trackStyle"
    role="group"
    :aria-label="label"
  >
    <i class="media-type-selection" aria-hidden="true" />
    <button
      v-for="option in options"
      :key="option.value"
      type="button"
      :aria-pressed="option.value === model"
      @click="select(option.value)"
    >
      <AppIcon :name="option.icon" />{{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.media-type-switch {
  --control-height: 44px;
  position: relative;
  display: grid;
  grid-template-columns: repeat(var(--segment-count), minmax(0, 1fr));
  min-width: calc(var(--segment-count) * 104px + 6px);
  height: var(--control-height);
  gap: 3px;
  padding: 3px;
  border: 1px solid var(--line);
  border-radius: 11px;
  background: var(--control);
  isolation: isolate;
}

.media-type-selection {
  position: absolute;
  z-index: -1;
  top: 3px;
  bottom: 3px;
  left: 3px;
  width: calc(
    (100% - 6px - (var(--segment-count) - 1) * 3px) / var(--segment-count)
  );
  border-radius: 8px;
  background: var(--subtle-surface);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.18);
  transform: translateX(calc(var(--segment-index) * (100% + 3px)));
  transition: transform 0.2s ease;
}

.media-type-switch button {
  z-index: 1;
  display: inline-flex;
  min-width: 0;
  min-height: 36px;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  padding: 0.45rem 0.75rem;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 0.84rem;
  white-space: nowrap;
}

.media-type-switch button:hover,
.media-type-switch button:focus-visible {
  border-color: transparent;
  background: var(--control-hover);
  transform: none;
}

.media-type-switch button[aria-pressed="true"] {
  background: transparent;
  color: var(--text);
}

@media (max-width: 420px) {
  .media-type-switch {
    width: 100%;
    min-width: 0;
  }

  .media-type-switch button {
    padding-inline: 0.45rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .media-type-selection {
    transition-duration: 0s;
  }
}
</style>
