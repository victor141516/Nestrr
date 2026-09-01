import {
  computed,
  reactive,
  ref,
  watch,
  type ComputedRef,
  type Ref,
} from "vue";
import type { RouteLocationNormalizedLoaded, Router } from "vue-router";
import { api, session } from "./api";
import type { TranslationKey } from "./i18n";
import { preferredAudioLanguage } from "./mediaPreferences";
import {
  playbackQuery,
  playbackRequested,
  queryString,
  withoutPlayback,
} from "./navigationState";
import type { PlarrEvent } from "./playerProtocol";
import { useAutoplayCountdown } from "./useAutoplayCountdown";

export type PlaybackAvailability = {
  available: boolean;
  playerUrl?: string;
  originalDownloadUrl?: string;
};

export type PlaybackItem = {
  id: number;
  title?: string;
  name?: string;
  show?: string;
  showId?: number;
  season_number?: number;
  episode_number?: number;
  [key: string]: unknown;
};

type UsePlaybackHostOptions = {
  router: Router;
  currentRoute: RouteLocationNormalizedLoaded;
  pathOnly: ComputedRef<string>;
  detail: Ref<any>;
  currentKind: ComputedRef<string>;
  availability: Record<string, PlaybackAvailability>;
  progress: Ref<any[]>;
  t: (key: TranslationKey) => string;
  title: (item: PlaybackItem) => string;
  flash: (message: string) => void;
  go: (path: string) => void;
  loadListState: () => Promise<void>;
};

