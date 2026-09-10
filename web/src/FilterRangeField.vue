<script setup lang="ts">
import AppIcon from "./AppIcon.vue";
import DualRangeSlider from "./DualRangeSlider.vue";

const props = withDefaults(
  defineProps<{
    icon: string;
    title: string;
    minValue: number | null;
    maxValue: number | null;
    floor: number;
    ceiling: number;
    step?: number;
    minLabel: string;
    maxLabel: string;
    formatValue?: (value: number) => string;
  }>(),
  { step: 1, formatValue: (value: number) => String(value) },
);
const emit = defineEmits<{
  "update:minValue": [value: number];
  "update:maxValue": [value: number];
}>();
</script>

<template>
  <fieldset class="filter-card range-card">
    <legend>
      <AppIcon :name="icon" /><b>{{ title }}</b>
    </legend>
    <DualRangeSlider
      :min-value="minValue"
      :max-value="maxValue"
      :floor="floor"
      :ceiling="ceiling"
      :step="step"
      :min-label="minLabel"
      :max-label="maxLabel"
      :format-value="props.formatValue"
      @update:min-value="emit('update:minValue', $event)"
      @update:max-value="emit('update:maxValue', $event)"
    />
  </fieldset>
</template>
