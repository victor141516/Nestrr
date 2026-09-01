<script setup lang="ts">
import { computed, ref } from "vue";
import AppIcon from "./AppIcon.vue";
import { session } from "./api";
import { tr } from "./i18n";
import type { CountryOption, ProductionCountryOption } from "./movieFilters";

const props = defineProps<{
  modelValue: string[];
  options: ProductionCountryOption[];
  countries: CountryOption[];
}>();
const emit = defineEmits<{ "update:modelValue": [value: string[]] }>();
const open = ref(false);
const root = ref<HTMLElement | null>(null);
const t = (key: any) => tr(session.user?.language, key);
const locale = computed(() => session.user?.language || "en-US");
const displayNames = computed(
  () => new Intl.DisplayNames([locale.value], { type: "region" }),
);

function label(option: ProductionCountryOption) {
  if (option.countryCode)
    return displayNames.value.of(option.countryCode) || option.countryCode;
  return t(option.id.slice("region:".length) as any);
}
function toggle(value: string) {
  emit(
    "update:modelValue",
    props.modelValue.includes(value)
      ? props.modelValue.filter((item) => item !== value)
      : [...props.modelValue, value],
  );
}
function onFocusOut(event: FocusEvent) {
  if (!root.value?.contains(event.relatedTarget as Node | null))
    open.value = false;
}
const summary = computed(() => {
  if (!props.modelValue.length) return t("anyCountry");
  const selected = props.options
    .filter((option) => props.modelValue.includes(option.value))
    .map(label);
  return selected.length <= 2
    ? selected.join(", ")
    : `${selected.slice(0, 2).join(", ")} +${selected.length - 2}`;
});
</script>

<template>
  <div
    ref="root"
    class="country-multiselect"
    @focusout="onFocusOut"
    @keydown.esc="open = false"
  >
    <button
      type="button"
      class="country-select-trigger"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span>{{ summary }}</span
      ><AppIcon name="chevronDown" :size="16" />
    </button>
    <div
      v-if="open"
      class="country-select-menu"
      role="group"
      :aria-label="t('productionCountry')"
    >
      <button
        v-for="option in options"
        :key="option.id"
        type="button"
        :class="{ selected: modelValue.includes(option.value) }"
        :aria-pressed="modelValue.includes(option.value)"
        @click="toggle(option.value)"
      >
        <span class="custom-checkbox"
          ><AppIcon
            v-if="modelValue.includes(option.value)"
            name="check"
            :size="14" /></span
        ><span>{{ label(option) }}</span>
      </button>
    </div>
  </div>
</template>
