import { computed, ref, shallowRef, type ComputedRef, type Ref } from "vue";
import type { RouteLocationNormalizedLoaded, Router } from "vue-router";
import { api, session } from "./api";
import {
  downloadedFiltersNeedDetails,
  downloadedMediaFromLibrary,
  matchesDownloadedMedia,
  sortDownloadedMedia,
  type DownloadedExploreMedia,
  type DownloadedLibraryItem,
  type ExploreSort,
} from "./downloadedExplore";
import { exploreQuery, parseExploreQuery } from "./discoverState";
import type { TranslationKey } from "./i18n";
import { resolveKeywords } from "./keywordCatalog";
import {
  activeMovieFilterCount,
  cloneMovieFilters,
  createMovieFilters,
  discoverParams,
  matchesRottenTomatoes,
  matchesAge,
  normalizeMovieFilters,
  productionCountryOptions,
  usAgeFromDetails,
  type CountryOption,
  type FilterOption,
  type MovieFilters,
} from "./movieFilters";
import { useLatestRequest } from "./useLatestRequest";

export type ExploreMedia = {
  id: number;
  title?: string;
  name?: string;
  overview?: string;
  poster_path?: string;
  backdrop_path?: string;
  release_date?: string;
  first_air_date?: string;
  media_type?: string;
  vote_average?: number;
  vote_count?: number;
  popularity?: number;
  genre_ids?: number[];
  rotten_tomatoes_score?: number;
};

type ExploreAvailability = {
  available?: boolean;
  state?: "complete" | "partial" | "none" | "downloading";
};

type ExploreResultSet = {
  items: ExploreMedia[];
  page: number;
  totalPages: number;
  totalResults: number;
  availability?: Array<[string, ExploreAvailability]>;
};

const RESULT_FADE_MS = 100;
const RESULT_CACHE_LIMIT = 12;

type UseExploreOptions = {
  router: Router;
  currentRoute: RouteLocationNormalizedLoaded;
  pathOnly: ComputedRef<string>;
  busy: Ref<boolean>;
  error: Ref<string>;
  availability: Record<string, ExploreAvailability>;
  checkCards: (media: ExploreMedia[]) => Promise<void>;
  rottenTomatoesEnabled: Ref<boolean>;
  t: (key: TranslationKey) => string;
  title: (media: ExploreMedia) => string;
};

