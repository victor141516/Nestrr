<script setup lang="ts">
import { computed } from "vue";
import AppIcon from "./AppIcon.vue";
import type { TranslationKey } from "./i18n";
import type { RecommendationRunStatus } from "./useAdministration";

const props = defineProps<{
  status: RecommendationRunStatus;
  translate: (key: TranslationKey) => string;
}>();

const emit = defineEmits<{ run: [] }>();

const discoveryPercent = computed(() => percent(props.status.discovery));
const hydrationPercent = computed(() => percent(props.status.hydration));

function percent(progress: { total: number; done: number }): number {
  if (progress.total <= 0) return 0;
  return Math.min(100, Math.round((progress.done / progress.total) * 100));
}
</script>

<template>
  <section class="panel recommendation-catalog-panel">
    <div class="recommendation-catalog-heading">
      <h2>{{ translate("recommendationCatalog") }}</h2>
      <button
        type="button"
        class="primary"
        :disabled="status.state === 'running'"
        @click="emit('run')"
      >
        <AppIcon
          :class="{ spin: status.state === 'running' }"
          :name="status.state === 'running' ? 'loader' : 'refresh'"
        />
        {{
          status.state === "running"
            ? translate("running")
            : translate("runNow")
        }}
      </button>
    </div>

    <div class="recommendation-facts">
      <p>
        <b>{{ status.catalogCount.toLocaleString() }}</b
        ><span>{{ translate("catalog") }}</span>
      </p>
      <p>
        <b>{{ status.usersScanned.toLocaleString() }}</b
        ><span>{{ translate("users") }}</span>
      </p>
      <p>
        <b>{{ status.pending.toLocaleString() }}</b
        ><span>{{ translate("pending") }}</span>
      </p>
    </div>

    <div class="recommendation-signals">
      <span
        >{{ translate("genres") }} <b>{{ status.signals.genres }}</b></span
      >
      <span
        >{{ translate("keywords") }} <b>{{ status.signals.keywords }}</b></span
      >
      <span
        >{{ translate("actors") }} <b>{{ status.signals.actors }}</b></span
      >
      <span
        >{{ translate("directors") }}
        <b>{{ status.signals.directors }}</b></span
      >
      <span
        >{{ translate("companies") }}
        <b>{{ status.signals.companies }}</b></span
      >
    </div>

    <div
      class="recommendation-progress-list"
      :aria-busy="status.state === 'running'"
    >
      <div>
        <span>{{ translate("discovery") }}</span>
        <b>{{ status.discovery.done }}/{{ status.discovery.total }}</b>
        <div class="recommendation-progress">
          <i :style="{ width: `${discoveryPercent}%` }" />
        </div>
        <small>
          {{ translate("found") }} {{ status.discovery.found || 0 }} ·
          {{ translate("newCandidates") }} {{ status.discovery.added || 0 }} ·
          {{ translate("failed") }} {{ status.discovery.failed }}
        </small>
      </div>
      <div>
        <span>{{ translate("hydration") }}</span>
        <b>{{ status.hydration.done }}/{{ status.hydration.total }}</b>
        <div class="recommendation-progress">
          <i :style="{ width: `${hydrationPercent}%` }" />
        </div>
        <small>
          {{ translate("pending") }}
          {{ Math.max(0, status.hydration.total - status.hydration.done) }} ·
          {{ translate("failed") }} {{ status.hydration.failed }}
        </small>
      </div>
    </div>

    <p v-if="status.lastError" class="inline-message error" role="alert">
      <AppIcon name="alert" />{{ status.lastError }}
    </p>
  </section>
</template>

<style scoped>
.recommendation-catalog-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.recommendation-catalog-heading h2 {
  margin: 0;
}

.recommendation-catalog-heading button {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  min-height: 40px;
}

.recommendation-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.75rem;
  margin-top: 1.25rem;
}

.recommendation-facts p {
  display: grid;
  gap: 0.15rem;
  margin: 0;
  padding: 0.9rem 1rem;
  border: 1px solid var(--line);
  border-radius: 12px;
  background: var(--panel-2);
}

.recommendation-facts b {
  font-size: 1.25rem;
}

.recommendation-facts span,
.recommendation-signals {
  color: var(--muted);
  font-size: 0.78rem;
}

.recommendation-signals {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem 1rem;
  margin-top: 1rem;
}

.recommendation-signals b {
  color: var(--text);
}

.recommendation-progress-list {
  display: grid;
  gap: 0.85rem;
  margin-top: 1.25rem;
}

.recommendation-progress-list > div {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.45rem 1rem;
  font-size: 0.85rem;
}

.recommendation-progress {
  grid-column: 1 / -1;
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--control);
}

.recommendation-progress i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--accent);
  transition: width 180ms ease;
}

.recommendation-progress-list small {
  grid-column: 1 / -1;
  color: var(--muted);
}

@media (max-width: 640px) {
  .recommendation-catalog-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .recommendation-catalog-heading button {
    justify-content: center;
  }

  .recommendation-facts {
    grid-template-columns: 1fr;
  }
}

@media (prefers-reduced-motion: reduce) {
  .recommendation-progress i {
    transition: none;
  }
}
</style>
