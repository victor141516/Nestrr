export function formatEpisodeCode(season: unknown, episode: unknown) {
  const seasonNumber = Number(season);
  const episodeNumber = Number(episode);
  if (
    !Number.isInteger(seasonNumber) ||
    seasonNumber < 0 ||
    !Number.isInteger(episodeNumber) ||
    episodeNumber < 0
  )
    return "";
  return `S${String(seasonNumber).padStart(2, "0")}E${String(episodeNumber).padStart(2, "0")}`;
}

export function torrentCategoryOptions(
  kind: string,
  categories: TorrentCategories,
) {
  const isMovie = kind === "movie";
  return {
    primary: isMovie ? categories.movie : categories.show,
    alternative: isMovie ? categories.show : categories.movie,
  };
}

export function torrentSearchDefaults(
  item: TorrentSearchMetadata,
  kind: TorrentMediaKind,
) {
  if (kind === "episode") {
    return {
      query:
        item.show ||
        item.showOriginalName ||
        item.originalShow ||
        item.original_name ||
        item.name ||
        "",
      include: [
        formatEpisodeCode(item.season_number, item.episode_number),
      ].filter(Boolean),
    };
  }
  if (kind === "movie")
    return { query: item.title || item.original_title || "", include: [] };
  return { query: item.name || item.original_name || "", include: [] };
}
export type TorrentMediaKind = "movie" | "tv" | "episode";

export type TorrentCategories = { movie: string; show: string };

export type TorrentSearchMetadata = {
  title?: string;
  original_title?: string;
  name?: string;
  original_name?: string;
  show?: string;
  showOriginalName?: string;
  originalShow?: string;
  season_number?: number;
  episode_number?: number;
};

export type TorrentSearchItem = TorrentSearchMetadata & { id: number };
