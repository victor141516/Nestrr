import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
  inject,
  type InjectionKey,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, artwork, session } from "./api";
import { tr, type TranslationKey } from "./i18n";
import { episodeImagePath } from "./episodeArtwork";
import { ACCENT_COLORS } from "./theme";
import { episodeDisplayTitle } from "./seasonState";
import { episodeCode, routeLoadQuery } from "./navigationState";
import AdvancedMovieFilters from "./AdvancedMovieFilters.vue";
import AppShellHeader from "./AppShellHeader.vue";
import AppIcon from "./AppIcon.vue";
import AuthView from "./AuthView.vue";
import EmbeddedPlayer from "./EmbeddedPlayer.vue";
import ListMediaRow from "./ListMediaRow.vue";
import MediaCard from "./MediaCard.vue";
import TorrentDownloadProgress from "./TorrentDownloadProgress.vue";
import TorrentSearchModal from "./TorrentSearchModal.vue";
import { useMediaLists } from "./useMediaLists";
import { useMediaDetail } from "./useMediaDetail";
import { useAdministration } from "./useAdministration";
import { useUserPreferences } from "./useUserPreferences";
import { useSessionBootstrap } from "./useSessionBootstrap";
import { useHomeLibrary } from "./useHomeLibrary";
import { useExplore, type ExploreMedia } from "./useExplore";
import { usePlaybackHost } from "./usePlaybackHost";
import {
  useAvailabilityTorrents,
  type Availability,
} from "./useAvailabilityTorrents";

