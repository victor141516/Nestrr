<script setup lang="ts">
import { computed } from "vue";
import { artwork, session } from "./api";
import { tr } from "./i18n";
import type { ListMembership, MediaList, MediaListItem } from "./mediaLists";
import AppIcon from "./AppIcon.vue";

const props = withDefaults(
  defineProps<{
    item: MediaListItem;
    lists: MediaList[];
    memberships: ListMembership[];
    pending?: boolean;
    busyListId?: number;
  }>(),
  { pending: false, busyListId: 0 },
);
const emit = defineEmits<{
  (event: "toggle-list", list: MediaList): void;
  (event: "new-list"): void;
}>();

const t = (key: any) => tr(session.user?.language, key);
const systemLists = computed(() =>
  props.lists.filter((list) => Boolean(list.systemKey)),
);
const customLists = computed(() =>
  props.lists.filter((list) => !list.systemKey),
);
const addedDate = computed(() => {
  if (!props.item.addedAt) return "";
  const date = new Date(props.item.addedAt);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat(session.user?.language || undefined, {
    dateStyle: "medium",
  }).format(date);
});

function listed(list: MediaList) {
  return props.memberships.some((membership) => membership.listId === list.id);
}
</script>

<template>
  <article :class="['list-media-row', { 'pending-removal': pending }]">
    <RouterLink
      class="list-row-poster"
      :to="`/${item.mediaType}/${item.tmdbId}`"
      :aria-label="item.title"
    >
      <img
        v-if="item.posterPath"
        :src="artwork(item.posterPath)"
        :alt="item.title"
        loading="lazy"
      />
      <span v-else class="poster-placeholder">
        <AppIcon :name="item.mediaType === 'tv' ? 'tv' : 'film'" :size="28" />
      </span>
    </RouterLink>

    <div class="list-row-copy">
      <RouterLink
        class="list-row-title"
        :to="`/${item.mediaType}/${item.tmdbId}`"
      >
        {{ item.title }}
      </RouterLink>
      <p v-if="item.overview">{{ item.overview }}</p>
      <small v-if="addedDate" class="list-row-date">
        <AppIcon name="clock" :size="15" />{{ t("dateAdded") }} ·
        <time :datetime="item.addedAt">{{ addedDate }}</time>
      </small>
    </div>

    <div class="list-row-actions" :aria-label="t('listActions')">
      <button
        v-for="list in systemLists"
        :key="list.id"
        type="button"
        :class="['list-row-action', list.systemKey, { active: listed(list) }]"
        :aria-label="
          list.systemKey
            ? t(
                list.systemKey === 'watch-later'
                  ? 'watchLater'
                  : list.systemKey,
              )
            : list.name
        "
        :title="
          list.systemKey
            ? t(
                list.systemKey === 'watch-later'
                  ? 'watchLater'
                  : list.systemKey,
              )
            : list.name
        "
        :aria-pressed="listed(list)"
        :disabled="busyListId === list.id"
        @click="emit('toggle-list', list)"
      >
        <AppIcon :name="list.icon" />
      </button>

      <details class="custom-list-menu list-row-custom-menu">
        <summary
          :class="{ active: customLists.some((list) => listed(list)) }"
          :title="t('customLists')"
          :aria-label="t('customLists')"
        >
          <AppIcon name="list" />
        </summary>
        <div>
          <button
            v-for="list in customLists"
            :key="list.id"
            type="button"
            :class="{ selected: listed(list) }"
            :disabled="busyListId === list.id"
            @click="emit('toggle-list', list)"
          >
            <AppIcon :name="listed(list) ? 'check' : 'plus'" />{{ list.name }}
          </button>
          <button
            class="create-inline-list"
            type="button"
            @click="emit('new-list')"
          >
            <AppIcon name="plus" />{{ t("newList") }}
          </button>
        </div>
      </details>
    </div>
  </article>
</template>
