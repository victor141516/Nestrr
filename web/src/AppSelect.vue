<script setup lang="ts" generic="T extends string">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  shallowRef,
  useId,
  useTemplateRef,
  watch,
} from "vue";
import AppIcon from "./AppIcon.vue";

export type AppSelectOption<T extends string = string> = {
  value: T;
  label: string;
};

const props = defineProps<{
  label: string;
  options: readonly AppSelectOption<T>[];
}>();
const model = defineModel<T>({ required: true });
const emit = defineEmits<{ change: [value: T] }>();

const open = shallowRef(false);
const root = useTemplateRef<HTMLElement>("root");
const trigger = useTemplateRef<HTMLButtonElement>("trigger");
const menuId = `app-select-${useId()}`;
const selectedOption = computed(
  () =>
    props.options.find((option) => option.value === model.value) ||
    props.options[0],
);

function optionButtons(): HTMLButtonElement[] {
  return Array.from(
    root.value?.querySelectorAll<HTMLButtonElement>("[role='menuitemradio']") ||
      [],
  );
}

function closeMenu(returnFocus = false): void {
  if (!open.value) return;
  open.value = false;
  if (returnFocus) void nextTick(() => trigger.value?.focus());
}

async function openMenu(focus: "selected" | "first" | "last" = "selected") {
  open.value = true;
  await nextTick();
  const buttons = optionButtons();
  if (!buttons.length) return;
  if (focus === "first") buttons[0]?.focus();
  else if (focus === "last") buttons.at(-1)?.focus();
  else
    (
      buttons.find((button) => button.dataset.value === model.value) ||
      buttons[0]
    )?.focus();
}

function toggleMenu(): void {
  if (open.value) closeMenu();
  else void openMenu();
}

function selectOption(value: T): void {
  model.value = value;
  emit("change", value);
  closeMenu(true);
}

function handleTriggerKeydown(event: KeyboardEvent): void {
  if (event.key === "ArrowDown") {
    event.preventDefault();
    void openMenu("first");
  } else if (event.key === "ArrowUp") {
    event.preventDefault();
    void openMenu("last");
  } else if (event.key === "Escape") {
    event.preventDefault();
    closeMenu(true);
  }
}

function handleMenuKeydown(event: KeyboardEvent): void {
  const buttons = optionButtons();
  const current = buttons.indexOf(document.activeElement as HTMLButtonElement);
  let next = current;
  if (event.key === "ArrowDown") next = (current + 1) % buttons.length;
  else if (event.key === "ArrowUp")
    next = (current - 1 + buttons.length) % buttons.length;
  else if (event.key === "Home") next = 0;
  else if (event.key === "End") next = buttons.length - 1;
  else if (event.key === "Escape") {
    event.preventDefault();
    closeMenu(true);
    return;
  } else if (event.key === "Tab") {
    closeMenu();
    return;
  } else return;
  event.preventDefault();
  buttons[next]?.focus();
}

function handleDocumentClick(event: MouseEvent): void {
  if (!root.value?.contains(event.target as Node)) closeMenu();
}

watch(model, () => closeMenu());
onMounted(() => document.addEventListener("click", handleDocumentClick));
onBeforeUnmount(() =>
  document.removeEventListener("click", handleDocumentClick),
);
</script>

<template>
  <div ref="root" class="app-select">
    <button
      ref="trigger"
      type="button"
      class="app-select-trigger"
      :aria-controls="menuId"
      :aria-expanded="open"
      :aria-label="`${label}: ${selectedOption?.label || ''}`"
      aria-haspopup="menu"
      @click="toggleMenu"
      @keydown="handleTriggerKeydown"
    >
      <span>{{ selectedOption?.label }}</span>
      <AppIcon class="app-select-chevron" name="chevronDown" :size="15" />
    </button>
    <div
      v-if="open"
      :id="menuId"
      class="app-select-menu"
      role="menu"
      :aria-label="label"
      @keydown="handleMenuKeydown"
    >
      <button
        v-for="option in options"
        :key="option.value"
        type="button"
        role="menuitemradio"
        :aria-checked="option.value === model"
        :data-value="option.value"
        @click="selectOption(option.value)"
      >
        <span>{{ option.label }}</span>
        <AppIcon
          class="app-select-check"
          name="check"
          :size="15"
          :class="{ visible: option.value === model }"
        />
      </button>
    </div>
  </div>
</template>

<style scoped>
.app-select {
  position: relative;
  min-width: 0;
  height: 100%;
  color: var(--text);
}

.app-select-trigger {
  display: flex;
  width: 100%;
  height: 100%;
  min-height: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.45rem;
  padding: 0 0.55rem 0 0.35rem;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text);
  font-size: inherit;
  line-height: 1;
}

.app-select-trigger:hover:not(:disabled) {
  border-color: transparent;
  background: var(--control-hover);
}

.app-select-trigger > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-select-chevron {
  flex: 0 0 auto;
  color: var(--muted);
  transition: transform 0.16s ease;
}

.app-select-trigger[aria-expanded="true"] .app-select-chevron {
  transform: rotate(180deg);
}

.app-select-menu {
  position: absolute;
  z-index: 45;
  top: calc(100% + 0.45rem);
  right: 0;
  display: grid;
  min-width: max(100%, 190px);
  gap: 2px;
  padding: 0.35rem;
  border: 1px solid var(--line);
  border-radius: 11px;
  background: var(--floating);
  box-shadow: var(--shadow);
}

.app-select-menu > button {
  display: flex;
  width: 100%;
  min-height: 38px;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.5rem 0.65rem;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  font-size: inherit;
  font-weight: 650;
  text-align: left;
  white-space: nowrap;
}

.app-select-menu > button:hover,
.app-select-menu > button:focus-visible {
  border-color: transparent;
  background: var(--control-hover);
  color: var(--text);
}

.app-select-menu > button[aria-checked="true"] {
  background: var(--accent-soft);
  color: var(--text);
}

.app-select-check {
  flex: 0 0 auto;
  color: var(--accent);
  opacity: 0;
}

.app-select-check.visible {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .app-select-chevron {
    transition-duration: 0s;
  }
}
</style>
