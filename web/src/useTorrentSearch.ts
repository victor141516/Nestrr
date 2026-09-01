import { computed, onScopeDispose, reactive } from "vue";
import { api } from "./api";
import {
  torrentCategoryOptions,
  torrentSearchDefaults,
  type TorrentCategories,
  type TorrentMediaKind,
  type TorrentSearchItem,
} from "./torrentDefaults";
import { filterTorrentResults, type TorrentResult } from "./torrentFilters";

type TorrentSearchResponse = { results?: TorrentResult[] };

export type TorrentAccepted = {
  item: TorrentSearchItem;
  kind: TorrentMediaKind;
  result: TorrentResult;
  category: string;
};

export type TorrentRequest = <T>(
  path: string,
  options?: RequestInit,
) => Promise<T>;

type UseTorrentSearchOptions = {
  item: TorrentSearchItem;
  kind: TorrentMediaKind;
  categories: TorrentCategories;
  failureMessage: string;
  request?: TorrentRequest;
};

function errorMessage(reason: unknown, fallback: string) {
  return reason instanceof Error && reason.message ? reason.message : fallback;
}

export function useTorrentSearch(options: UseTorrentSearchOptions) {
  const defaults = torrentSearchDefaults(options.item, options.kind);
  const request = options.request ?? api;
  const state = reactive({
    query: defaults.query,
    results: [] as TorrentResult[],
    loading: false,
    minSeeds: 0,
    trackers: [] as string[],
    include: defaults.include,
    exclude: [] as string[],
    submittingId: "",
    submittingCategory: "",
    submittedId: "",
    submittedCategory: "",
    submitError: "",
  });
  let active = true;
  let searchController: AbortController | undefined;

  const trackerOptions = computed(() =>
    [
      ...new Set(
        state.results
          .map((result) => result.Tracker)
          .filter((tracker): tracker is string => Boolean(tracker)),
      ),
    ].sort(),
  );
  const filteredResults = computed(() =>
    filterTorrentResults(state.results, {
      ...state,
      trackersActive: trackerOptions.value.length > 0,
    }),
  );
  const categoryOptions = computed(() =>
    torrentCategoryOptions(options.kind, options.categories),
  );

  async function search() {
    searchController?.abort();
    const controller = new AbortController();
    searchController = controller;
    state.loading = true;
    state.submitError = "";
    try {
      const response = await request<TorrentSearchResponse>(
        `/torrents/search?q=${encodeURIComponent(state.query)}`,
        { signal: controller.signal },
      );
      if (!active || controller.signal.aborted) return;
      state.results = Array.isArray(response.results) ? response.results : [];
      state.trackers = [...trackerOptions.value];
    } catch (reason: unknown) {
      if (!active || controller.signal.aborted) return;
      state.submitError = errorMessage(reason, options.failureMessage);
    } finally {
      if (active && searchController === controller) state.loading = false;
    }
  }

  async function submit(
    result: TorrentResult,
    category: string,
  ): Promise<TorrentAccepted | null> {
    if (!active || state.submittingId) return null;
    const id = String(result.ID);
    state.submittingId = id;
    state.submittingCategory = category;
    state.submitError = "";
    try {
      await request<void>("/torrents/submit", {
        method: "POST",
        body: JSON.stringify({
          URL: result.MagnetURI || result.Link,
          Category: category,
          MediaType: options.kind,
          TMDBID: options.item.id,
        }),
      });
      if (!active) return null;
      state.submittedId = id;
      state.submittedCategory = category;
      return {
        item: options.item,
        kind: options.kind,
        result,
        category,
      };
    } catch (reason: unknown) {
      if (active)
        state.submitError = errorMessage(reason, options.failureMessage);
      return null;
    } finally {
      if (active) {
        state.submittingId = "";
        state.submittingCategory = "";
      }
    }
  }

  function dispose() {
    active = false;
    searchController?.abort();
  }

  onScopeDispose(dispose);

  return {
    state,
    trackerOptions,
    filteredResults,
    categoryOptions,
    search,
    submit,
    dispose,
  };
}
