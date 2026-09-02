<script setup lang="ts">
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from "vue";
import Tagify from "@yaireo/tagify";
import "@yaireo/tagify/dist/tagify.css";
import type { FilterOption } from "./movieFilters";

type EntityTag = Tagify.BaseTagData & { id: number; name: string };
type PublishSuggestions = (options: FilterOption[]) => void;

const props = defineProps<{
  id: string;
  label: string;
  modelValue: FilterOption[];
  placeholder: string;
  search: (
    query: string,
    signal: AbortSignal,
    publish: PublishSuggestions,
  ) => Promise<void>;
  resolve: (ids: number[], signal?: AbortSignal) => Promise<FilterOption[]>;
  dropdownClass: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: FilterOption[]];
}>();

const input = useTemplateRef<HTMLInputElement>("input");
let tagify: Tagify<EntityTag> | null = null;
let syncing = false;
let debounceTimer: number | undefined;
let searchController: AbortController | null = null;
let hydrateController: AbortController | null = null;
let searchRevision = 0;
let focused = false;

function normalized(options: FilterOption[]) {
  const seen = new Set<number>();
  return options
    .map((option) => ({
      ...option,
      id: Number(option.id),
      name: String(option.name || option.id).trim(),
    }))
    .filter((option) => {
      if (!Number.isInteger(option.id) || option.id <= 0 || seen.has(option.id))
        return false;
      seen.add(option.id);
      return true;
    });
}

const toTag = (option: FilterOption): EntityTag => ({
  value: String(option.id),
  id: option.id,
  name: option.name,
});

function currentOptions(): FilterOption[] {
  return normalized(
    (tagify?.value || []).map((tag) => ({ id: tag.id, name: tag.name })),
  );
}

function sameOptions(left: FilterOption[], right: FilterOption[]) {
  return (
    left.length === right.length &&
    left.every(
      (option, index) =>
        option.id === right[index]?.id && option.name === right[index]?.name,
    )
  );
}

function selectedTags() {
  return normalized(props.modelValue).map(toTag);
}

function setSuggestions(options: FilterOption[], query: string) {
  if (!tagify) return;
  const merged = new Map<number, FilterOption>();
  for (const option of [...normalized(props.modelValue), ...options])
    merged.set(option.id, option);
  tagify.whitelist = [...merged.values()].map(toTag);
  if (focused) tagify.dropdown.show(query);
}

function syncFromModel() {
  if (!tagify) return;
  const next = normalized(props.modelValue);
  if (sameOptions(currentOptions(), next)) return;
  syncing = true;
  tagify.whitelist = selectedTags();
  tagify.removeAllTags({ withoutChangeEvent: true });
  if (next.length) tagify.addTags(next.map(toTag));
  syncing = false;
}

function emitCurrent() {
  if (!syncing) emit("update:modelValue", currentOptions());
}

function cancelSearch() {
  searchRevision += 1;
  if (debounceTimer !== undefined) window.clearTimeout(debounceTimer);
  debounceTimer = undefined;
  searchController?.abort();
  searchController = null;
  tagify?.loading(false);
}

function scheduleSearch(value: string) {
  cancelSearch();
  const query = value.trim();
  if (query.length < 2) {
    if (tagify) tagify.whitelist = selectedTags();
    tagify?.dropdown.hide();
    return;
  }
  const revision = searchRevision;
  debounceTimer = window.setTimeout(() => void runSearch(query, revision), 180);
}

async function runSearch(query: string, revision: number) {
  const controller = new AbortController();
  searchController = controller;
  tagify?.loading(true);
  try {
    await props.search(query, controller.signal, (options) => {
      if (!controller.signal.aborted && revision === searchRevision) {
        tagify?.loading(false);
        setSuggestions(options, query);
      }
    });
  } catch (reason) {
    if (!(reason instanceof DOMException && reason.name === "AbortError"))
      tagify?.dropdown.hide();
  } finally {
    if (searchController === controller) {
      searchController = null;
      tagify?.loading(false);
    }
  }
}

function onInput(event: CustomEvent<Tagify.InputEventData<EntityTag>>) {
  if ("value" in event.detail) scheduleSearch(event.detail.value);
}

function onEditInput(event: CustomEvent<Tagify.EditInputEventData<EntityTag>>) {
  scheduleSearch(event.detail.data.newValue);
}

async function hydrateSelection() {
  const unresolved = normalized(props.modelValue).filter(
    (option) => option.name === String(option.id),
  );
  if (!unresolved.length) return;
  hydrateController?.abort();
  const controller = new AbortController();
  hydrateController = controller;
  try {
    const resolved = await props.resolve(
      unresolved.map((option) => option.id),
      controller.signal,
    );
    const names = new Map(resolved.map((option) => [option.id, option.name]));
    emit(
      "update:modelValue",
      normalized(props.modelValue).map((option) => ({
        ...option,
        name: names.get(option.id) || option.name,
      })),
    );
  } catch {
    // IDs remain valid when a label cannot be refreshed.
  }
  if (hydrateController === controller) hydrateController = null;
}

function onFocus() {
  focused = true;
}

function onBlur() {
  focused = false;
}

onMounted(() => {
  if (!input.value) return;
  tagify = new Tagify<EntityTag>(input.value, {
    delimiters: null as unknown as string,
    enforceWhitelist: true,
    addTagOnBlur: false,
    backspace: true,
    duplicates: false,
    editTags: { clicks: 2, keepInvalid: false },
    tagTextProp: "name",
    trim: true,
    whitelist: selectedTags(),
    autoComplete: { enabled: false },
    dropdown: {
      enabled: 2,
      maxItems: 24,
      fuzzySearch: true,
      highlightFirst: true,
      closeOnSelect: true,
      mapValueTo: "name",
      searchKeys: ["name"],
      classname: props.dropdownClass,
    },
  });
  tagify.DOM.scope.setAttribute("aria-labelledby", `${props.id}-label`);
  tagify.DOM.input.setAttribute(
    "aria-label",
    `${props.label}. ${props.placeholder}`,
  );
  tagify.DOM.originalInput.setAttribute("aria-hidden", "true");
  tagify.DOM.originalInput.setAttribute("tabindex", "-1");
  tagify.on("change", emitCurrent);
  tagify.on("focus", onFocus);
  tagify.on("blur", onBlur);
  tagify.on("input", onInput);
  tagify.on("edit:input", onEditInput);
  syncFromModel();
  void hydrateSelection();
});

watch(() => props.modelValue, syncFromModel, { deep: true });
watch(
  () => props.placeholder,
  (placeholder) => tagify?.setPlaceholder(placeholder),
);

onBeforeUnmount(() => {
  cancelSearch();
  hydrateController?.abort();
  tagify?.off("change", emitCurrent);
  tagify?.off("focus", onFocus);
  tagify?.off("blur", onBlur);
  tagify?.off("input", onInput);
  tagify?.off("edit:input", onEditInput);
  tagify?.destroy();
  tagify = null;
});
</script>

<template>
  <div class="torrent-tag-field keyword-tag-field" :data-testid="`${id}-field`">
    <span :id="`${id}-label`" class="torrent-tag-label">{{ label }}</span>
    <input :id="id" ref="input" :placeholder="placeholder" />
  </div>
</template>
