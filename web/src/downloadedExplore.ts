import type { ExploreMedia } from "./useExplore";
import type { MovieFilters } from "./movieFilters";
import { matchesRottenTomatoes } from "./movieFilters";

export type DownloadedLibraryItem = {
  tmdbId?: number;
  mediaType?: string;
  metadata?: ExploreMedia;
};

export type DownloadedExploreMedia = ExploreMedia & {
  _ageRating?: number | null;
  runtime?: number;
  episode_run_time?: number[];
  origin_country?: string[];
  production_countries?: { iso_3166_1?: string }[];
  production_companies?: { id?: number }[];
  rotten_tomatoes_score?: number;
  genres?: { id?: number }[];
  keywords?: { keywords?: { id?: number }[]; results?: { id?: number }[] };
};

export type ExploreSort = "popularity" | "release" | "rating" | "title";

export function downloadedMediaFromLibrary(
  entries: DownloadedLibraryItem[],
  kind: "movie" | "tv",
): DownloadedExploreMedia[] {
  const unique = new Map<number, DownloadedExploreMedia>();
  for (const entry of entries) {
    if (entry.mediaType !== kind || !entry.tmdbId) continue;
    unique.set(entry.tmdbId, {
      ...(entry.metadata || {}),
      id: entry.tmdbId,
      media_type: kind,
    });
  }
  return [...unique.values()];
}

export function downloadedFiltersNeedDetails(
  filters: MovieFilters | null,
): boolean {
  if (!filters) return false;
  return Boolean(
    filters.runtimeMin !== null ||
    filters.runtimeMax !== null ||
    filters.productionCountries.length ||
    filters.includeKeywords.length ||
    filters.excludeKeywords.length ||
    filters.includeCompanies.length ||
    filters.ageMin !== null ||
    filters.ageMax !== null,
  );
}

function year(media: DownloadedExploreMedia): number {
  return Number((media.release_date || media.first_air_date || "").slice(0, 4));
}

function genreIDs(media: DownloadedExploreMedia): number[] {
  return media.genre_ids?.length
    ? media.genre_ids
    : (media.genres || [])
        .map((genre) => Number(genre.id))
        .filter(Number.isFinite);
}

function countryCodes(media: DownloadedExploreMedia): string[] {
  return [
    ...(media.origin_country || []),
    ...(media.production_countries || []).map(
      (country) => country.iso_3166_1 || "",
    ),
  ].filter(Boolean);
}

function runtime(media: DownloadedExploreMedia): number | null {
  const value = Number(media.runtime || media.episode_run_time?.[0]);
  return Number.isFinite(value) && value > 0 ? value : null;
}

function keywordIDs(media: DownloadedExploreMedia): number[] {
  return [
    ...(media.keywords?.keywords || []),
    ...(media.keywords?.results || []),
  ]
    .map((keyword) => Number(keyword.id))
    .filter(Number.isFinite);
}

export function matchesDownloadedMedia(
  media: DownloadedExploreMedia,
  query: string,
  filters: MovieFilters | null,
): boolean {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  if (
    normalizedQuery &&
    !(media.title || media.name || "")
      .toLocaleLowerCase()
      .includes(normalizedQuery)
  )
    return false;
  if (!filters) return true;

  const releaseYear = year(media);
  if (filters.yearMin !== null && releaseYear < filters.yearMin) return false;
  if (filters.yearMax !== null && releaseYear > filters.yearMax) return false;

  const rating = Number(media.vote_average) || 0;
  if (filters.ratingMin !== null && rating < filters.ratingMin) return false;
  if (filters.ratingMax !== null && rating > filters.ratingMax) return false;
  if (!matchesRottenTomatoes(media.rotten_tomatoes_score, filters))
    return false;

  const genres = genreIDs(media);
  if (filters.includeGenres.some((id) => !genres.includes(id))) return false;
  if (filters.excludeGenres.some((id) => genres.includes(id))) return false;

  const keywords = keywordIDs(media);
  if (filters.includeKeywords.some((keyword) => !keywords.includes(keyword.id)))
    return false;
  if (filters.excludeKeywords.some((keyword) => keywords.includes(keyword.id)))
    return false;

  const companies = (media.production_companies || [])
    .map((company) => Number(company.id))
    .filter(Number.isFinite);
  if (
    filters.includeCompanies.some((company) => !companies.includes(company.id))
  )
    return false;

  if (filters.productionCountries.length) {
    const selected = new Set(
      filters.productionCountries.flatMap((value) => value.split("|")),
    );
    if (!countryCodes(media).some((code) => selected.has(code))) return false;
  }

  const minutes = runtime(media);
  if (
    filters.runtimeMin !== null &&
    (minutes === null || minutes < filters.runtimeMin)
  )
    return false;
  if (
    filters.runtimeMax !== null &&
    (minutes === null || minutes > filters.runtimeMax)
  )
    return false;

  const age = media._ageRating;
  if (
    filters.ageMin !== null &&
    age !== null &&
    age !== undefined &&
    age < filters.ageMin
  )
    return false;
  if (
    filters.ageMax !== null &&
    age !== null &&
    age !== undefined &&
    age > filters.ageMax
  )
    return false;
  return true;
}

export function sortDownloadedMedia(
  media: DownloadedExploreMedia[],
  sort: ExploreSort,
): DownloadedExploreMedia[] {
  const direction = sort === "title" ? 1 : -1;
  return [...media].sort((left, right) => {
    const leftValue =
      sort === "title"
        ? (left.title || left.name || "").toLocaleLowerCase()
        : sort === "release"
          ? left.release_date || left.first_air_date || ""
          : sort === "rating"
            ? Number(left.vote_average) || 0
            : Number(left.popularity) || 0;
    const rightValue =
      sort === "title"
        ? (right.title || right.name || "").toLocaleLowerCase()
        : sort === "release"
          ? right.release_date || right.first_air_date || ""
          : sort === "rating"
            ? Number(right.vote_average) || 0
            : Number(right.popularity) || 0;
    return leftValue < rightValue
      ? -direction
      : leftValue > rightValue
        ? direction
        : 0;
  });
}