export function usePlaybackHost(options: UsePlaybackHostOptions) {
  const {
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
  } = options;
  const player = ref<{
    url: string;
    origin: string;
    item: PlaybackItem;
    kind: string;
  } | null>(null);
  const playbackGate = reactive({
    sessionID: "",
    accumulatedMs: 0,
    lastAt: 0,
    playing: false,
    qualified: false,
    hadProgress: false,
  });
  let playbackEventRevision = 0;
  const {
    countdown: autoplay,
    schedule: scheduleAutoplayCountdown,
    cancel: cancelAutoplay,
  } = useAutoplayCountdown();
  const wantsPlayback = computed(() =>
    playbackRequested(currentRoute.query as Record<string, any>),
  );

  function playerRoute(item: any, kind: string) {
    const path = kind === "episode" ? `/tv/${item.showId}` : pathOnly.value;
    return {
      path,
      query: playbackQuery(
        currentRoute.query as Record<string, any>,
        item,
        kind,
      ),
    };
  }
  function closedPlayerRoute() {
    return {
      path: pathOnly.value,
      query: withoutPlayback(currentRoute.query as Record<string, any>),
    };
  }
  function closePlayer(syncRoute = true) {
    cancelAutoplay();
    player.value = null;
    if (!syncRoute || !wantsPlayback.value) return;
    const target = closedPlayerRoute();
    const targetPath = router.resolve(target).fullPath;
    if (window.history.state?.back === targetPath) router.back();
    else void router.replace(target);
  }
  function openPlayer(
    m: PlaybackItem,
    a: PlaybackAvailability,
    kind = "movie",
    syncRoute = true,
  ) {
    if (!a?.playerUrl) return;
    const pref = progress.value.find(
      (p) => p.mediaType === kind && p.tmdbId === m.id,
    );
    const params = new URLSearchParams();
    if (pref?.resumeSeconds) params.set("start", String(pref.resumeSeconds));
    const audio = preferredAudioLanguage(
      pref?.audioLanguage,
      session.user?.language,
    );
    if (audio) params.set("audio", audio);
    if (pref?.subtitleLanguage) params.set("subtitles", pref.subtitleLanguage);
    if (pref?.subtitlesEnabled) params.set("subtitlesEnabled", "true");
    params.set("contentId", `${kind}:${m.id}`);
    const catalogTitle =
      kind === "episode" ? (m as any).show || title(m) : title(m);
    if (catalogTitle) params.set("title", catalogTitle);
    params.set("parentOrigin", location.origin);
    const url = a.playerUrl + (params.size ? "?" + params : "");
    player.value = { url, origin: new URL(a.playerUrl).origin, item: m, kind };
    Object.assign(playbackGate, {
      sessionID: "",
      accumulatedMs: 0,
      lastAt: 0,
      playing: false,
      qualified: Boolean(pref?.started),
      hadProgress: Boolean(pref?.started),
    });
    if (syncRoute) {
      const target = playerRoute(m, kind);
      if (wantsPlayback.value) void router.replace(target);
      else void router.push(target);
    }
  }
  function restorePlayerFromRoute() {
    if (!wantsPlayback.value || !detail.value) return;
    const kind = currentKind.value;
    if (kind === "movie") {
      if (
        player.value?.kind === "movie" &&
        player.value.item.id === detail.value.id
      )
        return;
      const a = availability[`movie:${detail.value.id}`];
      if (a?.playerUrl) openPlayer(detail.value, a, "movie", false);
      return;
    }
    if (kind !== "tv") return;
    const season = Number(queryString(currentRoute.query.season as any));
    const episode = Number(queryString(currentRoute.query.episode as any));
    if (!season || !episode) return;
    const currentItem: any = player.value?.item;
    if (
      player.value?.kind === "episode" &&
      currentItem?.season_number === season &&
      currentItem?.episode_number === episode
    )
      return;
    const item = (detail.value.seasons || [])
      .flatMap((value: any) => value._episodes || [])
      .find(
        (value: any) =>
          value.season_number === season && value.episode_number === episode,
      );
    const a = item && availability[`episode:${item.id}`];
    if (a?.playerUrl) openPlayer(item, a, "episode", false);
  }
  function playDetail() {
    const kind = currentKind.value,
      a = availability[`${kind}:${detail.value.id}`];
    if (a) openPlayer(detail.value, a, kind);
  }
  function renditionURL(a: PlaybackAvailability | undefined, quality: string) {
    return (
      a?.originalDownloadUrl?.replace(
        "/download/original",
        `/download/${quality}`,
      ) || "#"
    );
  }
  function playEpisode(e: any) {
    const a = availability[`episode:${e.id}`];
    if (a) openPlayer(e, a, "episode");
  }
  async function navigateEpisode(direction: "next" | "previous") {
    const p = player.value;
    if (!p || p.kind !== "episode") return;
    const item: any = p.item;
    const result = await api<any>("/playback/navigation", {
      method: "POST",
      body: JSON.stringify({
        direction,
        showId: item.showId,
        showName: item.show,
        season: item.season_number,
        episode: item.episode_number,
      }),
    });
    if (result.endOfSeries) {
      closePlayer();
      flash("End of series");
      return;
    }
    const next = {
      ...result.item,
      showId: result.showId,
      show: result.showName,
    };
    if (result.availability?.available) {
      openPlayer(next, result.availability, "episode");
    } else {
      closePlayer(false);
      go(
        `/tv/${result.showId}?season=${next.season_number}&episode=${next.episode_number}`,
      );
      flash(t("unavailable"));
    }
  }
  function scheduleAutoplay() {
    scheduleAutoplayCountdown(() => void navigateEpisode("next"));
  }
  async function onPlayerMessage(e: PlarrEvent) {
    const p = player.value;
    if (!p) return;
    const eventRevision = ++playbackEventRevision;
    const now = performance.now();
    if (playbackGate.sessionID !== e.sessionId) {
      playbackGate.sessionID = e.sessionId;
      playbackGate.accumulatedMs = 0;
      playbackGate.lastAt = now;
      playbackGate.playing = false;
      playbackGate.qualified = playbackGate.hadProgress;
    }
    if (e.type === "seek") {
      playbackGate.accumulatedMs = 0;
      playbackGate.qualified = playbackGate.hadProgress;
    } else if (playbackGate.playing && playbackGate.lastAt) {
      // Ignore long gaps caused by a sleeping/backgrounded tab. Heartbeats are
      // frequent, so a ten-second cap counts actual playback without allowing a
      // suspended page to satisfy the initial threshold.
      playbackGate.accumulatedMs += Math.min(10_000, now - playbackGate.lastAt);
    }
    if (playbackGate.accumulatedMs >= 30_000) playbackGate.qualified = true;
    playbackGate.playing =
      e.type === "play" ||
      (!e.paused &&
        !["pause", "closed", "page-hidden", "ended"].includes(e.type));
    playbackGate.lastAt = now;
    if (e.type === "closed") closePlayer();
    const saved = await api<any>("/playback/events", {
      method: "POST",
      body: JSON.stringify({
        ...e,
        mediaType: p.kind,
        tmdbId: p.item.id,
        showTmdbId: p.kind === "episode" ? (p.item as any).showId : 0,
        seasonNumber: p.kind === "episode" ? (p.item as any).season_number : 0,
        episodeNumber:
          p.kind === "episode" ? (p.item as any).episode_number : 0,
        qualified: playbackGate.qualified,
      }),
    }).catch(() => null);
    if (
      e.type === "closed" ||
      player.value !== p ||
      eventRevision !== playbackEventRevision
    )
      return;
    if (saved?.tmdbId) {
      const index = progress.value.findIndex(
        (item) =>
          item.mediaType === saved.mediaType && item.tmdbId === saved.tmdbId,
      );
      if (saved.started || saved.watched) {
        if (index >= 0) progress.value[index] = saved;
        else progress.value.unshift(saved);
      }
      playbackGate.hadProgress = Boolean(saved.started);
      if (saved.watched) await loadListState().catch(() => {});
    }
    if (e.type === "next-requested") void navigateEpisode("next");
    if (e.type === "previous-requested") void navigateEpisode("previous");
    if (e.type === "ended" && saved?.watched) {
      flash(t("watched"));
      if (p.kind === "episode") scheduleAutoplay();
    }
  }

  watch(wantsPlayback, (requested) => {
    if (!requested) closePlayer(false);
    else restorePlayerFromRoute();
  });

  return {
    autoplay,
    cancelAutoplay,
    closePlayer,
    navigateEpisode,
    onPlayerMessage,
    playDetail,
    playEpisode,
    player,
    renditionURL,
    restorePlayerFromRoute,
  };
}
