import { computed, onBeforeUnmount, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "./api";
import {
  mediaTypeFromQuery,
  mediaTypeQueryValue,
  type CatalogMediaType,
} from "./mediaType";

export type RecommendationMediaType = CatalogMediaType;

export type RecommendationItem = {
  id: number;
  media_type: RecommendationMediaType;
  title?: string;
  name?: string;
  original_title?: string;
  original_name?: string;
  overview: string;
  poster_path: string;
  backdrop_path: string;
  release_date?: string;
  first_air_date?: string;
  vote_average: number;
  vote_count: number;
  popularity: number;
  genre_ids: number[];
  recommendation_score: number;
};

type RecommendationResponse = {
  results: RecommendationItem[];
  total_results: number;
  media_type: RecommendationMediaType;
};

export function useRecommendations() {
  const route = useRoute();
  const router = useRouter();
  const mediaType = computed<RecommendationMediaType>(() =>
    mediaTypeFromQuery(route.query.type),
  );
  const items = shallowRef<RecommendationItem[]>([]);
  const loading = shallowRef(false);
  const error = shallowRef("");
  let controller: AbortController | null = null;

  async function load(): Promise<void> {
    controller?.abort();
    const request = new AbortController();
    const requestedType = mediaType.value;
    controller = request;
    loading.value = true;
    error.value = "";
    try {
      const response = await api<RecommendationResponse>(
        `/recommendations?kind=${requestedType}&limit=40`,
        { signal: request.signal },
      );
      if (controller !== request || mediaType.value !== requestedType) return;
      items.value = response.results;
    } catch (reason) {
      if (request.signal.aborted || controller !== request) return;
      error.value = reason instanceof Error ? reason.message : String(reason);
    } finally {
      if (controller === request) loading.value = false;
    }
  }

  async function selectMediaType(
    value: RecommendationMediaType,
  ): Promise<void> {
    await router.push({
      query: { ...route.query, type: mediaTypeQueryValue(value) },
    });
  }

  watch(mediaType, () => void load(), { immediate: true });
  onBeforeUnmount(() => controller?.abort());

  return { error, items, load, loading, mediaType, selectMediaType };
}
