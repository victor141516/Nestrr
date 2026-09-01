export type EpisodeProgressContext = {
  mediaType?: string;
  showTmdbId?: number;
  seasonNumber?: number;
  positionSeconds?: number;
  watched?: boolean;
  updatedAt?: string;
};

export function initialOpenSeason(
  showTmdbId: number,
  progress: EpisodeProgressContext[],
  routeSeason = 0,
) {
  if (routeSeason > 0) return routeSeason;
  const matching = progress
    .filter(
      (item) =>
        item.mediaType === "episode" &&
        item.showTmdbId === showTmdbId &&
        Number(item.seasonNumber) > 0,
    )
    .sort(
      (a, b) => Date.parse(b.updatedAt || "") - Date.parse(a.updatedAt || ""),
    );
  const inProgress = matching.find(
    (item) => !item.watched && Number(item.positionSeconds) > 0,
  );
  return Number(inProgress?.seasonNumber || matching[0]?.seasonNumber || 0);
}

export function episodeDisplayTitle(
  episodeLabel: string,
  episodeNumber: number,
  title: string,
) {
  return `${episodeLabel} ${episodeNumber}${title ? ` · ${title}` : ""}`;
}