type Media = ExploreMedia & {
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
  number_of_seasons?: number;
  seasons?: any[];
  belongs_to_collection?: any;
};
export function useNestrrApplication() {
  const currentRoute = useRoute(),
    router = useRouter(),
    route = computed(() => currentRoute.fullPath);
  const pageLoading = ref(false),
    detail = ref<any>(null);
  const progress = ref<any[]>([]);
  const toast = ref("");
  const detailProgress = ref<any | null>(null);
  const t = (key: TranslationKey) => tr(session.user?.language, key),
    title = (m: Media) => m.title || m.name || "",
    year = (m: Media) => (m.release_date || m.first_air_date || "").slice(0, 4);
  const currentKind = computed(
    () => route.value.split("/").filter(Boolean)[0] || "movie",
  );
  const healthyIntegrations = computed(
    () => Object.values(integrations).filter((v) => v.healthy).length,
  );
  const integrationCount = computed(() => Object.keys(integrations).length);
  const pathOnly = computed(() => currentRoute.path || "/");
  const validRoute = computed(
    () =>
      pathOnly.value === "/" ||
      pathOnly.value === "/library" ||
      pathOnly.value === "/lists" ||
      pathOnly.value === "/recommendations" ||
      pathOnly.value === "/settings" ||
      pathOnly.value === "/admin" ||
      /^\/lists\/\d+$/.test(pathOnly.value) ||
      /^\/(movie|tv|collection)\/\d+$/.test(pathOnly.value),
  );
  const {
    authenticate,
    busy,
    error,
    logout,
    needsBootstrap,
    ready,
    rottenTomatoesEnabled,
  } = useSessionBootstrap({ router, loadRoute: () => loadRoute() });
  const { continueItems, library, loadContinueItems, loadLibrary } =
    useHomeLibrary({ progress, title });
  const {
    availability,
    categories,
    checkCards,
    checkOne,
    currentAvailability,
    forceTorrent,
    forcingTorrents,
    integrations,
    loadStatus,
    mediaInput,
    onTorrentAccepted,
    openTorrent,
    streamInChunks,
    torrent,
  } = useAvailabilityTorrents({
    detail,
    currentKind,
    year,
    t,
    flash,
  });
  const {
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
    settleSearch,
    selectSearchType,
    setDownloadedOnly,
    visibleItems,
  } = useExplore({
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
  });
  const { loadDetail, loadSeasonEpisodes, onSeasonToggle, refreshDetail } =
    useMediaDetail({
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
      restorePlayerFromRoute: () => restorePlayerFromRoute(),
      t,
      flash,
    });
  const {
    acceptBulkImportedItem,
    activeList,
    activeListItems,
    busyListFor,
    finishBulkImport,
    inSystemList,
    listDirection,
    listEditor,
    listMediaType,
    listSort,
    loadListState,
    loadListsPage,
    mediaLists,
    membershipsFor,
    pendingRemovalFor,
    saveListEditor,
    setEpisodeWatched,
    setListMediaType,
    setListSort,
    setSeasonWatched,
    startedFor,
    systemListName,
    systemMembershipsFor,
    toggleCustomList,
    toggleListDirection,
    toggleRowList,
    toggleSystemList,
  } = useMediaLists({
    router,
    currentRoute,
    pathOnly,
    progress,
    detail,
    loadSeasonEpisodes: (season) => loadSeasonEpisodes(season),
    t,
    flash,
  });
  const {
    autoplay,
    cancelAutoplay,
    closePlayer,
    downloadMedia,
    navigateEpisode,
    onPlayerMessage,
    playDetail,
    playEpisode,
    player,
    restorePlayerFromRoute,
  } = usePlaybackHost({
    router,
    currentRoute,
    pathOnly,
    detail,
    currentKind,
    availability,
    progress,
    t,
    title,
    flash,
    go,
    loadListState,
    mediaInput,
  });
  const {
    addUser,
    adminAction,
    adminState,
    clearMetadata,
    confirmAdminAction,
    confirmMetadata,
    impersonate,
    loadAdmin,
    openAdminAction,
    recommendationRun,
    runRecommendationCatalog,
    stopImpersonation,
    users,
  } = useAdministration({
    router,
    t,
    flash,
    loadRoute: () => loadRoute(),
    go,
    busy,
  });
  const {
    accentColor,
    accentSaving,
    saveSettings,
    selectAccentColor,
    selectTheme,
    themePreference,
  } = useUserPreferences({
    t,
    flash,
    loadRoute: () => loadRoute(),
    error,
  });

  const documentTitle = computed(() => {
    if (!session.user) return "Nestrr";
    if (player.value) {
      const item: any = player.value.item;
      if (player.value.kind === "episode") {
        const parts = [
          item.show || title(item),
          episodeCode(item),
          title(item) === item.show ? "" : title(item),
        ].filter(Boolean);
        return `${parts.join(" · ")} · Nestrr`;
      }
      return `${title(item) || "Nestrr"}${title(item) ? " · Nestrr" : ""}`;
    }
    if (detail.value && /^\/(movie|tv|collection)\/\d+$/.test(pathOnly.value))
      return `${title(detail.value)} · Nestrr`;
    const page =
      pathOnly.value === "/"
        ? t("explore")
        : pathOnly.value === "/library"
          ? t("library")
          : pathOnly.value.startsWith("/lists")
            ? activeList.value?.systemKey
              ? systemListName(activeList.value.systemKey)
              : activeList.value?.name || t("lists")
            : pathOnly.value === "/recommendations"
              ? t("recommendations")
              : pathOnly.value === "/settings"
                ? t("settings")
                : pathOnly.value === "/admin"
                  ? t("admin")
                  : validRoute.value
                    ? ""
                    : t("pageNotFound");
    return page ? `${page} · Nestrr` : "Nestrr";
  });

  function go(path: string) {
    void router.push(path);
  }
  function flash(message: string) {
    toast.value = message;
    setTimeout(() => (toast.value = ""), 3000);
  }
  async function createWatchParty(media: any, kind: string) {
    const tab = window.open("", "_blank");
    if (tab) tab.opener = null;
    try {
      const room = await api<{ roomId: string; playerUrl: string }>(
        "/playback/rooms",
        { method: "POST", body: JSON.stringify(mediaInput(media, kind)) },
      );
      if (tab) tab.location.href = room.playerUrl;
      try {
        await navigator.clipboard.writeText(room.playerUrl);
        flash(t("watchPartyLinkCopied"));
      } catch {
        window.prompt(t("watchPartyShareLink"), room.playerUrl);
      }
    } catch (error) {
      tab?.close();
      flash(error instanceof Error ? error.message : t("watchParty"));
    }
  }
  function percent(p: any) {
    return Math.max(
      0,
      Math.min(
        100,
        p?.watched
          ? 100
          : p?.durationSeconds
            ? (100 * p.positionSeconds) / p.durationSeconds
            : 0,
      ),
    );
  }
  function fmtSeconds(value: number) {
    const seconds = Math.max(0, Math.floor(value || 0)),
      hours = Math.floor(seconds / 3600),
      minutes = Math.floor((seconds % 3600) / 60),
      rest = seconds % 60;
    return hours
      ? `${hours}:${String(minutes).padStart(2, "0")}:${String(rest).padStart(2, "0")}`
      : `${minutes}:${String(rest).padStart(2, "0")}`;
  }
  function progressFor(kind: string, id: number) {
    return progress.value.find((p) => p.mediaType === kind && p.tmdbId === id);
  }
  function integrationLabel(key: string) {
    return key === "qbittorrent"
      ? "qBittorrent"
      : key[0]?.toUpperCase() + key.slice(1);
  }
  let routeLoadRevision = 0;
  async function loadRoute() {
    const detailRevision = ++routeLoadRevision;
    if (!session.user) return;
    error.value = "";
    detail.value = null;
    detailProgress.value = null;
    const path = pathOnly.value;
    const usePageLoader = path !== "/";
    pageLoading.value = usePageLoader;
    try {
      const isDetail = /^\/(movie|tv|collection)\/\d+$/.test(path);
      const statusRequest = loadStatus();
      const listsRequest = loadListState();
      if (!isDetail) await statusRequest;
      else void statusRequest.catch(() => {});
      await listsRequest;
      if (detailRevision !== routeLoadRevision) return;
      if (path === "/")
        await Promise.all([
          loadExploreFromURL(),
          loadContinueItems(progress.value),
        ]);
      else if (path === "/library") await loadLibrary();
      else if (path === "/lists" || /^\/lists\/\d+$/.test(path))
        await loadListsPage(path);
      else if (path === "/recommendations") {
      } else if (path === "/settings") {
      } else if (path === "/admin") await loadAdmin();
      else if (
        path.startsWith("/movie/") ||
        path.startsWith("/tv/") ||
        path.startsWith("/collection/")
      )
        await loadDetail(path, false);
    } catch (e: any) {
      if (detailRevision === routeLoadRevision) error.value = e.message;
    } finally {
      if (detailRevision === routeLoadRevision && usePageLoader)
        pageLoading.value = false;
    }
  }
  watch(documentTitle, (value) => (document.title = value), {
    immediate: true,
  });
  watch(
    () =>
      router.resolve({
        path: pathOnly.value,
        query: routeLoadQuery(
          pathOnly.value,
          currentRoute.query as Record<string, any>,
        ),
      }).fullPath,
    () => void loadRoute(),
  );
  onMounted(() => {
    window.addEventListener("scroll", maybeLoadMore, { passive: true });
  });
  onBeforeUnmount(() => {
    window.removeEventListener("scroll", maybeLoadMore);
  });

  return {
    ACCENT_COLORS,
    accentColor,
    accentSaving,
    acceptBulkImportedItem,
    activeList,
    activeListItems,
    addUser,
    adminAction,
    adminState,
    appliedFilterChips,
    appliedMovieFilters,
    applyMovieFilters,
    artwork,
    authenticate,
    autoplay,
    availability,
    busy,
    busyListFor,
    cancelAutoplay,
    categories,
    changeExploreSort,
    checkCards,
    clearMetadata,
    closePlayer,
    confirmAdminAction,
    confirmMetadata,
    continueItems,
    currentAvailability,
    currentKind,
    detail,
    detailProgress,
    doSearch,
    downloadMedia,
    downloadedOnly,
    episodeDisplayTitle,
    episodeImagePath,
    error,
    exploreSort,
    filterOptionsLoading,
    filterPanelOpen,
    flash,
    fmtSeconds,
    finishBulkImport,
    forceTorrent,
    forcingTorrents,
    go,
    hasMoreSearchResults,
    healthyIntegrations,
    impersonate,
    inSystemList,
    integrationCount,
    integrationLabel,
    integrations,
    items,
    library,
    listDirection,
    listEditor,
    listMediaType,
    listSort,
    loadLibrary,
    loadListsPage,
    loadMoreSearchResults,
    loadRoute,
    loadSeasonEpisodes,
    loadingMore,
    logout,
    mediaLists,
    membershipsFor,
    movieCountries,
    movieFilterCount,
    movieFilterDraft,
    movieGenres,
    navigateEpisode,
    needsBootstrap,
    onPlayerMessage,
    onSeasonToggle,
    onTorrentAccepted,
    openAdminAction,
    recommendationRun,
    runRecommendationCatalog,
    openMovieFilters,
    openTorrent,
    pageLoading,
    pathOnly,
    pendingRemovalFor,
    percent,
    playDetail,
    createWatchParty,
    playEpisode,
    player,
    progressFor,
    query,
    ready,
    refreshDetail,
    removeMovieFilter,
    resetMovieFilters,
    resultsTransitioning,
    rottenTomatoesEnabled,
    saveListEditor,
    saveSettings,
    searchMode,
    searchStarted,
    searchTotalResults,
    searchType,
    settleSearch,
    selectAccentColor,
    selectSearchType,
    setDownloadedOnly,
    selectTheme,
    session,
    setEpisodeWatched,
    setListMediaType,
    setListSort,
    setSeasonWatched,
    startedFor,
    stopImpersonation,
    systemListName,
    systemMembershipsFor,
    t,
    themePreference,
    title,
    toast,
    toggleCustomList,
    toggleListDirection,
    toggleRowList,
    toggleSystemList,
    torrent,
    users,
    validRoute,
    visibleItems,
    year,
  };
}

export type NestrrApplication = ReturnType<typeof useNestrrApplication>;
export const nestrrApplicationKey: InjectionKey<NestrrApplication> =
  Symbol("nestrr-application");
export function useNestrrApplicationContext(): NestrrApplication {
  const application = inject(nestrrApplicationKey);
  if (!application)
    throw new Error("Nestrr application context is unavailable");
  return application;
}
