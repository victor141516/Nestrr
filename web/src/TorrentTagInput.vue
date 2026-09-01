<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import Tagify from "@yaireo/tagify";
import "@yaireo/tagify/dist/tagify.css";

const props = defineProps<{
  id: string;
  label: string;
  modelValue: string[];
  placeholder: string;
}>();
const emit = defineEmits<{ "update:modelValue": [value: string[]] }>();

const input = ref<HTMLInputElement | null>(null);
let tagify: Tagify | null = null;
let syncing = false;

function normalized(values: unknown[]) {
  return values
    .map((value) => String(value).trim())
    .filter((value, index, all) => value && all.indexOf(value) === index);
}

function currentValues() {
  return normalized((tagify?.value || []).map((tag) => tag.value));
}

function sameValues(left: string[], right: string[]) {
  return (
    left.length === right.length &&
    left.every((value, index) => value === right[index])
  );
}

function syncFromModel() {
  if (!tagify) return;
  const next = normalized(props.modelValue);
  if (sameValues(currentValues(), next)) return;
  syncing = true;
  tagify.removeAllTags({ withoutChangeEvent: true });
  if (next.length) tagify.addTags(next);
  syncing = false;
}

function onChange() {
  if (!syncing) emit("update:modelValue", currentValues());
}

function commitPending() {
  const value = String(tagify?.DOM.input.textContent || "").trim();
  if (value) tagify?.addTags([{ value }]);
}

function onInputKeydown(event: KeyboardEvent) {
  if (event.key === "Tab") commitPending();
}

onMounted(() => {
  if (!input.value) return;
  tagify = new Tagify(input.value, {
    // Null disables comma splitting. Enter remains the only commit key, so a
    // filter term can safely contain spaces, commas, and punctuation.
    delimiters: null as unknown as string,
    addTagOnBlur: false,
    backspace: true,
    duplicates: false,
    editTags: { clicks: 2, keepInvalid: false },
    dropdown: { enabled: false },
    trim: true,
  });
  tagify.DOM.scope.setAttribute("aria-labelledby", `${props.id}-label`);
  tagify.DOM.input.setAttribute(
    "aria-label",
    `${props.label}. ${props.placeholder}`,
  );
  tagify.DOM.originalInput.setAttribute("aria-hidden", "true");
  tagify.DOM.originalInput.setAttribute("tabindex", "-1");
  tagify.on("change", onChange);
  tagify.DOM.input.addEventListener("blur", commitPending);
  tagify.DOM.input.addEventListener("keydown", onInputKeydown);
  syncFromModel();
});

watch(() => props.modelValue, syncFromModel, { deep: true });
watch(
  () => props.placeholder,
  (placeholder) => tagify?.setPlaceholder(placeholder),
);

onBeforeUnmount(() => {
  tagify?.DOM.input.removeEventListener("blur", commitPending);
  tagify?.DOM.input.removeEventListener("keydown", onInputKeydown);
  tagify?.off("change", onChange);
  tagify?.destroy();
  tagify = null;
});
</script>

<template>
  <div class="torrent-tag-field" :data-testid="`${id}-field`">
    <span :id="`${id}-label`" class="torrent-tag-label">{{ label }}</span>
    <input :id="id" ref="input" :placeholder="placeholder" />
  </div>
</template>
