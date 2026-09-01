import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  type ComputedRef,
  type Ref,
} from "vue";
import { api, streamAvailability } from "./api";
import type { TranslationKey } from "./i18n";
import type { ExploreMedia } from "./useExplore";
import {
  type TorrentMediaKind,
  type TorrentSearchItem,
} from "./torrentDefaults";
import {
  waitForTorrentForceConfirmation,
  type TorrentDownload,
} from "./torrentState";
import type { TorrentAccepted } from "./useTorrentSearch";

export type Availability = {
  kind: string;
  tmdbId: number;
  available: boolean;
  state?: "complete" | "partial" | "none" | "downloading";
  relativePath?: string;
  playerUrl?: string;
  originalDownloadUrl?: string;
  path: string;
  torrent?: TorrentDownload;
};

export type Integration = {
  configured: boolean;
  healthy: boolean;
  message?: string;
};

type Options = {
  detail: Ref<any>;
  currentKind: ComputedRef<string>;
  year: (media: ExploreMedia) => string;
  t: (key: TranslationKey) => string;
  flash: (message: string) => void;
};

export function useAvailabilityTorrents(options: Options) {
  const { detail, currentKind, year, t, flash } = options;
  const availability = reactive<Record<string, Availability>>({});
  const integrations = reactive<Record<string, Integration>>({});
  const categories = reactive({ movie: "Movies", show: "Shows" });
  const forcingTorrents = reactive<Record<string, boolean>>({});
  const torrent = ref<{
    item: TorrentSearchItem;
    kind: TorrentMediaKind;
  } | null>(null);
  const currentAvailability = computed(() =>
    detail.value
      ? availability[`${currentKind.value}:${detail.value.id}`]
      : undefined,
  );

  async function loadStatus() {
    const status = await api<any>("/integrations/status");
    Object.assign(integrations, status.integrations);
    Object.assign(categories, status.categories);
  }

  function mediaInput(media: any, kind?: string) {
    const mediaKind =
      kind || media.media_type || (media.title ? "movie" : "tv");
    return {
      kind:
        mediaKind === "movie"
          ? "movie"
          : mediaKind === "episode"
            ? "episode"
            : "show",
      tmdbId: media.id,
      title: media.title,
      originalTitle: media.original_title,
      originalLanguage: media.showOriginalLanguage || media.original_language,
      year: Number(year(media)) || 0,
      show: media.show || media.name,
      originalShow: media.showOriginalName || media.original_name,
      showTmdbId: media.showId,
      season: media.season_number,
      episode: media.episode_number,
    };
  }

  async function checkOne(media: any, kind?: string) {
    if ((kind || media.media_type) === "collection") return null;
    const response = await api<any>("/availability/check", {
      method: "POST",
      body: JSON.stringify({ items: [mediaInput(media, kind)] }),
    });
    return response.items[0];
  }

  async function refreshTorrentAvailability(media: any, kind: string) {
    const next = await checkOne(media, kind);
    if (!next) return undefined;
    const resolved: Availability = {
      ...next,
      state: next.available
        ? "complete"
        : next.torrent
          ? "downloading"
          : "none",
    };
    availability[`${kind}:${media.id}`] = resolved;
    return resolved;
  }

  async function forceTorrent(media: any, kind: string) {
    const key = `${kind}:${media.id}`;
    if (forcingTorrents[key]) return;
    forcingTorrents[key] = true;
    try {
      await api("/torrents/force-start", {
        method: "POST",
        body: JSON.stringify({ MediaType: kind, TMDBID: media.id }),
      });
      await waitForTorrentForceConfirmation(async () => {
        const next = await refreshTorrentAvailability(media, kind);
        return next?.torrent;
      });
    } catch (reason: any) {
      flash(
        `${t("forceStartFailed")}: ${reason?.message || ""}`.replace(/: $/, ""),
      );
    } finally {
      forcingTorrents[key] = false;
    }
  }

  async function streamInChunks(
    inputs: unknown[],
    onItem: (item: any) => void,
  ) {
    for (let index = 0; index < inputs.length; index += 100)
      await streamAvailability(inputs.slice(index, index + 100), onItem);
  }

  async function checkCards(media: ExploreMedia[]) {
    const candidates = media.filter(
      (item) => item.media_type === "movie" || item.media_type === "tv",
    );
    if (!candidates.length) return;
    await streamInChunks(
      candidates.map((item) => mediaInput(item)),
      (item: any) => {
        const key =
          item.kind === "show" ? `tv:${item.tmdbId}` : `movie:${item.tmdbId}`;
        availability[key] = {
          ...item,
          state: item.available
            ? item.kind === "show"
              ? "partial"
              : "complete"
            : "none",
        };
      },
    );
  }

  function openTorrent(item: TorrentSearchItem, kind: string) {
    if (kind !== "movie" && kind !== "tv" && kind !== "episode") return;
    torrent.value = { item, kind };
  }

  async function onTorrentAccepted(payload: TorrentAccepted) {
    await refreshTorrentAvailability(payload.item, payload.kind);
  }

  let pollTimer: number | undefined;
  let pollBusy = false;
  async function pollTorrentDownloads() {
    if (pollBusy || !detail.value) return;
    const candidates: { item: any; kind: string }[] = [];
    const detailKey = `${currentKind.value}:${detail.value.id}`;
    if (
      (currentKind.value === "movie" || currentKind.value === "tv") &&
      availability[detailKey]?.torrent &&
      !availability[detailKey]?.available
    )
      candidates.push({ item: detail.value, kind: currentKind.value });
    if (currentKind.value === "tv")
      for (const season of detail.value.seasons || [])
        for (const episode of season._episodes || [])
          if (
            availability[`episode:${episode.id}`]?.torrent &&
            !availability[`episode:${episode.id}`]?.available
          )
            candidates.push({ item: episode, kind: "episode" });
    if (!candidates.length) return;
    pollBusy = true;
    try {
      await Promise.all(
        candidates.map(({ item, kind }) =>
          refreshTorrentAvailability(item, kind),
        ),
      );
    } finally {
      pollBusy = false;
    }
  }

  onMounted(() => {
    pollTimer = window.setInterval(() => void pollTorrentDownloads(), 3000);
  });
  onBeforeUnmount(() => {
    if (pollTimer) window.clearInterval(pollTimer);
  });

  return {
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
    refreshTorrentAvailability,
    streamInChunks,
    torrent,
  };
}
