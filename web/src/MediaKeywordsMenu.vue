<script setup lang="ts">
import {
  computed,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useTemplateRef,
} from "vue";
import AppIcon from "./AppIcon.vue";
import MediaMetadataGroup from "./MediaMetadataGroup.vue";
import type { MediaMetadataItem } from "./mediaMetadata";

const props = defineProps<{
  items: MediaMetadataItem[];
  label: string;
  noResultsLabel: string;
  searchLabel: string;
}>();

const inlineLimit = 6;
const collator = new Intl.Collator(undefined, {
  numeric: true,
  sensitivity: "base",
});
const query = shallowRef("");
const root = useTemplateRef<HTMLDetailsElement>("root");
const summary = useTemplateRef<HTMLElement>("summary");
const visibleItems = computed(() => {
  const normalized = query.value.trim().toLocaleLowerCase();
  const matches = normalized
    ? props.items.filter((item) =>
        item.label.toLocaleLowerCase().includes(normalized),
      )
    : props.items;
  return [...matches].sort((left, right) =>
    collator.compare(left.label, right.label),
  );
});

function close({ restoreFocus = false } = {}) {
  if (!root.value?.open) return;
  root.value.open = false;
  query.value = "";
  if (restoreFocus) summary.value?.focus();
}

function onPointerDown(event: PointerEvent) {
  if (
    root.value?.open &&
    event.target instanceof Node &&
    !root.value.contains(event.target)
  )
    close();
}

function onKeyDown(event: KeyboardEvent) {
  if (event.key === "Escape" && root.value?.open) {
    event.preventDefault();
    close({ restoreFocus: true });
  }
}

onMounted(() => {
  document.addEventListener("pointerdown", onPointerDown);
  document.addEventListener("keydown", onKeyDown);
});

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", onPointerDown);
  document.removeEventListener("keydown", onKeyDown);
});
</script>

<template>
  <MediaMetadataGroup
    v-if="items.length <= inlineLimit"
    icon="sparkles"
    :label="label"
    :items="items"
  />
  <details v-else ref="root" class="detail-keywords-menu">
    <summary ref="summary" :aria-label="label">
      <AppIcon name="sparkles" :size="15" />
      <span>{{ label }}</span>
      <small>{{ items.length }}</small>
      <AppIcon class="detail-keywords-chevron" name="chevronDown" :size="14" />
    </summary>
    <div class="detail-keywords-popover">
      <label class="sr-only" for="detail-keyword-search">{{
        searchLabel
      }}</label>
      <div v-if="items.length > 8" class="detail-keyword-search">
        <AppIcon name="search" :size="15" />
        <input
          id="detail-keyword-search"
          v-model="query"
          type="search"
          :placeholder="searchLabel"
        />
      </div>
      <div class="detail-keyword-list">
        <RouterLink
          v-for="item in visibleItems"
          :key="item.id"
          :to="item.to || '/'"
        >
          {{ item.label }}
        </RouterLink>
        <p v-if="!visibleItems.length">{{ noResultsLabel }}</p>
      </div>
    </div>
  </details>
</template>

<style scoped>
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  clip-path: inset(50%);
}

.detail-keywords-menu {
  position: relative;
  z-index: 5;
}

.detail-keywords-menu > summary {
  display: inline-flex;
  min-height: 0;
  align-items: center;
  gap: 0.45rem;
  padding: 0.25rem 0.52rem;
  border: 0;
  border-radius: 6px;
  background: rgba(15, 23, 34, 0.78);
  color: #c4cbd4;
  font-size: 0.8rem;
  cursor: pointer;
  list-style: none;
}

.detail-keywords-menu > summary::-webkit-details-marker {
  display: none;
}

.detail-keywords-menu > summary:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.detail-keywords-menu > summary > svg:first-child {
  color: var(--accent);
}

.detail-keywords-menu > summary small {
  color: #8f9aa8;
  font-size: 0.72rem;
}

.detail-keywords-chevron {
  transition: transform 0.16s ease;
}

.detail-keywords-menu[open] .detail-keywords-chevron {
  transform: rotate(180deg);
}

.detail-keywords-popover {
  position: absolute;
  top: calc(100% + 0.45rem);
  left: 0;
  width: min(300px, calc(100vw - 2rem));
  padding: 0.42rem;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--panel);
  box-shadow: var(--shadow);
}

.detail-keyword-search {
  display: flex;
  height: 32px;
  align-items: center;
  gap: 0.4rem;
  margin-bottom: 0.32rem;
  padding-inline: 0.52rem;
  border: 1px solid var(--line);
  border-radius: 6px;
  color: var(--muted);
}

.detail-keyword-search:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.detail-keyword-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--text);
}

.detail-keyword-list {
  display: grid;
  max-height: min(280px, 46vh);
  overflow-y: auto;
  overscroll-behavior: contain;
}

.detail-keyword-list a,
.detail-keyword-list p {
  margin: 0;
  min-height: 0;
  padding: 0.25rem 0.52rem;
  border-radius: 5px;
  color: var(--text-secondary);
  font-size: 0.8rem;
  line-height: 1.35;
}

.detail-keyword-list a:hover,
.detail-keyword-list a:focus-visible {
  outline: 0;
  background: var(--accent-soft);
  color: var(--text);
}

:global(html[data-theme="light"]) .detail-keywords-menu > summary {
  background: rgba(10, 16, 25, 0.76);
  color: #e5e9ef;
}

@media (max-width: 760px) {
  .detail-keywords-popover {
    position: fixed;
    top: auto;
    right: 1rem;
    bottom: calc(76px + env(safe-area-inset-bottom));
    left: 1rem;
    width: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .detail-keywords-chevron {
    transition-duration: 0s;
  }
}
</style>
