import type { ComputedRef, Ref } from "vue";
import type { RouteLocationNormalizedLoaded } from "vue-router";
import { api } from "./api";
import {
  createDetailRequest,
  isCurrentDetailRequest,
  isLatestDetailRequest,
  type DetailRequest,
} from "./detailState";
import type { TranslationKey } from "./i18n";
import { movieCredits } from "./mediaCredits";
import { queryString } from "./navigationState";
import { initialOpenSeason } from "./seasonState";
import type { Availability } from "./useAvailabilityTorrents";
import type { ExploreMedia } from "./useExplore";

type Options = {
  currentRoute: RouteLocationNormalizedLoaded;
  pathOnly: ComputedRef<string>;
  currentKind: ComputedRef<string>;
  detail: Ref<any>;
  detailProgress: Ref<any | null>;
  progress: Ref<any[]>;
  availability: Record<string, Availability>;
  checkOne: (media: any, kind?: string) => Promise<any>;
  checkCards: (media: ExploreMedia[]) => Promise<void>;
  mediaInput: (media: any, kind?: string) => unknown;
  streamInChunks: (
    inputs: unknown[],
    onItem: (item: any) => void,
  ) => Promise<void>;
  restorePlayerFromRoute: () => void;
  t: (key: TranslationKey) => string;
  flash: (message: string) => void;
};

