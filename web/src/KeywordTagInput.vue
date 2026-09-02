<script setup lang="ts">
import AutocompleteTagInput from "./AutocompleteTagInput.vue";
import {
  mergeKeywordSuggestions,
  resolveKeywords,
  searchKeywords,
} from "./keywordCatalog";
import type { KeywordOption } from "./movieFilters";

defineProps<{
  id: string;
  label: string;
  modelValue: KeywordOption[];
  placeholder: string;
}>();
defineEmits<{
  "update:modelValue": [value: KeywordOption[]];
}>();

async function search(
  query: string,
  signal: AbortSignal,
  publish: (options: KeywordOption[]) => void,
) {
  const prefix = await searchKeywords(query, "prefix", signal);
  publish(prefix);
  const contains = await searchKeywords(query, "contains", signal);
  publish(mergeKeywordSuggestions(prefix, contains));
}
</script>

<template>
  <AutocompleteTagInput
    :id="id"
    :model-value="modelValue"
    :label="label"
    :placeholder="placeholder"
    :search="search"
    :resolve="resolveKeywords"
    dropdown-class="nestrr-keyword-suggestions"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>
