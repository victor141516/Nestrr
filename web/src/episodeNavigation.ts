export type EpisodeNavigationCapabilities = {
  hasPrevious: boolean;
  hasNext: boolean;
};

type EpisodeLike = {
  season_number?: number;
  episode_number?: number;
};

type SeasonLike = {
  season_number?: number;
  episode_count?: number;
  _episodes?: EpisodeLike[];
};

function episodeCount(season: SeasonLike): number {
  const declared = Number(season.episode_count);
  if (Number.isFinite(declared) && declared > 0) return declared;
  return Math.max(
    0,
    ...(season._episodes || []).map((episode) =>
      Number(episode.episode_number || 0),
    ),
  );
}

export function episodeNavigationCapabilities(
  show: { seasons?: SeasonLike[] } | null | undefined,
  episode: EpisodeLike,
): EpisodeNavigationCapabilities {
  const seasonNumber = Number(episode.season_number || 0);
  const episodeNumber = Number(episode.episode_number || 0);
  if (seasonNumber <= 0 || episodeNumber <= 0)
    return { hasPrevious: false, hasNext: false };

  const seasons = (show?.seasons || [])
    .filter((season) => Number(season.season_number) > 0)
    .sort(
      (left, right) =>
        Number(left.season_number) - Number(right.season_number),
    );
  const currentSeason = seasons.find(
    (season) => Number(season.season_number) === seasonNumber,
  );
  const loadedEpisodes = currentSeason?._episodes || [];
  const loadedIndex = loadedEpisodes.findIndex(
    (candidate) => Number(candidate.episode_number) === episodeNumber,
  );
  const previousInSeason =
    loadedIndex >= 0 ? loadedIndex > 0 : episodeNumber > 1;
  const nextInSeason =
    loadedIndex >= 0
      ? loadedIndex < loadedEpisodes.length - 1
      : episodeNumber < episodeCount(currentSeason || {});

  return {
    hasPrevious:
      previousInSeason ||
      seasons.some(
        (season) =>
          Number(season.season_number) < seasonNumber &&
          episodeCount(season) > 0,
      ),
    hasNext:
      nextInSeason ||
      seasons.some(
        (season) =>
          Number(season.season_number) > seasonNumber &&
          episodeCount(season) > 0,
      ),
  };
}
