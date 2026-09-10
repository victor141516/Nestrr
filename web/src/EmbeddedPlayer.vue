<script setup lang="ts">
import { onBeforeUnmount, onMounted, useTemplateRef } from "vue";
import AppIcon from "./AppIcon.vue";
import { isPlarrEvent, type PlarrEvent } from "./playerProtocol";

const props = defineProps<{
  url: string;
  origin: string;
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
      :key="url"
      ref="frame"
      class="player-frame"
      :src="url"
      allow="autoplay; fullscreen; picture-in-picture"
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