export function useExplore(options: UseExploreOptions) {
  const {
    router,
    currentRoute,
    pathOnly,
    busy,
    error,
    availability,
    checkCards,
    rottenTomatoesEnabled,
    t,
    title,
  } = options;
  const loadingMore = ref(false);
  const query = ref("");
  const searchType = ref<"movie" | "tv">("movie");
  const downloadedOnly = ref(false);
  const searchStarted = ref(false);
  const items = shallowRef<ExploreMedia[]>([]);
  const resultsTransitioning = shallowRef(false);
  const exploreSort = ref<ExploreSort>("popularity");
  const filterPanelOpen = ref(false);
  const filterOptionsLoading = ref(false);
  const movieGenres = ref<FilterOption[]>([]);
  const movieCountries = ref<CountryOption[]>([]);
  const filterGenreKind = ref<"movie" | "tv" | null>(null);
  const movieFilterDraft = ref<MovieFilters>(createMovieFilters());
  const appliedMovieFilters = ref<MovieFilters | null>(null);
  const searchPage = ref(1);
  const searchTotalResults = ref(0);
  const searchTotalPages = ref(0);
  const searchMode = ref<"idle" | "text" | "filters">("idle");
  const exploreRequests = useLatestRequest();
  const filterOptionRequests = useLatestRequest();
  const resultsCache = new Map<string, ExploreResultSet>();

  const visibleItems = computed(() => items.value);
  const movieFilterCount = computed(() =>
    activeMovieFilterCount(appliedMovieFilters.value),
  );
  const hasMoreSearchResults = computed(
    () => searchPage.value < searchTotalPages.value,
  );
  const genreName = (id: number) =>
    movieGenres.value.find((genre) => genre.id === id)?.name || String(id);
  const countryName = (value: string) => {
    const option = productionCountryOptions(movieCountries.value).find(
      (item) => item.value === value,
    );
    if (option?.countryCode)
      return (
        new Intl.DisplayNames([session.user?.language || "en-US"], {
          type: "region",
        }).of(option.countryCode) || option.countryCode
      );
    return option
      ? t(option.id.slice("region:".length) as TranslationKey)
      : value;
  };
  const appliedFilterChips = computed(() => {
    const filters = appliedMovieFilters.value;
    if (!filters) return [] as { group: string; label: string }[];
    const chips: { group: string; label: string }[] = [];
    if (filters.yearMin !== null || filters.yearMax !== null)
      chips.push({
        group: "year",
        label: `${t("releaseYear")}: ${filters.yearMin ?? 1870}–${filters.yearMax ?? new Date().getFullYear() + 5}`,
      });
    if (filters.ratingMin !== null || filters.ratingMax !== null)
      chips.push({
        group: "rating",
        label: `TMDB: ${filters.ratingMin ?? 0}–${filters.ratingMax ?? 10}`,
      });
    if (
      rottenTomatoesEnabled.value &&
      (filters.rottenTomatoesMin !== null || filters.rottenTomatoesMax !== null)
    )
      chips.push({
        group: "rottenTomatoes",
        label: `${t("rottenTomatoes")}: ${filters.rottenTomatoesMin ?? 0}–${filters.rottenTomatoesMax ?? 100}%`,
      });
    if (
      searchType.value === "movie" &&
      (filters.runtimeMin !== null || filters.runtimeMax !== null)
    )
      chips.push({
        group: "runtime",
        label: `${t("runtime")}: ${filters.runtimeMin ?? 0}–${filters.runtimeMax ?? 600} min`,
      });
    if (filters.ageMin !== null || filters.ageMax !== null)
      chips.push({
        group: "age",
        label: `${t("ageRating")}: ${filters.ageMin ?? 0}–${filters.ageMax ?? 18}`,
      });
    if (filters.includeGenres.length)
      chips.push({
        group: "includeGenres",
        label: `+ ${filters.includeGenres.map(genreName).join(", ")}`,
      });
    if (filters.excludeGenres.length)
      chips.push({
        group: "excludeGenres",
        label: `− ${filters.excludeGenres.map(genreName).join(", ")}`,
      });
    if (filters.includeKeywords.length)
      chips.push({
        group: "includeKeywords",
        label: `+ ${filters.includeKeywords.map((keyword) => keyword.name).join(", ")}`,
      });
    if (filters.excludeKeywords.length)
      chips.push({
        group: "excludeKeywords",
        label: `− ${filters.excludeKeywords.map((keyword) => keyword.name).join(", ")}`,
      });
    if (filters.includeCompanies.length)
      chips.push({
        group: "company",
        label: `${t("productionCompanies")}: ${filters.includeCompanies.map((company) => company.name).join(", ")}`,
      });
    if (filters.productionCountries.length)
      chips.push({
        group: "country",
        label: `${t("productionCountry")}: ${filters.productionCountries.map(countryName).join(", ")}`,
      });
    return chips;
  });

  function resultCacheKey() {
    const filters = appliedMovieFilters.value
      ? {
          ...appliedMovieFilters.value,
          includeKeywords: appliedMovieFilters.value.includeKeywords.map(
            (keyword) => keyword.id,
          ),
          excludeKeywords: appliedMovieFilters.value.excludeKeywords.map(
            (keyword) => keyword.id,
          ),
          includeCompanies: appliedMovieFilters.value.includeCompanies.map(
            (company) => company.id,
          ),
        }
      : null;
    return JSON.stringify({
      kind: searchType.value,
      query: query.value.trim(),
      downloaded: downloadedOnly.value,
      filters,
      sort: exploreSort.value,
      language: session.user?.language || "en-US",
    });
  }

  function cloneResultSet(result: ExploreResultSet): ExploreResultSet {
    return {
      ...result,
      items: [...result.items],
      availability: result.availability
        ? result.availability.map(([key, value]) => [key, { ...value }])
        : undefined,
    };
  }

  function cachedResult(key: string) {
    const result = resultsCache.get(key);
    if (!result) return null;
    resultsCache.delete(key);
    resultsCache.set(key, result);
    return cloneResultSet(result);
  }

  function cacheResult(key: string, result: ExploreResultSet) {
    resultsCache.delete(key);
    resultsCache.set(key, cloneResultSet(result));
    while (resultsCache.size > RESULT_CACHE_LIMIT) {
      const oldest = resultsCache.keys().next().value;
      if (oldest === undefined) break;
      resultsCache.delete(oldest);
    }
  }

  function commitResult(result: ExploreResultSet, append = false) {
    items.value = append ? [...items.value, ...result.items] : result.items;
    searchPage.value = result.page;
    searchTotalPages.value = result.totalPages;
    searchTotalResults.value = result.totalResults;
    for (const [key, value] of result.availability || [])
      availability[key] = value;
    if (!result.availability) void checkCards(result.items);
  }

  function resultSetWithCurrentItems(): ExploreResultSet {
    return {
      items: items.value,
      page: searchPage.value,
      totalPages: searchTotalPages.value,
      totalResults: searchTotalResults.value,
    };
  }

  function waitForResultFade() {
    if (!items.value.length) return Promise.resolve();
    resultsTransitioning.value = true;
    return new Promise<void>((resolve) => setTimeout(resolve, RESULT_FADE_MS));
  }

  function discoverySortBy() {
    if (exploreSort.value === "release")
      return searchType.value === "movie"
        ? "primary_release_date.desc"
        : "first_air_date.desc";
    if (exploreSort.value === "rating") return "vote_average.desc";
    if (exploreSort.value === "title")
      return searchType.value === "movie"
        ? "original_title.asc"
        : "original_name.asc";
    return "popularity.desc";
  }

  async function fetchDiscovery(
    request: number,
    page = 1,
  ): Promise<ExploreResultSet | null> {
    const kind = searchType.value;
    const params = new URLSearchParams({
      language: session.user?.language || "en-US",
      page: String(page),
      sort_by: discoverySortBy(),
    });
    if (exploreSort.value === "rating") params.set("vote_count.gte", "100");
    const result = await api<any>(`/catalog/discover/${kind}?${params}`);
    if (!exploreRequests.isLatest(request)) return null;
    const next = (result.results || []).map((item: any) => ({
      ...item,
      media_type: kind,
    }));
    return {
      items: next,
      page: Number(result.page) || page,
      totalPages: Number(result.total_pages) || 1,
      totalResults: Number(result.total_results) || next.length,
    };
  }
  async function fetchTextSearch(
    request: number,
    page = 1,
  ): Promise<ExploreResultSet | null> {
    const kind = searchType.value;
    const searchQuery = query.value;
    const r = await api<any>(
      `/catalog/search/${kind}?query=${encodeURIComponent(searchQuery)}&language=${session.user?.language}&page=${page}`,
    );
    if (!exploreRequests.isLatest(request)) return null;
    const next = (r.results || []).map((x: any) => ({
      ...x,
      media_type: kind,
    }));
    return {
      items: sortTextMedia(next),
      page: Number(r.page) || page,
      totalPages: Number(r.total_pages) || 1,
      totalResults: Number(r.total_results) || next.length,
    };
  }
  async function fetchDownloaded(
    request: number,
  ): Promise<ExploreResultSet | null> {
    const kind = searchType.value;
    const response = await api<{ items?: DownloadedLibraryItem[] }>("/library");
    if (!exploreRequests.isLatest(request)) return null;
    let next = downloadedMediaFromLibrary(response.items || [], kind);
    const filters = appliedMovieFilters.value;
    if (downloadedFiltersNeedDetails(filters)) {
      const suffix = [
        kind === "movie" ? "release_dates" : "content_ratings",
        filters?.includeKeywords.length || filters?.excludeKeywords.length
          ? "keywords"
          : "",
      ]
        .filter(Boolean)
        .join(",");
      next = await Promise.all(
        next.map(async (media): Promise<DownloadedExploreMedia> => {
          try {
            const details = await api<DownloadedExploreMedia>(
              `/catalog/${kind}/${media.id}?append_to_response=${suffix}&language=${session.user?.language || "en-US"}`,
            );
            return {
              ...media,
              ...details,
              id: media.id,
              media_type: kind,
              _ageRating: usAgeFromDetails(details, kind),
            };
          } catch {
            return media;
          }
        }),
      );
      if (!exploreRequests.isLatest(request)) return null;
    }
    next = sortDownloadedMedia(
      next.filter((media) =>
        matchesDownloadedMedia(media, query.value, filters),
      ),
      exploreSort.value,
    );
    return {
      items: next,
      page: 1,
      totalPages: 1,
      totalResults: next.length,
      availability: next.map((media) => [
        `${kind}:${media.id}`,
        {
          available: true,
          state: kind === "movie" ? "complete" : "partial",
        },
      ]),
    };
  }
  async function navigateExplore(nextQuery: Record<string, string>) {
    const target = { path: "/", query: nextQuery };
    if (router.resolve(target).fullPath === currentRoute.fullPath)
      await loadExploreFromURL();
    else await router.push(target);
  }
  async function doSearch() {
    if (!query.value.trim()) return;
    await navigateExplore(
      exploreQuery(searchType.value, query.value, null, downloadedOnly.value),
    );
  }
  async function loadMovieFilterOptions() {
    const kind = searchType.value;
    if (
      filterGenreKind.value === kind &&
      movieGenres.value.length &&
      movieCountries.value.length
    )
      return;
    const request = filterOptionRequests.begin();
    filterOptionsLoading.value = true;
    try {
      const requests: [Promise<any>, Promise<any>] = [
        api<any>(
          `/catalog/genre/${kind}/list?language=${session.user?.language}`,
        ),
        movieCountries.value.length
          ? Promise.resolve(movieCountries.value)
          : api<any>(
              `/catalog/configuration/countries?language=${session.user?.language}`,
            ),
      ];
      const [genres, countries] = await Promise.all(requests);
      if (!filterOptionRequests.isLatest(request)) return;
      movieGenres.value = genres.genres || [];
      filterGenreKind.value = kind;
      if (!movieCountries.value.length)
        movieCountries.value = (countries || []).sort(
          (a: CountryOption, b: CountryOption) =>
            (a.native_name || a.english_name).localeCompare(
              b.native_name || b.english_name,
            ),
        );
    } finally {
      if (filterOptionRequests.isLatest(request))
        filterOptionsLoading.value = false;
    }
  }
  async function openMovieFilters() {
    filterPanelOpen.value = true;
    movieFilterDraft.value = cloneMovieFilters(
      appliedMovieFilters.value || createMovieFilters(session.user?.language),
    );
    try {
      await loadMovieFilterOptions();
    } catch (e: any) {
      error.value = e.message;
    }
  }
  async function filterAges(
    results: any[],
    filters: MovieFilters,
    kind: "movie" | "tv",
  ) {
    if (filters.ageMin === null && filters.ageMax === null) return results;
    return (
      await Promise.all(
        results.map(async (item) => {
          try {
            const suffix =
              kind === "movie" ? "release_dates" : "content_ratings";
            const details = await api<any>(
              `/catalog/${kind}/${item.id}?append_to_response=${suffix}`,
            );
            return matchesAge(usAgeFromDetails(details, kind), filters)
              ? item
              : null;
          } catch {
            return item;
          }
        }),
      )
    ).filter(Boolean);
  }
  async function fetchFiltered(
    request: number,
    page = 1,
  ): Promise<ExploreResultSet | null> {
    const filters = appliedMovieFilters.value;
    if (!filters) return null;
    const kind = searchType.value;
    const params = discoverParams(
      filters,
      session.user?.language || "en-US",
      page,
      kind,
      exploreSort.value === "release"
        ? kind === "movie"
          ? "primary_release_date.desc"
          : "first_air_date.desc"
        : exploreSort.value === "rating"
          ? "vote_average.desc"
          : exploreSort.value === "title"
            ? "original_title.asc"
            : "popularity.desc",
    );
    if (exploreSort.value === "rating") params.set("vote_count.gte", "100");
    const r = await api<any>(`/catalog/discover/${kind}?${params}`);
    if (!exploreRequests.isLatest(request)) return null;
    const raw = (r.results || []).map((x: any) => ({
      ...x,
      media_type: kind,
    }));
    const ageFiltered = await filterAges(raw, filters, kind);
    if (!exploreRequests.isLatest(request)) return null;
    const next = ageFiltered.filter((item) =>
      matchesRottenTomatoes(item.rotten_tomatoes_score, filters),
    );
    return {
      items: next,
      page: Number(r.page) || page,
      totalPages: Number(r.total_pages) || 1,
      totalResults: Number(r.total_results) || next.length,
    };
  }
  async function applyMovieFilters(filters: MovieFilters) {
    movieFilterDraft.value = normalizeMovieFilters(filters);
    await navigateExplore(
      exploreQuery(
        searchType.value,
        "",
        movieFilterDraft.value,
        downloadedOnly.value,
      ),
    );
  }
  async function resetMovieFilters(
    filters = createMovieFilters(session.user?.language),
  ) {
    movieFilterDraft.value = cloneMovieFilters(filters);
    await navigateExplore(
      exploreQuery(searchType.value, "", null, downloadedOnly.value),
    );
  }
  async function removeMovieFilter(group: string) {
    if (!appliedMovieFilters.value) return;
    const next = cloneMovieFilters(appliedMovieFilters.value);
    if (group === "year") {
      next.yearMin = null;
      next.yearMax = null;
    }
    if (group === "rating") {
      next.ratingMin = null;
      next.ratingMax = null;
    }
    if (group === "rottenTomatoes") {
      next.rottenTomatoesMin = null;
      next.rottenTomatoesMax = null;
    }
    if (group === "runtime") {
      next.runtimeMin = null;
      next.runtimeMax = null;
    }
    if (group === "age") {
      next.ageMin = null;
      next.ageMax = null;
    }
    if (group === "includeGenres") next.includeGenres = [];
    if (group === "excludeGenres") next.excludeGenres = [];
    if (group === "includeKeywords") next.includeKeywords = [];
    if (group === "excludeKeywords") next.excludeKeywords = [];
    if (group === "company") next.includeCompanies = [];
    if (group === "country") next.productionCountries = [];
    await navigateExplore(
      exploreQuery(
        searchType.value,
        "",
        activeMovieFilterCount(next) ? next : null,
        downloadedOnly.value,
      ),
    );
  }
  async function loadExploreFromURL() {
    const request = exploreRequests.begin();
    loadingMore.value = false;
    const state = parseExploreQuery(
      currentRoute.query as Record<string, unknown>,
      session.user?.language,
    );
    searchType.value = state.kind;
    query.value = state.query;
    downloadedOnly.value = state.downloaded;
    if (state.filters && !rottenTomatoesEnabled.value) {
      state.filters.rottenTomatoesMin = null;
      state.filters.rottenTomatoesMax = null;
      if (!activeMovieFilterCount(state.filters)) state.filters = null;
    }
    appliedMovieFilters.value = state.filters
      ? cloneMovieFilters(state.filters)
      : null;
    movieFilterDraft.value = cloneMovieFilters(
      state.filters || createMovieFilters(session.user?.language),
    );
    if (state.filters) void hydrateKeywordNames(state.filters, request);
    searchStarted.value = Boolean(state.query || state.filters);
    searchMode.value = state.query
      ? "text"
      : state.filters
        ? "filters"
        : "idle";
    const cacheKey = resultCacheKey();
    const cached = cachedResult(cacheKey);
    // A cached destination can replace the current collection synchronously.
    // Only hide the old collection when the destination must wait on the API.
    const fade = cached ? null : waitForResultFade();
    busy.value = true;
    try {
      const freshResult = (async () => {
        if (searchMode.value !== "idle") await loadMovieFilterOptions();
        if (!exploreRequests.isLatest(request)) return null;
        if (downloadedOnly.value) return fetchDownloaded(request);
        if (searchMode.value === "text") return fetchTextSearch(request, 1);
        if (searchMode.value === "filters") return fetchFiltered(request, 1);
        return fetchDiscovery(request, 1);
      })();

      if (cached) {
        if (!exploreRequests.isLatest(request)) return;
        commitResult(cached);
        resultsTransitioning.value = false;
      }

      const fresh = await freshResult;
      if (!fresh || !exploreRequests.isLatest(request)) return;
      if (fade) await fade;
      if (!exploreRequests.isLatest(request)) return;
      commitResult(fresh);
      cacheResult(cacheKey, fresh);
      resultsTransitioning.value = false;
    } finally {
      if (exploreRequests.isLatest(request)) {
        busy.value = false;
        resultsTransitioning.value = false;
      }
    }
  }

  async function hydrateKeywordNames(filters: MovieFilters, request: number) {
    const keywords = [...filters.includeKeywords, ...filters.excludeKeywords];
    const unresolved = keywords.filter(
      (keyword) => keyword.name === String(keyword.id),
    );
    if (!unresolved.length) return;
    try {
      const resolved = await resolveKeywords(
        unresolved.map((keyword) => keyword.id),
      );
      if (!exploreRequests.isLatest(request)) return;
      const names = new Map(
        resolved.map((keyword) => [keyword.id, keyword.name]),
      );
      const hydrate = (source: MovieFilters) => {
        const next = cloneMovieFilters(source);
        next.includeKeywords = next.includeKeywords.map((keyword) => ({
          ...keyword,
          name: names.get(keyword.id) || keyword.name,
        }));
        next.excludeKeywords = next.excludeKeywords.map((keyword) => ({
          ...keyword,
          name: names.get(keyword.id) || keyword.name,
        }));
        return next;
      };
      if (appliedMovieFilters.value)
        appliedMovieFilters.value = hydrate(appliedMovieFilters.value);
      movieFilterDraft.value = hydrate(movieFilterDraft.value);
    } catch {
      // Filtering still works from IDs when the local catalog is unavailable.
    }
  }
  async function loadMoreSearchResults() {
    if (!hasMoreSearchResults.value || busy.value || loadingMore.value) return;
    const request = exploreRequests.begin();
    loadingMore.value = true;
    try {
      const next =
        searchMode.value === "filters"
          ? await fetchFiltered(request, searchPage.value + 1)
          : searchMode.value === "text"
            ? await fetchTextSearch(request, searchPage.value + 1)
            : await fetchDiscovery(request, searchPage.value + 1);
      if (!next || !exploreRequests.isLatest(request)) return;
      commitResult(next, true);
      if (searchMode.value === "text") sortTextResults();
      cacheResult(resultCacheKey(), resultSetWithCurrentItems());
    } finally {
      if (exploreRequests.isLatest(request)) loadingMore.value = false;
    }
  }
  async function selectSearchType(type: "movie" | "tv") {
    const mediaTypeChanged = searchType.value !== type;
    searchType.value = type;
    filterGenreKind.value = null;
    const filters = appliedMovieFilters.value
      ? cloneMovieFilters(appliedMovieFilters.value)
      : null;
    if (type === "tv" && filters) {
      filters.runtimeMin = null;
      filters.runtimeMax = null;
    }
    if (mediaTypeChanged && filters) {
      // TMDB assigns different genre IDs to movies and TV. Keeping the old IDs
      // produces invalid chips and empty results after changing media type.
      filters.includeGenres = [];
      filters.excludeGenres = [];
    }
    await navigateExplore(
      exploreQuery(
        type,
        searchMode.value === "text" ? query.value : "",
        searchMode.value === "filters" ? filters : null,
        downloadedOnly.value,
      ),
    );
  }
  async function setDownloadedOnly(value: boolean) {
    if (downloadedOnly.value === value) return;
    downloadedOnly.value = value;
    await navigateExplore(
      exploreQuery(
        searchType.value,
        searchMode.value === "text" ? query.value : "",
        searchMode.value === "filters" ? appliedMovieFilters.value : null,
        value,
      ),
    );
  }
  function sortTextMedia(media: ExploreMedia[]) {
    const direction = exploreSort.value === "title" ? 1 : -1;
    return [...media].sort((a: any, b: any) => {
      const left =
        exploreSort.value === "title"
          ? title(a).toLocaleLowerCase()
          : exploreSort.value === "release"
            ? a.release_date || a.first_air_date || ""
            : exploreSort.value === "rating"
              ? Number(a.vote_count) >= 100
                ? Number(a.vote_average)
                : -1
              : Number(a.popularity);
      const right =
        exploreSort.value === "title"
          ? title(b).toLocaleLowerCase()
          : exploreSort.value === "release"
            ? b.release_date || b.first_air_date || ""
            : exploreSort.value === "rating"
              ? Number(b.vote_count) >= 100
                ? Number(b.vote_average)
                : -1
              : Number(b.popularity);
      return left < right ? -direction : left > right ? direction : 0;
    });
  }
  function sortTextResults() {
    items.value = sortTextMedia(items.value);
  }
  async function changeExploreSort() {
    loadingMore.value = false;
    if (downloadedOnly.value) {
      items.value = sortDownloadedMedia(items.value, exploreSort.value);
      cacheResult(resultCacheKey(), resultSetWithCurrentItems());
      return;
    }
    if (searchMode.value === "text") {
      sortTextResults();
      cacheResult(resultCacheKey(), resultSetWithCurrentItems());
      return;
    }
    const request = exploreRequests.begin();
    const fade = waitForResultFade();
    if (searchMode.value === "filters") {
      busy.value = true;
      try {
        const next = await fetchFiltered(request, 1);
        if (!next || !exploreRequests.isLatest(request)) return;
        await fade;
        if (!exploreRequests.isLatest(request)) return;
        commitResult(next);
        cacheResult(resultCacheKey(), next);
      } finally {
        if (exploreRequests.isLatest(request)) {
          busy.value = false;
          resultsTransitioning.value = false;
        }
      }
    } else {
      busy.value = true;
      try {
        const next = await fetchDiscovery(request, 1);
        if (!next || !exploreRequests.isLatest(request)) return;
        await fade;
        if (!exploreRequests.isLatest(request)) return;
        commitResult(next);
        cacheResult(resultCacheKey(), next);
      } finally {
        if (exploreRequests.isLatest(request)) {
          busy.value = false;
          resultsTransitioning.value = false;
        }
      }
    }
  }
  function maybeLoadMore() {
    if (
      pathOnly.value === "/" &&
      hasMoreSearchResults.value &&
      !busy.value &&
      !loadingMore.value &&
      document.documentElement.scrollHeight -
        window.scrollY -
        window.innerHeight <
        900
    )
      void loadMoreSearchResults();
  }

  return {
    appliedFilterChips,
    appliedMovieFilters,
    applyMovieFilters,
    changeExploreSort,
    doSearch,
    downloadedOnly,
    exploreSort,
    filterOptionsLoading,
    filterPanelOpen,
    hasMoreSearchResults,
    items,
    loadExploreFromURL,
    loadMoreSearchResults,
    loadingMore,
    maybeLoadMore,
    movieCountries,
    movieFilterCount,
    movieFilterDraft,
    movieGenres,
    openMovieFilters,
    query,
    removeMovieFilter,
    resetMovieFilters,
    resultsTransitioning,
    searchMode,
    searchStarted,
    searchTotalResults,
    searchType,
    selectSearchType,
    setDownloadedOnly,
    visibleItems,
  };
}
