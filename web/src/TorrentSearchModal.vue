<script setup lang="ts">
import AppDialog from "./AppDialog.vue";
import AppIcon from "./AppIcon.vue";
import TorrentTagInput from "./TorrentTagInput.vue";
import TrackerMultiSelect from "./TrackerMultiSelect.vue";
import type { TranslationKey } from "./i18n";
import type {
  TorrentCategories,
  TorrentMediaKind,
  TorrentSearchItem,
} from "./torrentDefaults";
import type { TorrentResult } from "./torrentFilters";
import { useTorrentSearch, type TorrentAccepted } from "./useTorrentSearch";

const props = defineProps<{
  item: TorrentSearchItem;
  kind: TorrentMediaKind;
  categories: TorrentCategories;
  translate: (key: TranslationKey) => string;
}>();
const emit = defineEmits<{
  close: [];
  accepted: [payload: TorrentAccepted];
  notice: [message: string];
}>();

const {
  state,
  trackerOptions,
  filteredResults,
  categoryOptions,
  search,
  submit,
  dispose,
} = useTorrentSearch({
  item: props.item,
  kind: props.kind,
  categories: props.categories,
  failureMessage: props.translate("torrentFailed"),
});

const title = props.item.title || props.item.name || "";

function close() {
  dispose();
  emit("close");
}

function closeCategoryMenu(event?: MouseEvent) {
  const target = event?.currentTarget as HTMLElement | null;
  const menu = (target?.closest("details") ??
    target
      ?.closest(".torrent-send-split")
      ?.querySelector("details")) as HTMLDetailsElement | null;
  if (menu) menu.open = false;
}

async function submitResult(
  result: TorrentResult,
  category: string,
  event?: MouseEvent,
) {
  closeCategoryMenu(event);
  const accepted = await submit(result, category);
  if (accepted) {
    emit("notice", props.translate("torrentAccepted"));
    emit("accepted", accepted);
  } else if (state.submitError) {
    emit("notice", `${props.translate("torrentFailed")}: ${state.submitError}`);
  }
}
</script>

<template>
  <AppDialog
    panel-class="torrent-modal"
    labelledby="torrent-search-title"
    :busy="state.loading || Boolean(state.submittingId)"
    @close="close"
  >
    <button
      type="button"
      class="close"
      :aria-label="translate('close')"
      @click="close"
    >
      <AppIcon name="x" />
    </button>
    <h2 id="torrent-search-title">{{ translate("searchTorrent") }}</h2>
    <p class="modal-subtitle">{{ title }}</p>
    <form class="torrent-search" @submit.prevent="search">
      <div class="search-input">
        <AppIcon name="search" /><input
          v-model="state.query"
          data-dialog-initial-focus
        />
      </div>
      <button class="primary" :disabled="state.loading">
        {{ state.loading ? translate("searching") : translate("search") }}
      </button>
    </form>
    <p v-if="state.submitError" class="submit-status error" role="alert">
      <AppIcon name="alert" :size="16" />{{ state.submitError }}
    </p>
    <details class="torrent-filters">
      <summary>
        <AppIcon name="filter" :size="15" />{{ translate("filters") }}
      </summary>
      <div class="filters">
        <label class="torrent-filter-row torrent-seeds-field"
          >{{ translate("minSeeds")
          }}<input v-model.number="state.minSeeds" type="number" min="0"
        /></label>
        <TorrentTagInput
          id="torrent-include"
          v-model="state.include"
          :label="translate('includeWords')"
          :placeholder="translate('tagInputHint')"
        />
        <TorrentTagInput
          id="torrent-exclude"
          v-model="state.exclude"
          :label="translate('excludeWords')"
          :placeholder="translate('tagInputHint')"
        />
        <TrackerMultiSelect
          v-model="state.trackers"
          :label="translate('trackers')"
          :options="trackerOptions"
          :all-label="translate('allTrackers')"
          :none-label="translate('noTrackers')"
          :selected-label="translate('selectedTrackers')"
          :select-all-label="translate('selectAll')"
          :select-none-label="translate('clearAll')"
        />
      </div>
    </details>
    <div class="torrent-results">
      <article v-for="result in filteredResults" :key="result.ID">
        <div class="torrent-icon"><AppIcon name="download" /></div>
        <div>
          <b>{{ result.Title }}</b
          ><small
            >{{ result.Tracker }} · <AppIcon name="arrowUp" :size="13" />
            {{ result.Seeders }} {{ translate("seeds") }} ·
            {{ (Number(result.Size || 0) / 1073741824).toFixed(2) }} GB</small
          >
        </div>
        <div class="torrent-send-split">
          <button
            type="button"
            class="torrent-send-primary"
            :disabled="
              Boolean(state.submittingId) ||
              state.submittedId === String(result.ID)
            "
            @click="submitResult(result, categoryOptions.primary, $event)"
          >
            <AppIcon
              v-if="state.submittedId === String(result.ID)"
              name="check"
              :size="14"
            />{{
              state.submittingId === String(result.ID)
                ? `${translate("sendingTo")} ${state.submittingCategory}`
                : state.submittedId === String(result.ID)
                  ? `${translate("sentTo")} ${state.submittedCategory}`
                  : `${translate("sendTo")} ${categoryOptions.primary}`
            }}
          </button>
          <details
            class="torrent-send-more"
            :class="{
              disabled:
                Boolean(state.submittingId) ||
                state.submittedId === String(result.ID),
            }"
          >
            <summary
              :aria-label="translate('otherTorrentCategory')"
              :aria-disabled="
                Boolean(state.submittingId) ||
                state.submittedId === String(result.ID)
              "
              @click="
                (Boolean(state.submittingId) ||
                  state.submittedId === String(result.ID)) &&
                $event.preventDefault()
              "
            >
              <AppIcon name="chevronDown" :size="16" />
            </summary>
            <div class="torrent-send-popover">
              <button
                type="button"
                @click="
                  submitResult(result, categoryOptions.alternative, $event)
                "
              >
                {{ translate("sendTo") }} {{ categoryOptions.alternative }}
              </button>
            </div>
          </details>
        </div>
      </article>
    </div>
    <div v-if="state.loading" class="empty-state compact-empty">
      <span class="spinner" />
      <h3>{{ translate("searching") }}</h3>
    </div>
    <div
      v-else-if="!state.results.length && !state.submitError"
      class="empty-state compact-empty"
    >
      <AppIcon name="search" :size="28" />
      <h3>{{ translate("torrentPrompt") }}</h3>
    </div>
  </AppDialog>
</template>
