<script setup lang="ts">
import MediaDetailHero from "./MediaDetailHero.vue";
import RelatedMedia from "./RelatedMedia.vue";
import SeasonList from "./SeasonList.vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";

const { detail } = useNestrrApplicationContext();
</script>

<template>
  <section v-if="detail" class="detail-page">
    <MediaDetailHero :detail="detail" />
    <div class="detail-body">
      <SeasonList :detail="detail" />
      <RelatedMedia :detail="detail" />
    </div>
  </section>
</template>
