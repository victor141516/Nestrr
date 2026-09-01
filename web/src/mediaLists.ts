export type MediaList = {
  id: number;
  systemKey?: "favorites" | "watch-later" | "watched";
  name: string;
  color: "coral" | "amber" | "teal" | "violet";
  icon: string;
  itemCount: number;
  createdAt: string;
  updatedAt: string;
};

export type ListMembership = {
  listId: number;
  systemKey?: string;
  mediaType: "movie" | "tv";
  tmdbId: number;
  addedAt: string;
};

export type MediaListItem = {
  mediaType: "movie" | "tv";
  tmdbId: number;
  title: string;
  originalTitle?: string;
  overview?: string;
  posterPath?: string;
  releaseDate?: string;
  voteAverage?: number;
  voteCount?: number;
  popularity?: number;
  runtime?: number;
  addedAt?: string;
};

export function canonicalListMediaType(kind: string): "movie" | "tv" {
  return kind === "movie" ? "movie" : "tv";
}

export function mediaListItem(media: any, kind?: string): MediaListItem {
  const mediaType = canonicalListMediaType(
    kind || media.media_type || (media.title ? "movie" : "tv"),
  );
  return {
    mediaType,
    tmdbId: Number(media.id ?? media.tmdbId),
    title: media.title || media.name || "",
    originalTitle: media.original_title || media.original_name || "",
    overview: media.overview || "",
    posterPath: media.poster_path || media.posterPath || "",
    releaseDate:
      media.release_date || media.first_air_date || media.releaseDate || "",
    voteAverage: Number(media.vote_average ?? media.voteAverage) || 0,
    voteCount: Number(media.vote_count ?? media.voteCount) || 0,
    popularity: Number(media.popularity) || 0,
    runtime: Number(media.runtime) || Number(media.episode_run_time?.[0]) || 0,
  };
}

export function membershipKey(mediaType: string, tmdbId: number) {
  return `${canonicalListMediaType(mediaType)}:${tmdbId}`;
}
