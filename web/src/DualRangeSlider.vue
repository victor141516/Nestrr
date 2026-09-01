<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
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

const lower = computed(() => props.minValue ?? props.floor);
const upper = computed(() => props.maxValue ?? props.ceiling);
const span = computed(() => Math.max(1, props.ceiling - props.floor));
const trackStyle = computed(() => ({
  "--slider-start": `${(100 * (lower.value - props.floor)) / span.value}%`,
  "--slider-end": `${(100 * (upper.value - props.floor)) / span.value}%`,
}));
const selectedRange = computed(
  () => `${props.formatValue(lower.value)} – ${props.formatValue(upper.value)}`,
);

function updateLower(event: Event) {
  const value = Number((event.target as HTMLInputElement).value);
  emit("update:minValue", Math.min(value, upper.value));
}

function updateUpper(event: Event) {
  const value = Number((event.target as HTMLInputElement).value);
  emit("update:maxValue", Math.max(value, lower.value));
}
</script>

<template>
  <div class="dual-range" :style="trackStyle">
    <div class="dual-range-track">
      <input
        class="dual-range-input lower"
        type="range"
        :min="floor"
        :max="ceiling"
        :step="step"
        :value="lower"
        :aria-label="minLabel"
        :aria-valuetext="formatValue(lower)"
        @input="updateLower"
      />
      <input
        class="dual-range-input upper"
        type="range"
        :min="floor"
        :max="ceiling"
        :step="step"
        :value="upper"
        :aria-label="maxLabel"
        :aria-valuetext="formatValue(upper)"
        @input="updateUpper"
      />
    </div>
    <div class="dual-range-value" aria-live="polite">{{ selectedRange }}</div>
  </div>
</template>
