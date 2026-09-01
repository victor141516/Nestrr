<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import {
  useBulkListImport,
  type BulkImportEvent,
  type BulkImportStatus,
} from "./useBulkListImport";

const props = defineProps<{
  listId: number;
  translate: (key: TranslationKey) => string;
}>();
const emit = defineEmits<{
  close: [];
  imported: [event: BulkImportEvent];
  finished: [];
}>();

const {
  canSubmit,
  completed,
  entries,
  error,
  running,
  text,
  unresolved,
  submit,
} = useBulkListImport({
  listId: () => props.listId,
  onImported: (event) => emit("imported", event),
  onFinished: () => emit("finished"),
});

const statusKeys: Record<BulkImportStatus, TranslationKey> = {
  queued: "queued",
  database_search: "searchingDatabase",
  database_match: "foundInDatabase",
  database_miss: "notFoundInDatabase",
  tmdb_search: "searchingTmdb",
  tmdb_match: "foundInTmdb",
  saved: "addedToList",
  not_found: "notFound",
  error: "importFailed",
  complete: "complete",
};
const active = computed(() => running.value || entries.value.length > 0);
const lineCount = computed(
  () => text.value.split(/\r?\n/).filter((line) => line.trim()).length,
);

function statusLabel(event: BulkImportEvent) {
  if (event.status === "saved" && !event.added)
    return props.translate("alreadyInList");
  return props.translate(statusKeys[event.status]);
}

function statusIcon(status: BulkImportStatus) {
  if (
    status === "queued" ||
    status === "database_search" ||
    status === "tmdb_search"
  )
    return "loader";
  if (status === "not_found" || status === "error") return "alert";
  return "check";
}
</script>

<template>
  <section class="bulk-list-import" :aria-busy="running">
    <div class="bulk-import-heading">
      <h2>{{ translate("bulkAdd") }}</h2>
      <button
        class="bulk-import-close"
        :aria-label="translate('close')"
        :title="translate('close')"
        @click="emit('close')"
      >
        <AppIcon name="x" />
      </button>
    </div>
    <label class="bulk-import-field">
      <span>{{ translate("titles") }}</span>
      <textarea
        v-model="text"
        :placeholder="translate('oneTitlePerLine')"
        :disabled="running"
        rows="7"
        maxlength="65536"
      />
    </label>
    <div class="bulk-import-actions">
      <span class="bulk-import-count">{{ lineCount }}/500</span>
      <button class="primary" :disabled="!canSubmit" @click="submit">
        <AppIcon
          :name="running ? 'loader' : 'plus'"
          :class="{ spinning: running }"
        />
        {{ translate(running ? "working" : "bulkAdd") }}
      </button>
    </div>

    <div v-if="active" class="bulk-import-results" aria-live="polite">
      <div
        v-for="entry in entries"
        :key="entry.index"
        :class="['bulk-import-result', entry.status]"
      >
        <span class="bulk-import-result-title">{{ entry.input }}</span>
        <span class="bulk-import-result-status">
          <AppIcon
            :name="statusIcon(entry.status)"
            :class="{
              spinning: ['queued', 'database_search', 'tmdb_search'].includes(
                entry.status,
              ),
            }"
          />
          {{ statusLabel(entry) }}
        </span>
      </div>
    </div>
    <p v-if="error" class="bulk-import-error" role="alert">{{ error }}</p>
    <div v-if="completed && unresolved.length" class="bulk-import-unresolved">
      <b>{{ translate("notFound") }}</b>
      <span v-for="entry in unresolved" :key="entry.index">{{
        entry.input
      }}</span>
    </div>
  </section>
</template>

<style scoped>
.bulk-list-import {
  margin: 1rem 0 1.25rem;
  padding: 1rem;
  border: 1px solid var(--line);
  border-radius: 14px;
  background: var(--panel);
}
.bulk-import-heading,
.bulk-import-actions,
.bulk-import-result {
  display: flex;
  align-items: center;
}
.bulk-import-heading {
  justify-content: space-between;
  margin-bottom: 0.8rem;
}
.bulk-import-heading h2 {
  margin: 0;
  font-size: 1rem;
}
.bulk-import-close {
  width: 42px;
  min-height: 42px;
  padding: 0;
}
.bulk-import-field {
  display: grid;
  gap: 0.4rem;
  color: var(--muted);
  font-size: 0.78rem;
}
.bulk-import-field textarea {
  width: 100%;
  min-height: 148px;
  resize: vertical;
  line-height: 1.45;
}
.bulk-import-actions {
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.75rem;
}
.bulk-import-count {
  color: var(--muted);
  font-size: 0.78rem;
}
.bulk-import-results {
  display: grid;
  gap: 0.35rem;
  max-height: 300px;
  overflow: auto;
  margin-top: 0.9rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--line);
}
.bulk-import-result {
  justify-content: space-between;
  gap: 1rem;
  min-height: 34px;
  padding: 0.35rem 0.5rem;
  border-radius: 8px;
  background: var(--control);
  font-size: 0.82rem;
}
.bulk-import-result-title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bulk-import-result-status {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 0.35rem;
  color: var(--muted);
}
.bulk-import-result.saved .bulk-import-result-status,
.bulk-import-result.database_match .bulk-import-result-status,
.bulk-import-result.tmdb_match .bulk-import-result-status {
  color: #5fd3c2;
}
.bulk-import-result.not_found .bulk-import-result-status,
.bulk-import-result.error .bulk-import-result-status,
.bulk-import-error {
  color: #ff8b91;
}
.bulk-import-error {
  margin: 0.75rem 0 0;
}
.bulk-import-unresolved {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 0.75rem;
  margin-top: 0.8rem;
  color: var(--muted);
  font-size: 0.82rem;
}
.bulk-import-unresolved b {
  width: 100%;
  color: var(--text);
}
.spinning {
  animation: bulk-spin 0.8s linear infinite;
}
@keyframes bulk-spin {
  to {
    transform: rotate(360deg);
  }
}
@media (max-width: 600px) {
  .bulk-list-import {
    padding: 0.8rem;
  }
  .bulk-import-result {
    align-items: flex-start;
    flex-direction: column;
    gap: 0.2rem;
  }
}
</style>
