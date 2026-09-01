<script setup lang="ts">
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AppIcon from "./AppIcon.vue";

const { t } = useNestrrApplicationContext();
</script>

<template>
  <section class="content page not-found">
    <span>404</span>
    <h1>{{ t("pageNotFound") }}</h1>
    <p>{{ t("pageNotFoundHelp") }}</p>
    <RouterLink class="primary" to="/"
      ><AppIcon name="compass" />{{ t("explore") }}</RouterLink
    >
  </section>
</template>
