<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  onUnmounted,
  useTemplateRef,
} from "vue";
import { televisionPlayerURL } from "./television";
import AppIcon from "./AppIcon.vue";
import { isPlarrEvent, type PlarrEvent } from "./playerProtocol";

const props = defineProps<{
  url: string;
  origin: string;
  television?: boolean;
  autoplaySeconds?: number;
  labels: {
    upNext: string;
    nextEpisodeIn: string;
    playNow: string;
    cancel: string;
  };
}>();

const emit = defineEmits<{
  playerEvent: [event: PlarrEvent];
  playNext: [];
  cancelAutoplay: [];
}>();

const frame = useTemplateRef<HTMLIFrameElement>("frame");
const playerURL = computed(() =>
  props.television ? televisionPlayerURL(props.url) : props.url,
);
const opener = document.activeElement as HTMLElement | null;
function focusPlayer() {
  if (props.television) frame.value?.focus();
}
onUnmounted(() => {
  if (props.television)
    void nextTick(() => opener?.isConnected && opener.focus());
});

function onMessage(event: MessageEvent<unknown>): void {
  if (
    event.origin !== props.origin ||
    event.source !== frame.value?.contentWindow ||
    !isPlarrEvent(event.data)
  )
    return;
  emit("playerEvent", event.data);
}

onMounted(() => window.addEventListener("message", onMessage));
onBeforeUnmount(() => window.removeEventListener("message", onMessage));
</script>

<template>
  <div class="player-overlay">
    <iframe
      :key="playerURL"
      ref="frame"
      class="player-frame"
      :src="playerURL"
      title="Plarr"
      @load="focusPlayer"
      allow="autoplay; fullscreen; picture-in-picture; clipboard-write"
      allowfullscreen
    />
    <div v-if="autoplaySeconds !== undefined" class="autoplay">
      <p class="eyebrow">{{ labels.upNext }}</p>
      <b>{{ labels.nextEpisodeIn }} {{ autoplaySeconds }}s</b>
      <div>
        <button class="primary" @click="emit('playNext')">
          <AppIcon name="play" />{{ labels.playNow }}</button
        ><button @click="emit('cancelAutoplay')">{{ labels.cancel }}</button>
      </div>
    </div>
  </div>
</template>
