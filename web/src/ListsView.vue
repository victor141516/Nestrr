<script setup lang="ts">
import { shallowRef, watch } from "vue";
import { useNestrrApplicationContext } from "./useNestrrApplication";
import AppIcon from "./AppIcon.vue";
import BulkListImport from "./BulkListImport.vue";
import ListToolbar from "./ListToolbar.vue";
import ListMediaRow from "./ListMediaRow.vue";
import type { BulkImportEvent } from "./useBulkListImport";

const {
  acceptBulkImportedItem,
  activeList,
  activeListItems,
  busy,
  busyListFor,
  detail,
  finishBulkImport,
  listDirection,
  listEditor,
  listMediaType,
  listSort,
  mediaLists,
  membershipsFor,
  pathOnly,
  pendingRemovalFor,
  systemListName,
  setListMediaType,
  setListSort,
  t,
  title,
  toggleListDirection,
  toggleRowList,
} = useNestrrApplicationContext();

const bulkImportOpen = shallowRef(false);

watch(
  () => activeList.value?.id,
  () => (bulkImportOpen.value = false),
);

function onBulkImported(event: BulkImportEvent) {
  if (!activeList.value || !event.item) return;
  acceptBulkImportedItem(activeList.value.id, event.item, Boolean(event.added));
}
</script>

<template>
  <section class="content page lists-page">
    <template v-if="pathOnly === '/lists'">
      <div class="lists-heading">
        <button
          class="primary"
          @click="listEditor = { mode: 'create', name: '' }"
        >
          <AppIcon name="plus" />{{ t("newList") }}
        </button>
      </div>
      <div class="list-overview-grid">
        <RouterLink
          v-for="list in mediaLists"
          :key="list.id"
          :to="`/lists/${list.id}`"
          :class="['list-overview-card', list.color]"
        >
          <span class="list-overview-icon"
            ><AppIcon :name="list.icon" :size="25"
          /></span>
          <div>
            <b>{{
              list.systemKey ? systemListName(list.systemKey) : list.name
            }}</b
            ><small>{{ list.itemCount }} {{ t("titles") }}</small>
          </div>
          <AppIcon name="chevronRight" />
        </RouterLink>
      </div>
    </template>
    <template v-else-if="activeList">
      <div class="list-detail-heading">
        <RouterLink class="back-link" to="/lists"
          ><AppIcon name="chevronRight" />{{ t("allLists") }}</RouterLink
        >
        <div class="section-heading">
          <div>
            <h1>
              {{
                activeList.systemKey
                  ? systemListName(activeList.systemKey)
                  : activeList.name
              }}
            </h1>
            <p>{{ activeList.itemCount }} {{ t("titles") }}</p>
          </div>
          <div class="list-detail-actions">
            <button
              :class="{ primary: bulkImportOpen }"
              :aria-expanded="bulkImportOpen"
              @click="bulkImportOpen = !bulkImportOpen"
            >
              <AppIcon name="plus" />{{ t("bulkAdd") }}
            </button>
            <div v-if="!activeList.systemKey" class="list-management-actions">
              <button
                :aria-label="t('rename')"
                :title="t('rename')"
                @click="
                  listEditor = {
                    mode: 'rename',
                    list: activeList,
                    name: activeList.name,
                  }
                "
              >
                <AppIcon name="pencil" />
              </button>
              <button
                class="danger-outline"
                :aria-label="t('delete')"
                :title="t('delete')"
                @click="
                  listEditor = {
                    mode: 'delete',
                    list: activeList,
                    name: activeList.name,
                  }
                "
              >
                <AppIcon name="trash" />
              </button>
            </div>
          </div>
        </div>
      </div>
      <BulkListImport
        v-if="bulkImportOpen"
        :list-id="activeList.id"
        :translate="t"
        @close="bulkImportOpen = false"
        @imported="onBulkImported"
        @finished="finishBulkImport"
      />
      <ListToolbar
        :media-type="listMediaType"
        :sort="listSort"
        :direction="listDirection"
        :translate="t"
        @select-media-type="setListMediaType"
        @select-sort="setListSort"
        @toggle-direction="toggleListDirection"
      />
      <div v-if="activeListItems.length" class="list-media-rows">
        <ListMediaRow
          v-for="item in activeListItems"
          :key="`${item.mediaType}:${item.tmdbId}`"
          :item="item"
          :lists="mediaLists"
          :memberships="membershipsFor(item.mediaType, item.tmdbId)"
          :pending="pendingRemovalFor(item)"
          :busy-list-id="busyListFor(item)"
          @toggle-list="toggleRowList($event, item)"
          @new-list="listEditor = { mode: 'create', name: '' }"
        />
      </div>
      <div v-else class="empty-state">
        <AppIcon :name="activeList.icon" :size="32" />
        <h2>{{ t("emptyList") }}</h2>
      </div>
    </template>
  </section>
</template>
