export type MovieFilters = {
  yearMin: number | null;
  yearMax: number | null;
  ratingMin: number | null;
  ratingMax: number | null;
  rottenTomatoesMin: number | null;
  rottenTomatoesMax: number | null;
  runtimeMin: number | null;
  runtimeMax: number | null;
  includeGenres: number[];
  excludeGenres: number[];
  includeKeywords: KeywordOption[];
  excludeKeywords: KeywordOption[];
  includeCompanies: FilterOption[];
  cast: PersonOption[];
  directors: PersonOption[];
  productionCountries: string[];
  ageMin: number | null;
  ageMax: number | null;
};

export type FilterOption = { id: number; name: string };
export type KeywordOption = FilterOption;
export type PersonOption = FilterOption & {
  profilePath?: string;
  knownForDepartment?: string;
};
export type CountryOption = {
  iso_3166_1: string;
  english_name: string;
  native_name?: string;
};
export type ProductionCountryOption = {
  id: string;
  kind: "country" | "region";
  value: string;
  countryCode?: string;
  countryCount: number;
};

// Ranked by TMDB discover/movie total_results on 2026-08-29. Regional options
// deliberately exclude these countries so selecting a country and a region never overlaps.
export const PRINCIPAL_PRODUCTION_COUNTRIES = [
  "US",
  "FR",
  "GB",
  "DE",
  "JP",
  "ES",
  "IN",
  "BR",
] as const;

const PRODUCTION_REGIONS = [
  {
    id: "easternEurope",
    codes:
      "AL BA BG BY CZ EE HR HU LT LV MD ME MK PL RO RS SI SK UA XK SU XC YU CS",
  },
  {
    id: "otherEurope",
    codes:
      "AD AT BE CH CY DK FI FO GI GR IE IM IS LI LU MC MT NL NO PT SE SJ SM VA",
  },
  { id: "eastAsia", codes: "CN HK KP KR MN MO TW" },
  {
    id: "southSoutheastAsia",
    codes: "AF BD BN BT ID KH LA LK MM MV MY NP PH PK SG TH TL VN",
  },
  {
    id: "middleEastCentralAsia",
    codes:
      "AE AM AZ BH GE IL IQ IR JO KG KW KZ LB OM PS QA SA SY TJ TM TR UZ YE",
  },
  {
    id: "latinAmericaCaribbean",
    codes:
      "AG AI AN AR AW BB BO BQ BS BZ CL CO CR CU CW DM DO EC FK GD GF GP GT GY HN HT JM KN KY LC MQ MS NI PA PE PR PY SR SV SX TC TT UY VC VE VG VI",
  },
] as const;

export function productionCountryOptions(
  countries: CountryOption[],
): ProductionCountryOption[] {
  const available = new Map(
    countries
      .filter((country) => /^[A-Z]{2}$/.test(country.iso_3166_1))
      .map((country) => [country.iso_3166_1, country]),
  );
  const assigned = new Set<string>();
  const options: ProductionCountryOption[] = [];
  for (const code of PRINCIPAL_PRODUCTION_COUNTRIES) {
    if (!available.has(code)) continue;
    assigned.add(code);
    options.push({
      id: `country:${code}`,
      kind: "country",
      value: code,
      countryCode: code,
      countryCount: 1,
    });
  }
  for (const region of PRODUCTION_REGIONS) {
    const codes = region.codes
      .split(" ")
      .filter((code) => available.has(code) && !assigned.has(code));
    codes.forEach((code) => assigned.add(code));
    if (codes.length)
      options.push({
        id: `region:${region.id}`,
        kind: "region",
        value: codes.join("|"),
        countryCount: codes.length,
      });
  }
  const remaining = [...available.keys()]
    .filter((code) => !assigned.has(code))
    .sort();
  if (remaining.length)
    options.push({
      id: "region:otherRegions",
      kind: "region",
      value: remaining.join("|"),
      countryCount: remaining.length,
    });
  return options;
}

