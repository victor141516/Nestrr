<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import {
  isTorrentDownloaded,
  isTorrentDownloading,
  isTorrentForceable,
  torrentPercent,
  torrentStateName,
  type TorrentDownload,
} from "./torrentState";

const props = defineProps<{
  torrent: TorrentDownload;
  completedLabel: string;
  forceLabel: string;
  forcing?: boolean;
}>();
defineEmits<{ forceStart: [] }>();

const percent = computed(() => torrentPercent(props.torrent.progress));
const stateName = computed(() => torrentStateName(props.torrent.state));
const downloaded = computed(() => isTorrentDownloaded(props.torrent.state));
const active = computed(() => isTorrentDownloading(props.torrent.state));
const forceable = computed(() => isTorrentForceable(props.torrent.state));
</script>

<template>
  <div class="torrent-download-status">
    <div
      v-if="downloaded"
      class="torrent-organization-pending"
      role="status"
      :aria-label="`${completedLabel}. ${stateName}`"
      :title="stateName"
    >
      <AppIcon name="checked" />
      <span>{{ completedLabel }}</span>
    </div>
    <div
      v-else
      :class="['torrent-progress-bar', { active, inactive: !active }]"
      role="progressbar"
      :aria-label="`${stateName}: ${percent}%`"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-valuenow="percent"
      :title="stateName"
    >
      <i :style="{ width: `${percent}%` }" />
      <span>{{ percent }}%</span>
    </div>
    <button
      v-if="!downloaded && forceable"
      type="button"
      class="torrent-force-button"
      :aria-label="forceLabel"
      :title="forceLabel"
      :disabled="forcing"
      @click="$emit('forceStart')"
    >
      <AppIcon
        :name="forcing ? 'loader' : 'force'"
        :class="{ spin: forcing }"
      />
    </button>
  </div>
</template>
