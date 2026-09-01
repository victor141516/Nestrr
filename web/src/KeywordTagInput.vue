<script setup lang="ts">
import { onBeforeUnmount, onMounted, useTemplateRef, watch } from "vue";
import Tagify from "@yaireo/tagify";
import "@yaireo/tagify/dist/tagify.css";
import {
  mergeKeywordSuggestions,
  resolveKeywords,
  searchKeywords,
} from "./keywordCatalog";
import type { KeywordOption } from "./movieFilters";

type KeywordTag = Tagify.BaseTagData & {
  id: number;
  name: string;
};

const props = defineProps<{
  id: string;
  label: string;
  modelValue: KeywordOption[];
  placeholder: string;
}>();
const emit = defineEmits<{
  "update:modelValue": [value: KeywordOption[]];
}>();

const input = useTemplateRef<HTMLInputElement>("input");
let tagify: Tagify<KeywordTag> | null = null;
let syncing = false;
let debounceTimer: number | undefined;
let searchController: AbortController | null = null;
let hydrateController: AbortController | null = null;
let searchRevision = 0;
let focused = false;

function normalized(options: KeywordOption[]) {
  const seen = new Set<number>();
  return options
    .map((option) => ({
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

const toTag = (option: KeywordOption): KeywordTag => ({
  value: String(option.id),
  id: option.id,
  name: option.name,
});

function currentOptions(): KeywordOption[] {
  return normalized(
    (tagify?.value || []).map((tag) => ({ id: tag.id, name: tag.name })),
  );
}

function sameOptions(left: KeywordOption[], right: KeywordOption[]) {
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

function setSuggestions(options: KeywordOption[], query: string) {
  if (!tagify) return;
  const merged = new Map<number, KeywordOption>();
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
    const prefix = await searchKeywords(query, "prefix", controller.signal);
    if (controller.signal.aborted || revision !== searchRevision) return;
    tagify?.loading(false);
    setSuggestions(prefix, query);

    const contains = await searchKeywords(query, "contains", controller.signal);
    if (controller.signal.aborted || revision !== searchRevision) return;
    setSuggestions(mergeKeywordSuggestions(prefix, contains), query);
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

function onInput(event: CustomEvent<Tagify.InputEventData<KeywordTag>>) {
  if ("value" in event.detail) scheduleSearch(event.detail.value);
}

function onEditInput(
  event: CustomEvent<Tagify.EditInputEventData<KeywordTag>>,
) {
  scheduleSearch(event.detail.data.newValue);
}

function onFocus() {
  focused = true;
}

function onBlur() {
  focused = false;
}

async function hydrateSelection() {
  const unresolved = normalized(props.modelValue).filter(
    (keyword) => keyword.name === String(keyword.id),
  );
  if (!unresolved.length) return;
  hydrateController?.abort();
  const controller = new AbortController();
  hydrateController = controller;
  try {
    const resolved = await resolveKeywords(
      unresolved.map((keyword) => keyword.id),
      controller.signal,
    );
    const names = new Map(
      resolved.map((keyword) => [keyword.id, keyword.name]),
    );
    emit(
      "update:modelValue",
      normalized(props.modelValue).map((keyword) => ({
        ...keyword,
        name: names.get(keyword.id) || keyword.name,
      })),
    );
  } catch {
    // The selected IDs remain valid even when the local catalog is unavailable.
  }
  if (hydrateController === controller) hydrateController = null;
}

onMounted(() => {
  if (!input.value) return;
  tagify = new Tagify<KeywordTag>(input.value, {
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
      classname: "nestrr-keyword-suggestions",
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
  hydrateController = null;
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
