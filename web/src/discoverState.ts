import {
  createMovieFilters,
  normalizeMovieFilters,
  type MovieFilters,
} from "./movieFilters";
import type { ExploreSort } from "./downloadedExplore";
import {
  mediaTypeFromQuery,
  mediaTypeQueryValue,
  type CatalogMediaType,
} from "./mediaType";

export type ExploreKind = CatalogMediaType;
export type ExploreURLState = {
  kind: ExploreKind;
  query: string;
  filters: MovieFilters | null;
  downloaded: boolean;
  sort: ExploreSort;
};

function exploreSort(value: unknown): ExploreSort {
  return value === "release" || value === "rating" || value === "title"
    ? value
    : "popularity";
}

const numberOrNull = (value: unknown) => {
  if (typeof value !== "string" || value.trim() === "") return null;
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
};
const list = (value: unknown) =>
  typeof value === "string"
    ? value
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean)
    : [];
const keywordList = (value: unknown) =>
  list(value)
    .map(Number)
    .filter((id) => Number.isInteger(id) && id > 0)
    .map((id) => ({ id, name: String(id) }));

function namedOptionList(idsValue: unknown, namesValue: unknown) {
  const ids = list(idsValue)
    .map(Number)
    .filter((id) => Number.isInteger(id) && id > 0);
  let names: unknown[] = [];
  if (typeof namesValue === "string") {
    try {
      const parsed = JSON.parse(namesValue);
      if (Array.isArray(parsed)) names = parsed;
    } catch {
      names = [];
    }
  }
  return ids.map((id, index) => ({
    id,
    name: String(names[index] || id).trim(),
  }));
}

export function parseExploreQuery(
  query: Record<string, unknown>,
  language = "en-US",
): ExploreURLState {
  const kind = mediaTypeFromQuery(query.type);
  const filters = normalizeMovieFilters({
    ...createMovieFilters(language),
    yearMin: numberOrNull(query.yearMin),
    yearMax: numberOrNull(query.yearMax),
    ratingMin: numberOrNull(query.ratingMin),
    ratingMax: numberOrNull(query.ratingMax),
    rottenTomatoesMin: numberOrNull(query.rottenTomatoesMin),
    rottenTomatoesMax: numberOrNull(query.rottenTomatoesMax),
    runtimeMin: kind === "movie" ? numberOrNull(query.runtimeMin) : null,
    runtimeMax: kind === "movie" ? numberOrNull(query.runtimeMax) : null,
    ageMin: numberOrNull(query.ageMin),
    ageMax: numberOrNull(query.ageMax),
    includeGenres: list(query.includeGenres)
      .map(Number)
      .filter(Number.isFinite),
    excludeGenres: list(query.excludeGenres)
      .map(Number)
      .filter(Number.isFinite),
    includeKeywords: keywordList(query.includeKeywords),
    excludeKeywords: keywordList(query.excludeKeywords),
    includeCompanies: namedOptionList(
      query.includeCompanies,
      query.companyNames,
    ),
    cast: kind === "movie" ? namedOptionList(query.cast, query.castNames) : [],
    directors:
      kind === "movie"
        ? namedOptionList(query.directors, query.directorNames)
        : [],
    productionCountries: list(query.countries),
  });
  const hasFilters = Object.keys(query).some(
    (key) => !["type", "q", "downloaded", "sort"].includes(key),
  );
  return {
    kind,
    query: typeof query.q === "string" ? query.q : "",
    filters: hasFilters ? filters : null,
    downloaded: query.downloaded === "1",
    sort: exploreSort(query.sort),
  };
}

export function exploreQuery(
  kind: ExploreKind,
  query: string,
  filters: MovieFilters | null,
  downloaded = false,
  sort: ExploreSort = "popularity",
) {
  const result: Record<string, string> = {
    type: mediaTypeQueryValue(kind),
  };
  if (query.trim()) result.q = query.trim();
  if (downloaded) result.downloaded = "1";
  if (sort !== "popularity") result.sort = sort;
  if (!filters) return result;
  const f = normalizeMovieFilters(filters);
  const put = (key: string, value: number | null) => {
    if (value !== null) result[key] = String(value);
  };
  put("yearMin", f.yearMin);
  put("yearMax", f.yearMax);
  put("ratingMin", f.ratingMin);
  put("ratingMax", f.ratingMax);
  put("rottenTomatoesMin", f.rottenTomatoesMin);
  put("rottenTomatoesMax", f.rottenTomatoesMax);
  if (kind === "movie") {
    put("runtimeMin", f.runtimeMin);
    put("runtimeMax", f.runtimeMax);
  }
  put("ageMin", f.ageMin);
  put("ageMax", f.ageMax);
  if (f.includeGenres.length) result.includeGenres = f.includeGenres.join(",");
  if (f.excludeGenres.length) result.excludeGenres = f.excludeGenres.join(",");
  if (f.includeKeywords.length)
    result.includeKeywords = f.includeKeywords
      .map((keyword) => keyword.id)
      .join(",");
  if (f.excludeKeywords.length)
    result.excludeKeywords = f.excludeKeywords
      .map((keyword) => keyword.id)
      .join(",");
  if (f.includeCompanies.length) {
    result.includeCompanies = f.includeCompanies
      .map((company) => company.id)
      .join(",");
    result.companyNames = JSON.stringify(
      f.includeCompanies.map((company) => company.name),
    );
  }
  if (kind === "movie" && f.cast.length) {
    result.cast = f.cast.map((person) => person.id).join(",");
    result.castNames = JSON.stringify(f.cast.map((person) => person.name));
  }
  if (kind === "movie" && f.directors.length) {
    result.directors = f.directors.map((person) => person.id).join(",");
    result.directorNames = JSON.stringify(
      f.directors.map((person) => person.name),
    );
  }
  if (f.productionCountries.length)
    result.countries = f.productionCountries.join(",");
  return result;
}
