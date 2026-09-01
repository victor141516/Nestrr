<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import AppIcon from "./AppIcon.vue";

const props = defineProps<{
  label: string;
  modelValue: string[];
  options: string[];
  allLabel: string;
  noneLabel: string;
  selectedLabel: string;
  selectAllLabel: string;
  selectNoneLabel: string;
}>();
const emit = defineEmits<{ "update:modelValue": [value: string[]] }>();

const open = ref(false);
const root = ref<HTMLElement | null>(null);
const allSelected = computed(
  () =>
    props.options.length > 0 &&
    props.options.every((option) => props.modelValue.includes(option)),
);
const summary = computed(() => {
  if (allSelected.value || !props.options.length) return props.allLabel;
  if (!props.modelValue.length) return props.noneLabel;
  return props.selectedLabel.replace(
    "{count}",
    String(props.modelValue.length),
  );
});

function toggle(value: string) {
  emit(
    "update:modelValue",
    props.modelValue.includes(value)
      ? props.modelValue.filter((item) => item !== value)
      : [...props.modelValue, value],
  );
}
function selectAll() {
  emit("update:modelValue", [...props.options]);
}
function selectNone() {
  emit("update:modelValue", []);
}
function onDocumentPointerDown(event: PointerEvent) {
  if (!root.value?.contains(event.target as Node | null)) open.value = false;
}

onMounted(() =>
  document.addEventListener("pointerdown", onDocumentPointerDown),
);
onBeforeUnmount(() =>
  document.removeEventListener("pointerdown", onDocumentPointerDown),
);
</script>

<template>
  <div ref="root" class="tracker-multiselect" @keydown.esc="open = false">
    <span class="torrent-tag-label">{{ label }}</span>
    <button
      type="button"
      class="tracker-select-trigger"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span>{{ summary }}</span
      ><AppIcon name="chevronDown" :size="16" />
    </button>
    <div v-if="open" class="tracker-select-menu">
      <div class="tracker-select-actions">
        <button
          type="button"
          :aria-label="selectAllLabel"
          :title="selectAllLabel"
          @click="selectAll"
        >
          <AppIcon name="checked" :size="17" />
        </button>
        <button
          type="button"
          :aria-label="selectNoneLabel"
          :title="selectNoneLabel"
          @click="selectNone"
        >
          <AppIcon name="x" :size="17" />
        </button>
      </div>
      <div class="tracker-select-options" role="group" :aria-label="label">
        <button
          v-for="option in options"
          :key="option"
          type="button"
          :class="{ selected: modelValue.includes(option) }"
          :aria-pressed="modelValue.includes(option)"
          @click="toggle(option)"
        >
          <span class="custom-checkbox"
            ><AppIcon
              v-if="modelValue.includes(option)"
              name="check"
              :size="14" /></span
          ><span>{{ option }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
