<script setup lang="ts" generic="T extends string">
import AppSelect, { type AppSelectOption } from "./AppSelect.vue";

defineProps<{
  label: string;
  options: readonly AppSelectOption<T>[];
}>();
const model = defineModel<T>({ required: true });
const emit = defineEmits<{ change: [value: T] }>();
</script>

<template>
  <div class="sort-control">
    <span>{{ label }}</span>
    <AppSelect
      v-model="model"
      :label="label"
      :options="options"
      @change="emit('change', $event)"
    />
  </div>
</template>

<style scoped>
.sort-control {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 180px;
  height: 44px;
  align-items: center;
  gap: 0.45rem;
  padding: 0 0.4rem 0 0.75rem;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--control);
  color: var(--muted);
  font-size: 0.84rem;
  white-space: nowrap;
}

.sort-control:focus-within {
  border-color: var(--accent);
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

@media (max-width: 420px) {
  .sort-control {
    min-width: 0;
  }
}
</style>
