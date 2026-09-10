<script setup lang="ts">
import AppIcon from "./AppIcon.vue";

defineProps<{ placeholder: string; actionLabel: string }>();
const model = defineModel<string>({ required: true });
const emit = defineEmits<{ submit: []; settle: [] }>();

function onFocusOut(event: FocusEvent) {
  const form = event.currentTarget as HTMLFormElement;
  const next = event.relatedTarget;
  if (next instanceof Node && form.contains(next)) return;
  emit("settle");
}
</script>

<template>
  <form
    class="search-control"
    @submit.prevent="emit('submit')"
    @focusout="onFocusOut"
  >
    <div class="search-control-input">
      <AppIcon name="search" />
      <input v-model="model" :placeholder="placeholder" />
    </div>
    <button class="primary" :disabled="!model.trim()">
      <AppIcon name="search" />{{ actionLabel }}
    </button>
  </form>
</template>

<style scoped>
.search-control {
  display: grid;
  min-width: 0;
  grid-template-columns: minmax(220px, 1fr) auto;
  gap: 0.7rem;
}

.search-control-input {
  display: flex;
  height: 44px;
  min-width: 0;
  align-items: center;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--control);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease;
}

.search-control-input:focus-within {
  border-color: var(--accent);
  outline: 2px solid var(--accent);
  outline-offset: 2px;
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.search-control-input > :first-child {
  flex: 0 0 auto;
  margin-left: 0.9rem;
  color: var(--muted);
}

.search-control-input input {
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  padding: 0 0.9rem 0 0.55rem;
  border: 0;
  outline: 0;
  background: transparent;
}

.search-control > button {
  height: 44px;
  gap: 0.45rem;
  font-size: 0.84rem;
}

.search-control > button:hover:not(:disabled) {
  transform: none;
}

@media (max-width: 760px) {
  .search-control {
    grid-template-columns: 1fr;
  }
}
</style>
