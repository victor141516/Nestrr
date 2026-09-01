<script setup lang="ts">
import {
  nextTick,
  onBeforeUnmount,
  onMounted,
  onUnmounted,
  shallowRef,
  useTemplateRef,
} from "vue";

withDefaults(
  defineProps<{
    panelClass: string;
    labelledby: string;
    describedby?: string;
    role?: "dialog" | "alertdialog";
    busy?: boolean;
  }>(),
  {
    describedby: undefined,
    role: "dialog",
    busy: false,
  },
);

const emit = defineEmits<{ close: [] }>();
const dialogPanel = useTemplateRef<HTMLElement>("dialogPanel");
const opener = shallowRef<HTMLElement | null>(
  typeof document !== "undefined" &&
    document.activeElement instanceof HTMLElement
    ? document.activeElement
    : null,
);
let previousBodyOverflow = "";

const focusableSelector = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "summary",
  '[tabindex]:not([tabindex="-1"])',
].join(",");

function focusableElements(): HTMLElement[] {
  if (!dialogPanel.value) return [];
  return Array.from(
    dialogPanel.value.querySelectorAll<HTMLElement>(focusableSelector),
  ).filter((element) => {
    const style = window.getComputedStyle(element);
    return style.display !== "none" && style.visibility !== "hidden";
  });
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === "Escape") {
    event.preventDefault();
    emit("close");
    return;
  }
  if (event.key !== "Tab") return;

  const focusable = focusableElements();
  if (!focusable.length) {
    event.preventDefault();
    dialogPanel.value?.focus();
    return;
  }

  const first = focusable[0]!;
  const last = focusable[focusable.length - 1]!;
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

onMounted(async () => {
  previousBodyOverflow = document.body.style.overflow;
  document.body.style.overflow = "hidden";
  await nextTick();
  const initial =
    dialogPanel.value?.querySelector<HTMLElement>(
      "[data-dialog-initial-focus], [autofocus]",
    ) || focusableElements()[0];
  (initial || dialogPanel.value)?.focus();
});

onBeforeUnmount(() => {
  document.body.style.overflow = previousBodyOverflow;
});

onUnmounted(() => {
  opener.value?.focus();
});
</script>

<template>
  <div
    class="modal-backdrop"
    @click.self="emit('close')"
    @keydown="handleKeydown"
  >
    <section
      ref="dialogPanel"
      :class="panelClass"
      :role="role"
      aria-modal="true"
      :aria-labelledby="labelledby"
      :aria-describedby="describedby"
      :aria-busy="busy || undefined"
      tabindex="-1"
    >
      <slot />
    </section>
  </div>
</template>
