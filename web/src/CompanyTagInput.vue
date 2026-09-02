<script setup lang="ts">
import AutocompleteTagInput from "./AutocompleteTagInput.vue";
import {
  mergeCompanies,
  resolveCompanies,
  searchCompanies,
} from "./companyCatalog";
import type { FilterOption } from "./movieFilters";

defineProps<{
  id: string;
  label: string;
  modelValue: FilterOption[];
  placeholder: string;
}>();
defineEmits<{
  "update:modelValue": [value: FilterOption[]];
}>();

async function search(
  query: string,
  signal: AbortSignal,
  publish: (options: FilterOption[]) => void,
) {
  const local = await searchCompanies(query, "local", signal);
  publish(local);
  const remote = await searchCompanies(query, "remote", signal);
  publish(mergeCompanies(local, remote));
}
</script>

<template>
  <AutocompleteTagInput
    :id="id"
    :model-value="modelValue"
    :label="label"
    :placeholder="placeholder"
    :search="search"
    :resolve="resolveCompanies"
    dropdown-class="nestrr-company-suggestions"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>