export function useMediaDetail(options: Options) {
  const {
    currentRoute,
    pathOnly,
    currentKind,
    detail,
    detailProgress,
    progress,
    availability,
    checkOne,
    checkCards,
    mediaInput,
    streamInChunks,
    restorePlayerFromRoute,
    t,
    flash,
  } = options;

  let detailRequestRevision = 0;
  async function loadDetail(
    path: string,
    refreshMetadata = false,
    revision = ++detailRequestRevision,
  ) {
    const parts = path.split("/").filter(Boolean);
    const kind = parts[0] || "movie",
      id = parts[1] || "0";
    const request = createDetailRequest(revision, kind, id);
    const refreshQuery = refreshMetadata ? "?refresh=true" : "";
    const keywordsRequest =
      kind === "movie" || kind === "tv"
        ? api<any>(`/catalog/${kind}/${id}/keywords${refreshQuery}`).catch(
            () => null,
          )
        : Promise.resolve(null);
    const creditsRequest =
      kind === "movie"
        ? api<any>(`/catalog/movie/${id}/credits${refreshQuery}`).catch(
            () => null,
          )
        : Promise.resolve(null);
    const [doc, progressDoc, progressList] = await Promise.all([
      api<any>(`/catalog/${kind}/${id}${refreshQuery}`),
      kind === "movie"
        ? api<any>(`/progress/movie/${id}`).catch(() => null)
        : Promise.resolve(null),
      kind === "collection"
        ? Promise.resolve({ items: progress.value })
        : api<any>("/progress").catch(() => ({ items: progress.value })),
    ]);
    if (
      !isLatestDetailRequest(request, detailRequestRevision) ||
      pathOnly.value !== `/${kind}/${id}`
    )
      return;
    progress.value = progressList.items || progress.value;
    doc.media_type = kind;
    doc._keywords = [];
    doc._credits = { directors: [], cast: [] };
    doc._recommendations = [];
    if (kind === "tv" && doc.seasons) {
      const routeSeason = Number(queryString(currentRoute.query.season as any));
      const openSeason = initialOpenSeason(
        Number(id),
        progress.value,
        routeSeason,
      );
      for (const season of doc.seasons) {
        season._episodes = [];
        season._loaded = false;
        season._loading = false;
        season._loadError = "";
        season._open = openSeason > 0 && season.season_number === openSeason;
      }
    }
    detail.value = doc;
    detailProgress.value = progressDoc;
    void hydrateDetailKeywords(keywordsRequest, request);
    void hydrateDetailCredits(creditsRequest, request);
    if (kind === "tv") {
      // Use the reactive season proxy exposed through detail.value. Mutating a
      // season from the raw TMDB document after assigning it to a Vue ref does
      // not notify the template, which left automatically opened seasons on the
      // loading state until the user toggled them manually.
      const open = detail.value.seasons?.find(
        (season: any) => season.season_number > 0 && season._open,
      );
      if (open) void loadSeasonEpisodes(open);
      scheduleSeasonPreload(detail.value);
    }
    void hydrateDetail(doc, request);
  }

  async function hydrateDetailKeywords(
    requestPromise: Promise<any | null>,
    request: DetailRequest,
  ) {
    const payload = await requestPromise;
    if (
      !payload ||
      !isCurrentDetailRequest(
        request,
        detailRequestRevision,
        currentKind.value,
        detail.value,
      )
    )
      return;
    const source = request.kind === "tv" ? payload.results : payload.keywords;
    const seen = new Set<number>();
    detail.value._keywords = (source || []).flatMap((keyword: any) => {
      const id = Number(keyword?.id);
      const name = String(keyword?.name || "").trim();
      if (!Number.isInteger(id) || id <= 0 || !name || seen.has(id)) return [];
      seen.add(id);
      return [{ id, name }];
    });
  }

  async function hydrateDetailCredits(
    requestPromise: Promise<any | null>,
    request: DetailRequest,
  ) {
    const payload = await requestPromise;
    if (
      !payload ||
      request.kind !== "movie" ||
      !isCurrentDetailRequest(
        request,
        detailRequestRevision,
        currentKind.value,
        detail.value,
      )
    )
      return;
    detail.value._credits = movieCredits(payload);
  }

  function scheduleSeasonPreload(show: any) {
    // Let Vue paint the lightweight season list first. The next task starts all
    // season requests together so collapsed rows gain availability information
    // without making the initial detail page wait for them.
    window.setTimeout(() => {
      if (detail.value !== show) return;
      for (const season of show.seasons || [])
        if (season.season_number >= 0) void loadSeasonEpisodes(season);
    }, 0);
  }

  async function hydrateDetail(doc: any, request: DetailRequest) {
    const { kind, id } = request;
    const a = await checkOne(doc, kind).catch(() => null);
    if (
      !isCurrentDetailRequest(
        request,
        detailRequestRevision,
        currentKind.value,
        detail.value,
      )
    )
      return;
    if (a)
      availability[`${kind}:${id}`] = {
        ...a,
        state: a.available ? "complete" : a.torrent ? "downloading" : "none",
      };
    restorePlayerFromRoute();
    if (kind !== "movie" && kind !== "tv") return;
    const recommendations = await api<any>(
      `/catalog/${kind}/${id}/recommendations`,
    ).catch(() => ({ results: [] }));
    if (
      !isCurrentDetailRequest(
        request,
        detailRequestRevision,
        currentKind.value,
        detail.value,
      )
    )
      return;
    const nextRecommendations = (recommendations.results || [])
      .slice(0, 12)
      .map((x: any) => ({ ...x, media_type: kind }));
    detail.value._recommendations = nextRecommendations;
    void checkCards(nextRecommendations);
  }

  function updateSeasonAvailability(season: any) {
    const episodes = season._episodes || [];
    const checked = episodes.filter(
      (episode: any) => availability[`episode:${episode.id}`],
    ).length;
    const available = episodes.filter(
      (episode: any) => availability[`episode:${episode.id}`]?.available,
    ).length;
    season._availableCount = available;
    season._availabilityState =
      checked < episodes.length
        ? "checking"
        : available === 0
          ? "none"
          : available === episodes.length
            ? "complete"
            : "partial";
  }

  async function loadSeasonEpisodes(season: any) {
    if (season._loaded || season._loading || !detail.value) return;
    const show = detail.value;
    const showID = show.id;
    season._loading = true;
    season._loadError = "";
    try {
      const seasonDoc = await api<any>(
        `/catalog/tv/${showID}/season/${season.season_number}`,
      );
      if (detail.value !== show) return;
      season._episodes = (seasonDoc.episodes || []).map((episode: any) => ({
        ...episode,
        show: show.name,
        showId: show.id,
        showOriginalName: show.original_name,
        showOriginalLanguage: show.original_language,
      }));
      season._loaded = true;
      updateSeasonAvailability(season);
      if (season._episodes.length)
        await streamInChunks(
          season._episodes.map((episode: any) =>
            mediaInput(episode, "episode"),
          ),
          (value: any) => {
            if (detail.value !== show) return;
            availability[`episode:${value.tmdbId}`] = value;
            updateSeasonAvailability(season);
            restorePlayerFromRoute();
          },
        ).catch(() => {
          if (detail.value === show) updateSeasonAvailability(season);
        });
      if (detail.value === show) restorePlayerFromRoute();
    } catch (reason: any) {
      if (detail.value === show)
        season._loadError = reason?.message || t("episodesLoadFailed");
    } finally {
      if (detail.value === show) season._loading = false;
    }
  }

  function onSeasonToggle(season: any, event: Event) {
    const open = (event.currentTarget as HTMLDetailsElement).open;
    season._open = open;
    if (open) void loadSeasonEpisodes(season);
  }

  async function refreshDetail() {
    const path = currentRoute.path || "/";
    await loadDetail(path, true);
    flash(t("refresh"));
  }

  return {
    loadDetail,
    loadSeasonEpisodes,
    onSeasonToggle,
    refreshDetail,
  };
}
