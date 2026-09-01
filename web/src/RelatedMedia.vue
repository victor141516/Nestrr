<script setup lang="ts">
import AppIcon from "./AppIcon.vue";
import MediaCard from "./MediaCard.vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";

defineProps<{ detail: any }>();
const { availability, startedFor, systemMembershipsFor, t } =
  useNestrrApplicationContext();
</script>

<template>
  <section v-if="detail.belongs_to_collection" class="section-block">
    <h2>{{ t("collection") }}</h2>
    <RouterLink
      class="collection-card"
      :to="`/collection/${detail.belongs_to_collection.id}`"
    >
      <AppIcon name="sparkles" />{{ detail.belongs_to_collection.name
      }}<AppIcon name="chevronRight" />
    </RouterLink>
  </section>
  <section v-if="detail.parts" class="section-block">
    <h2>{{ t("titles") }}</h2>
    <div class="media-grid">
      <MediaCard
        v-for="item in detail.parts"
        :key="item.id"
        :m="{ ...item, media_type: 'movie' }"
        :to="`/movie/${item.id}`"
        :availability="availability[`movie:${item.id}`]"
        :memberships="systemMembershipsFor('movie', item.id)"
        :started="startedFor('movie', item.id)"
      />
    </div>
  </section>
  <section v-if="detail._recommendations?.length" class="section-block">
    <h2>{{ t("related") }}</h2>
    <div class="media-grid">
      <MediaCard
        v-for="item in detail._recommendations"
        :key="item.id"
        :m="item"
        :to="`/${item.media_type}/${item.id}`"
        :availability="availability[`${item.media_type}:${item.id}`]"
        :memberships="systemMembershipsFor(item.media_type, item.id)"
        :started="startedFor(item.media_type, item.id)"
      />
    </div>
  </section>
</template>
