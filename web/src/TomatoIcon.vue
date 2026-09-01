<script setup lang="ts">
withDefaults(defineProps<{ size?: number; strokeWidth?: number }>(), {
  size: 18,
  strokeWidth: 2,
});
</script>

<template>
  <svg
    aria-hidden="true"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path
      d="M12 8c-5 0-8 2.7-8 6.2C4 18 7.4 21 12 21s8-3 8-6.8C20 10.7 17 8 12 8Z"
    />
    <path d="m12 8-1.8-3 3.1 1.1L16 4.5l-.7 3.2L18 9" />
    <path d="M12 8V4" />
  </svg>
</template>
