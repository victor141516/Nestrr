import {
  onBeforeUnmount,
  reactive,
  ref,
  type ComputedRef,
  type Ref,
} from "vue";
import type { Router } from "vue-router";
import { api } from "./api";
import type { TranslationKey } from "./i18n";
import {
  canonicalListMediaType,
  mediaListItem,
  membershipKey,
  type ListMembership,
  type MediaList,
  type MediaListItem,
} from "./mediaLists";

type PendingListRemoval = {
  listId: number;
  mediaType: "movie" | "tv";
  tmdbId: number;
  addedAt: string;
  timer: number;
};

type Options = {
  router: Router;
  pathOnly: ComputedRef<string>;
  progress: Ref<any[]>;
  detail: Ref<any>;
  loadSeasonEpisodes: (season: any) => Promise<void>;
  t: (key: TranslationKey) => string;
  flash: (message: string) => void;
};

export function useMediaLists(options: Options) {
  const { router, pathOnly, progress, detail, loadSeasonEpisodes, t, flash } =
    options;
  const mediaLists = ref<MediaList[]>([]);
  const listMemberships = ref<ListMembership[]>([]);
  const activeList = ref<MediaList | null>(null);
  const activeListItems = ref<MediaListItem[]>([]);
  const listMediaType = ref<"all" | "movie" | "tv">("all");
  const listSort = ref("addedAt");
  const listDirection = ref<"asc" | "desc">("desc");
  const listEditor = ref<{
    mode: "create" | "rename" | "delete";
    list?: MediaList;
    name: string;
  } | null>(null);
  const pendingListRemovals = reactive<Record<string, PendingListRemoval>>({});
  const listMutationBusy = reactive<Record<string, boolean>>({});

  function progressFor(kind: string, id: number) {
    return progress.value.find(
      (item) => item.mediaType === kind && item.tmdbId === id,
    );
  }

  function membershipsFor(kind: string, id: number) {
    const key = membershipKey(kind, id);
    return listMemberships.value.filter(
      (membership) =>
        membershipKey(membership.mediaType, membership.tmdbId) === key,
    );
  }
  function listMutationKey(listId: number, kind: string, id: number) {
    return `${listId}:${canonicalListMediaType(kind)}:${id}`;
  }
  function pendingRemovalFor(item: MediaListItem) {
    if (!activeList.value) return false;
    return Boolean(
      pendingListRemovals[
        listMutationKey(activeList.value.id, item.mediaType, item.tmdbId)
      ],
    );
  }
  function busyListFor(item: MediaListItem) {
    const membership = mediaLists.value.find(
      (list) =>
        listMutationBusy[listMutationKey(list.id, item.mediaType, item.tmdbId)],
    );
    return membership?.id || 0;
  }
  function clearPendingListRemoval(key: string) {
    const pending = pendingListRemovals[key];
    if (!pending) return;
    window.clearTimeout(pending.timer);
    delete pendingListRemovals[key];
  }
  function clearPendingListRemovals() {
    for (const key of Object.keys(pendingListRemovals))
      clearPendingListRemoval(key);
  }
  function scheduleListRemoval(
    list: MediaList,
    item: MediaListItem,
    addedAt: string,
  ) {
    const key = listMutationKey(list.id, item.mediaType, item.tmdbId);
    clearPendingListRemoval(key);
    pendingListRemovals[key] = {
      listId: list.id,
      mediaType: item.mediaType,
      tmdbId: item.tmdbId,
      addedAt,
      timer: window.setTimeout(() => {
        delete pendingListRemovals[key];
        if (activeList.value?.id !== list.id) return;
        activeListItems.value = activeListItems.value.filter(
          (candidate) =>
            candidate.mediaType !== item.mediaType ||
            candidate.tmdbId !== item.tmdbId,
        );
      }, 2000),
    };
  }
  function syncActiveListCount() {
    if (!activeList.value) return;
    const current = mediaLists.value.find(
      (list) => list.id === activeList.value?.id,
    );
    if (current) activeList.value.itemCount = current.itemCount;
  }
  function sortActiveItems(items: MediaListItem[]) {
    const multiplier = listDirection.value === "asc" ? 1 : -1;
    const value = (item: MediaListItem): string | number => {
      if (listSort.value === "title") return item.title.toLocaleLowerCase();
      if (listSort.value === "releaseDate") return item.releaseDate || "";
      if (listSort.value === "rating")
        return (item.voteCount || 0) >= 100 ? item.voteAverage || 0 : -1;
      if (listSort.value === "popularity") return item.popularity || 0;
      if (listSort.value === "runtime") return item.runtime || 0;
      return item.addedAt || "";
    };
    return [...items].sort((left, right) => {
      const a = value(left);
      const b = value(right);
      return (
        multiplier *
        (typeof a === "string" && typeof b === "string"
          ? a.localeCompare(b)
          : Number(a) - Number(b))
      );
    });
  }
  function acceptBulkImportedItem(
    listId: number,
    item: MediaListItem,
    added: boolean,
  ) {
    const key = membershipKey(item.mediaType, item.tmdbId);
    const existingIndex = activeListItems.value.findIndex(
      (candidate) =>
        membershipKey(candidate.mediaType, candidate.tmdbId) === key,
    );
    if (
      activeList.value?.id === listId &&
      (listMediaType.value === "all" || listMediaType.value === item.mediaType)
    ) {
      const next = [...activeListItems.value];
      if (existingIndex >= 0) next[existingIndex] = item;
      else next.push(item);
      activeListItems.value = sortActiveItems(next);
    }
    if (!added) return;
    if (
      !listMemberships.value.some(
        (membership) =>
          membership.listId === listId &&
          membershipKey(membership.mediaType, membership.tmdbId) === key,
      )
    ) {
      listMemberships.value = [
        {
          listId,
          mediaType: item.mediaType,
          tmdbId: item.tmdbId,
          addedAt: item.addedAt || new Date().toISOString(),
        },
        ...listMemberships.value,
      ];
    }
    const list = mediaLists.value.find((candidate) => candidate.id === listId);
    if (list) list.itemCount += 1;
    if (activeList.value?.id === listId)
      activeList.value.itemCount =
        list?.itemCount ?? activeList.value.itemCount + 1;
  }
  async function finishBulkImport() {
    const [listsDoc, membershipsDoc] = await Promise.all([
      api<any>("/lists"),
      api<any>("/list-memberships"),
    ]);
    mediaLists.value = listsDoc.lists || [];
    listMemberships.value = membershipsDoc.items || [];
    syncActiveListCount();
  }
  function systemMembershipsFor(kind: string, id: number) {
    return membershipsFor(kind, id)
      .map((membership) => membership.systemKey)
      .filter(Boolean) as string[];
  }
  function inSystemList(key: string, kind: string, id: number) {
    return systemMembershipsFor(kind, id).includes(key);
  }
  function startedFor(kind: string, id: number) {
    if (kind === "movie") {
      const item = progressFor("movie", id);
      return Boolean(item?.started && !item?.watched);
    }
    return progress.value.some(
      (item) =>
        item.mediaType === "episode" &&
        item.showTmdbId === id &&
        item.started &&
        !item.watched,
    );
  }
  function systemListName(key: string) {
    return key === "favorites"
      ? t("favorites")
      : key === "watch-later"
        ? t("watchLater")
        : t("watched");
  }

  async function loadListState() {
    const [listsDoc, membershipsDoc, progressDoc] = await Promise.all([
      api<any>("/lists"),
      api<any>("/list-memberships"),
      api<any>("/progress"),
    ]);
    mediaLists.value = listsDoc.lists || [];
    listMemberships.value = membershipsDoc.items || [];
    progress.value = progressDoc.items || [];
  }
  async function loadListsPage(path = pathOnly.value) {
    clearPendingListRemovals();
    activeList.value = null;
    activeListItems.value = [];
    const match = path.match(/^\/lists\/(\d+)$/);
    if (!match) return;
    const id = Number(match[1]);
    const params = new URLSearchParams({
      sort: listSort.value,
      direction: listDirection.value,
    });
    if (listMediaType.value !== "all")
      params.set("mediaType", listMediaType.value);
    const result = await api<any>(`/lists/${id}/items?${params}`);
    activeList.value = result.list;
    activeListItems.value = result.items || [];
  }

  async function setEpisodeWatched(episode: any, watched: boolean) {
    const p = await api<any>(`/progress/episode/${episode.id}/watched`, {
      method: "POST",
      body: JSON.stringify({
        watched,
        showTmdbId: detail.value?.id,
        seasonNumber: episode.season_number,
        episodeNumber: episode.episode_number,
      }),
    });
    const index = progress.value.findIndex(
      (x) => x.mediaType === "episode" && x.tmdbId === episode.id,
    );
    if (index >= 0) progress.value[index] = p;
    else progress.value.unshift(p);
    await loadListState();
    flash(watched ? t("markedWatched") : t("markedUnwatched"));
  }
  async function setSeasonWatched(season: any) {
    if (!season._loaded) await loadSeasonEpisodes(season);
    const result = await api<any>("/progress/episodes/watched", {
      method: "POST",
      body: JSON.stringify({
        watched: true,
        episodes: (season._episodes || []).map((episode: any) => ({
          tmdbId: episode.id,
          showTmdbId: detail.value?.id,
          seasonNumber: episode.season_number,
          episodeNumber: episode.episode_number,
        })),
      }),
    });
    for (const item of result.items || []) {
      const index = progress.value.findIndex(
        (value) =>
          value.mediaType === "episode" && value.tmdbId === item.tmdbId,
      );
      if (index >= 0) progress.value[index] = item;
      else progress.value.unshift(item);
    }
    await loadListState();
    flash(t("seasonMarkedWatched"));
  }
  async function toggleListMembership(
    list: MediaList,
    media: any,
    kind: string,
  ) {
    const canonicalKind = canonicalListMediaType(kind);
    const id = media.id ?? media.tmdbId;
    const mutationKey = listMutationKey(list.id, canonicalKind, id);
    if (listMutationBusy[mutationKey]) return;
    const membership = membershipsFor(canonicalKind, id).find(
      (membership) => membership.listId === list.id,
    );
    const pending = pendingListRemovals[mutationKey];
    listMutationBusy[mutationKey] = true;
    try {
      if (membership) {
        await api(`/lists/${list.id}/items/${canonicalKind}/${id}`, {
          method: "DELETE",
        });
        listMemberships.value = listMemberships.value.filter(
          (item) =>
            !(
              item.listId === list.id &&
              item.mediaType === canonicalKind &&
              item.tmdbId === id
            ),
        );
        if (activeList.value?.id === list.id)
          scheduleListRemoval(
            list,
            media as MediaListItem,
            membership.addedAt || media.addedAt || "",
          );
      } else {
        const snapshot = mediaListItem(media, canonicalKind);
        if (pending?.addedAt) snapshot.addedAt = pending.addedAt;
        if (pending) clearPendingListRemoval(mutationKey);
        try {
          await api(`/lists/${list.id}/items`, {
            method: "POST",
            body: JSON.stringify(snapshot),
          });
        } catch (cause) {
          if (pending && activeList.value?.id === list.id)
            activeListItems.value = activeListItems.value.filter(
              (candidate) =>
                candidate.mediaType !== canonicalKind ||
                candidate.tmdbId !== id,
            );
          throw cause;
        }
      }
      await loadListState();
      syncActiveListCount();
      flash(membership ? t("removedFromList") : t("addedToList"));
    } finally {
      delete listMutationBusy[mutationKey];
    }
  }
  async function toggleSystemList(key: string, media: any, kind: string) {
    const list = mediaLists.value.find((item) => item.systemKey === key);
    if (list) await toggleListMembership(list, media, kind);
  }
  async function toggleCustomList(list: MediaList, media: any, kind: string) {
    await toggleListMembership(list, media, kind);
  }
  async function toggleRowList(list: MediaList, item: MediaListItem) {
    await toggleListMembership(list, item, item.mediaType);
  }
  async function saveListEditor() {
    const editor = listEditor.value;
    if (!editor) return;
    if (editor.mode === "delete" && editor.list) {
      await api(`/lists/${editor.list.id}`, { method: "DELETE" });
      if (activeList.value?.id === editor.list.id) await router.push("/lists");
    } else if (editor.mode === "rename" && editor.list) {
      await api(`/lists/${editor.list.id}`, {
        method: "PATCH",
        body: JSON.stringify({ name: editor.name.trim() }),
      });
    } else if (editor.mode === "create") {
      await api("/lists", {
        method: "POST",
        body: JSON.stringify({ name: editor.name.trim() }),
      });
    }
    listEditor.value = null;
    await loadListState();
    if (/^\/lists\/\d+$/.test(pathOnly.value)) await loadListsPage();
  }

  onBeforeUnmount(clearPendingListRemovals);

  return {
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
    setSeasonWatched,
    startedFor,
    systemListName,
    systemMembershipsFor,
    toggleCustomList,
    toggleRowList,
    toggleSystemList,
  };
}
