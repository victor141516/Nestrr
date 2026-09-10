<script setup lang="ts">
defineProps<{ label: string }>();
const model = defineModel<boolean>({ required: true });
const emit = defineEmits<{ change: [value: boolean] }>();

function update(event: Event): void {
  const checked = (event.target as HTMLInputElement).checked;
  model.value = checked;
  emit("change", checked);
}
</script>

<template>
  <label class="boolean-switch">
    <span>{{ label }}</span>
    <input :checked="model" type="checkbox" @change="update" />
    <i aria-hidden="true" />
  </label>
</template>

<style scoped>
.boolean-switch {
  display: inline-flex;
  height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 0.55rem;
  padding: 0 0.7rem;
  border: 1px solid var(--line);
  border-radius: 10px;
  background: var(--control);
  color: var(--muted);
  font-size: 0.84rem;
  white-space: nowrap;
}

.boolean-switch input {
  position: absolute;
  opacity: 0;
}

.boolean-switch i {
  position: relative;
  width: 34px;
  height: 20px;
  flex: 0 0 auto;
  border: 1px solid var(--line-strong);
  border-radius: 999px;
  background: var(--subtle-surface);
}

.boolean-switch i::after {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--muted);
  content: "";
  transition:
    transform 0.16s ease,
    background 0.16s ease;
}

.boolean-switch input:checked + i {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.boolean-switch input:checked + i::after {
  background: var(--accent);
  transform: translateX(14px);
}

.boolean-switch input:focus-visible + i {
  outline: 2px solid var(--accent);
  outline-offset: 3px;
}

@media (prefers-reduced-motion: reduce) {
  .boolean-switch i::after {
    transition-duration: 0s;
  }
}
</style>