export function createMovieFilters(_language = "en-US"): MovieFilters {
  return {
    yearMin: null,
    yearMax: null,
    ratingMin: null,
    ratingMax: null,
    rottenTomatoesMin: null,
    rottenTomatoesMax: null,
    runtimeMin: null,
    runtimeMax: null,
    includeGenres: [],
    excludeGenres: [],
    includeKeywords: [],
    excludeKeywords: [],
    includeCompanies: [],
    cast: [],
    directors: [],
    productionCountries: [],
    ageMin: null,
    ageMax: null,
  };
}

export function cloneMovieFilters(filters: MovieFilters): MovieFilters {
  return {
    ...filters,
    includeGenres: [...filters.includeGenres],
    excludeGenres: [...filters.excludeGenres],
    includeKeywords: filters.includeKeywords.map((keyword) => ({ ...keyword })),
    excludeKeywords: filters.excludeKeywords.map((keyword) => ({ ...keyword })),
    includeCompanies: filters.includeCompanies.map((company) => ({
      ...company,
    })),
    cast: filters.cast.map((person) => ({ ...person })),
    directors: filters.directors.map((person) => ({ ...person })),
    productionCountries: [...filters.productionCountries],
  };
}

function orderedRange(
  min: number | null,
  max: number | null,
): [number | null, number | null] {
  if (min !== null && max !== null && min > max) return [max, min];
  return [min, max];
}

export function normalizeMovieFilters(filters: MovieFilters): MovieFilters {
  const copy = cloneMovieFilters(filters);
  [copy.yearMin, copy.yearMax] = orderedRange(copy.yearMin, copy.yearMax);
  [copy.ratingMin, copy.ratingMax] = orderedRange(
    copy.ratingMin,
    copy.ratingMax,
  );
  [copy.rottenTomatoesMin, copy.rottenTomatoesMax] = orderedRange(
    copy.rottenTomatoesMin,
    copy.rottenTomatoesMax,
  );
  [copy.runtimeMin, copy.runtimeMax] = orderedRange(
    copy.runtimeMin,
    copy.runtimeMax,
  );
  [copy.ageMin, copy.ageMax] = orderedRange(copy.ageMin, copy.ageMax);
  copy.ratingMin =
    copy.ratingMin === null ? null : Math.max(0, Math.min(10, copy.ratingMin));
  copy.ratingMax =
    copy.ratingMax === null ? null : Math.max(0, Math.min(10, copy.ratingMax));
  copy.rottenTomatoesMin =
    copy.rottenTomatoesMin === null
      ? null
      : Math.max(0, Math.min(100, copy.rottenTomatoesMin));
  copy.rottenTomatoesMax =
    copy.rottenTomatoesMax === null
      ? null
      : Math.max(0, Math.min(100, copy.rottenTomatoesMax));
  copy.runtimeMin =
    copy.runtimeMin === null ? null : Math.max(0, copy.runtimeMin);
  copy.runtimeMax =
    copy.runtimeMax === null ? null : Math.max(0, copy.runtimeMax);
  copy.ageMin =
    copy.ageMin === null ? null : Math.max(0, Math.min(18, copy.ageMin));
  copy.ageMax =
    copy.ageMax === null ? null : Math.max(0, Math.min(18, copy.ageMax));
  if (copy.yearMin !== null && copy.yearMin <= 1870) copy.yearMin = null;
  if (copy.yearMax !== null && copy.yearMax >= new Date().getFullYear() + 5)
    copy.yearMax = null;
  if (copy.ratingMin !== null && copy.ratingMin <= 0) copy.ratingMin = null;
  if (copy.ratingMax !== null && copy.ratingMax >= 10) copy.ratingMax = null;
  if (copy.rottenTomatoesMin !== null && copy.rottenTomatoesMin <= 0)
    copy.rottenTomatoesMin = null;
  if (copy.rottenTomatoesMax !== null && copy.rottenTomatoesMax >= 100)
    copy.rottenTomatoesMax = null;
  if (copy.runtimeMin !== null && copy.runtimeMin <= 0) copy.runtimeMin = null;
  if (copy.runtimeMax !== null && copy.runtimeMax >= 600)
    copy.runtimeMax = null;
  if (copy.ageMin !== null && copy.ageMin <= 0) copy.ageMin = null;
  if (copy.ageMax !== null && copy.ageMax >= 18) copy.ageMax = null;
  copy.includeGenres = [...new Set(copy.includeGenres)].filter(
    (id) => !copy.excludeGenres.includes(id),
  );
  copy.excludeGenres = [...new Set(copy.excludeGenres)];
  const normalizeKeywords = (keywords: KeywordOption[]) => {
    const seen = new Set<number>();
    return keywords
      .map((keyword) => ({
        id: Number(keyword.id),
        name: String(keyword.name || keyword.id).trim(),
      }))
      .filter((keyword) => {
        if (
          !Number.isInteger(keyword.id) ||
          keyword.id <= 0 ||
          seen.has(keyword.id)
        )
          return false;
        seen.add(keyword.id);
        return true;
      });
  };
  copy.excludeKeywords = normalizeKeywords(copy.excludeKeywords);
  const excludedKeywordIDs = new Set(
    copy.excludeKeywords.map((keyword) => keyword.id),
  );
  copy.includeKeywords = normalizeKeywords(copy.includeKeywords).filter(
    (keyword) => !excludedKeywordIDs.has(keyword.id),
  );
  copy.includeCompanies = normalizeKeywords(copy.includeCompanies);
  copy.cast = normalizeKeywords(copy.cast);
  copy.directors = normalizeKeywords(copy.directors);
  copy.productionCountries = [
    ...new Set(copy.productionCountries.filter(Boolean)),
  ];
  return copy;
}

