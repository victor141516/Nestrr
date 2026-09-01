export type EpisodeArtwork = {
  still_path?: string | null;
};

export type ShowArtwork = {
  backdrop_path?: string | null;
  poster_path?: string | null;
};

export function episodeImagePath(
  episode: EpisodeArtwork,
  show: ShowArtwork | null | undefined,
) {
  return episode.still_path || show?.backdrop_path || show?.poster_path || "";
}
