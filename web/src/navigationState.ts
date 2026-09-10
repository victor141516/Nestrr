export type RouteQueryValue = string | null | (string | null)[] | undefined;
export type RouteQuery = Record<string, RouteQueryValue>;

export function queryString(value: RouteQueryValue): string {
  if (Array.isArray(value)) return value[0] || "";
  return value || "";
}

export function playbackRequested(query: RouteQuery): boolean {
  return queryString(query.play) === "1";
}

export function withoutPlayback(query: RouteQuery): RouteQuery {
  const next = { ...query };
  delete next.play;
  return next;
}

export function playbackQuery(
  query: RouteQuery,
  item: {
    season_number?: number;
    episode_number?: number;
  },
  kind: string,
): RouteQuery {
  const next: RouteQuery = { ...query, play: "1" };
  if (kind === "episode") {
    next.season = String(item.season_number || "");
    next.episode = String(item.episode_number || "");
  }
  return next;
}

export function routeLoadQuery(path: string, query: RouteQuery): RouteQuery {
  const next = withoutPlayback(query);
  if (path.startsWith("/tv/")) {
    delete next.season;
    delete next.episode;
  }
  if (path === "/recommendations") delete next.type;
  if (/^\/lists\/\d+$/.test(path)) {
    delete next.type;
    delete next.sort;
    delete next.direction;
  }
  return next;
}

export function episodeCode(item: {
  season_number?: number;
  episode_number?: number;
}): string {
  if (!item.season_number || !item.episode_number) return "";
  return `S${String(item.season_number).padStart(2, "0")}E${String(item.episode_number).padStart(2, "0")}`;
}