export function activeMovieFilterCount(filters: MovieFilters | null) {
  if (!filters) return 0;
  return (
    Number(filters.yearMin !== null || filters.yearMax !== null) +
    Number(filters.ratingMin !== null || filters.ratingMax !== null) +
    Number(
      filters.rottenTomatoesMin !== null || filters.rottenTomatoesMax !== null,
    ) +
    Number(filters.runtimeMin !== null || filters.runtimeMax !== null) +
    Number(filters.includeGenres.length > 0) +
    Number(filters.excludeGenres.length > 0) +
    Number(filters.includeKeywords.length > 0) +
    Number(filters.excludeKeywords.length > 0) +
    Number(filters.includeCompanies.length > 0) +
    Number(filters.cast.length > 0) +
    Number(filters.directors.length > 0) +
    Number(filters.productionCountries.length > 0) +
    Number(filters.ageMin !== null || filters.ageMax !== null)
  );
}

export function matchesRottenTomatoes(
  score: unknown,
  filters: MovieFilters,
): boolean {
  if (filters.rottenTomatoesMin === null && filters.rottenTomatoesMax === null)
    return true;
  const value = Number(score);
  if (!Number.isFinite(value)) return false;
  if (filters.rottenTomatoesMin !== null && value < filters.rottenTomatoesMin)
    return false;
  if (filters.rottenTomatoesMax !== null && value > filters.rottenTomatoesMax)
    return false;
  return true;
}

