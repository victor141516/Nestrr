<script setup lang="ts">
import AutocompleteTagInput from "./AutocompleteTagInput.vue";
import type { PersonOption } from "./movieFilters";
import { mergePeople, resolvePeople, searchPeople } from "./personCatalog";

defineProps<{
  id: string;
  label: string;
  modelValue: PersonOption[];
  placeholder: string;
}>();
defineEmits<{
  "update:modelValue": [value: PersonOption[]];
}>();

async function search(
  query: string,
  signal: AbortSignal,
  publish: (options: PersonOption[]) => void,
) {
  const local = await searchPeople(query, "local", signal);
  publish(local);
  const remote = await searchPeople(query, "remote", signal);
  publish(mergePeople(local, remote));
}
</script>

<template>
  <AutocompleteTagInput
    :id="id"
    :model-value="modelValue"
    :label="label"
    :placeholder="placeholder"
    :search="search"
    :resolve="resolvePeople"
    dropdown-class="nestrr-person-suggestions"
    @update:model-value="$emit('update:modelValue', $event)"
  />
</template>