export function discoverParams(
  filters: MovieFilters,
  language: string,
  page = 1,
  kind: "movie" | "tv" = "movie",
  sortBy = "popularity.desc",
) {
  const normalized = normalizeMovieFilters(filters);
  const params = new URLSearchParams({
    language,
    page: String(page),
    include_adult: "false",
    sort_by: sortBy,
  });
  if (kind === "movie") params.set("include_video", "false");
  const dateKey = kind === "movie" ? "primary_release_date" : "first_air_date";
  if (normalized.yearMin !== null)
    params.set(`${dateKey}.gte`, `${normalized.yearMin}-01-01`);
  if (normalized.yearMax !== null)
    params.set(`${dateKey}.lte`, `${normalized.yearMax}-12-31`);
  if (normalized.ratingMin !== null)
    params.set("vote_average.gte", String(normalized.ratingMin));
  if (normalized.ratingMax !== null)
    params.set("vote_average.lte", String(normalized.ratingMax));
  if (kind === "movie" && normalized.runtimeMin !== null)
    params.set("with_runtime.gte", String(normalized.runtimeMin));
  if (kind === "movie" && normalized.runtimeMax !== null)
    params.set("with_runtime.lte", String(normalized.runtimeMax));
  if (normalized.includeGenres.length)
    params.set("with_genres", normalized.includeGenres.join(","));
  if (normalized.excludeGenres.length)
    params.set("without_genres", normalized.excludeGenres.join(","));
  if (normalized.includeKeywords.length)
    params.set(
      "with_keywords",
      normalized.includeKeywords.map((keyword) => keyword.id).join(","),
    );
  if (normalized.excludeKeywords.length)
    params.set(
      "without_keywords",
      normalized.excludeKeywords.map((keyword) => keyword.id).join(","),
    );
  if (normalized.includeCompanies.length)
    params.set(
      "with_companies",
      normalized.includeCompanies.map((company) => company.id).join(","),
    );
  if (kind === "movie" && normalized.cast.length)
    params.set(
      "with_cast",
      normalized.cast.map((person) => person.id).join(","),
    );
  if (kind === "movie" && normalized.directors.length)
    params.set(
      "directors",
      normalized.directors.map((person) => person.id).join(","),
    );
  if (normalized.productionCountries.length) {
    const countryCodes = [
      ...new Set(
        normalized.productionCountries
          .flatMap((value) => value.split("|"))
          .filter(Boolean),
      ),
    ];
    params.set("with_origin_country", countryCodes.join("|"));
  }
  return params;
}

export const movieDiscoverParams = (
  filters: MovieFilters,
  language: string,
  page = 1,
) => discoverParams(filters, language, page, "movie");

const MOVIE_US_AGES: Record<string, number> = {
  G: 0,
  PG: 10,
  "PG-13": 13,
  R: 17,
  "NC-17": 18,
};
const TV_US_AGES: Record<string, number> = {
  "TV-Y": 0,
  "TV-Y7": 7,
  "TV-G": 10,
  "TV-PG": 12,
  "TV-14": 14,
  "TV-MA": 18,
};

export function usAgeFromDetails(details: any, kind: "movie" | "tv") {
  if (kind === "movie") {
    const releases =
      details?.release_dates?.results?.find(
        (item: any) => item.iso_3166_1 === "US",
      )?.release_dates || [];
    const ages = releases
      .map((item: any) => MOVIE_US_AGES[item.certification])
      .filter((age: unknown) => typeof age === "number");
    return ages.length ? Math.max(...ages) : null;
  }
  const rating = details?.content_ratings?.results?.find(
    (item: any) => item.iso_3166_1 === "US",
  )?.rating;
  return typeof TV_US_AGES[rating] === "number" ? TV_US_AGES[rating] : null;
}

export function matchesAge(age: number | null, filters: MovieFilters) {
  if (age === null) return true;
  return (
    (filters.ageMin === null || age >= filters.ageMin) &&
    (filters.ageMax === null || age <= filters.ageMax)
  );
}

export function genreState(filters: MovieFilters, genreID: number) {
  if (filters.includeGenres.includes(genreID)) return "include";
  if (filters.excludeGenres.includes(genreID)) return "exclude";
  return "neutral";
}

export function cycleGenre(
  filters: MovieFilters,
  genreID: number,
): MovieFilters {
  const copy = cloneMovieFilters(filters);
  const state = genreState(copy, genreID);
  copy.includeGenres = copy.includeGenres.filter((id) => id !== genreID);
  copy.excludeGenres = copy.excludeGenres.filter((id) => id !== genreID);
  if (state === "neutral") copy.includeGenres.push(genreID);
  if (state === "include") copy.excludeGenres.push(genreID);
  return copy;
}
