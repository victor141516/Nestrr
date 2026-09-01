import { describe, expect, it } from "vitest";
import {
  activeMovieFilterCount,
  createMovieFilters,
  cycleGenre,
  discoverParams,
  genreState,
  matchesAge,
  matchesRottenTomatoes,
  movieDiscoverParams,
  normalizeMovieFilters,
  productionCountryOptions,
  usAgeFromDetails,
} from "./movieFilters";

describe("movie filters", () => {
  it("combines every supported filter into a TMDB discover request", () => {
    const filters = {
      ...createMovieFilters("es-ES"),
      yearMin: 1980,
      yearMax: 1999,
      ratingMin: 6.5,
      ratingMax: 9,
      rottenTomatoesMin: 70,
      rottenTomatoesMax: 95,
      runtimeMin: 80,
      runtimeMax: 150,
      includeGenres: [12, 878],
      excludeGenres: [27],
      includeKeywords: [{ id: 9715, name: "superhero" }],
      excludeKeywords: [{ id: 180734, name: "masked superhero" }],
      includeCompanies: [{ id: 33, name: "Universal Pictures" }],
      productionCountries: ["ES", "FR|DE"],
      ageMin: 7,
      ageMax: 18,
    };
    const params = movieDiscoverParams(filters, "es-ES", 3);
    expect(Object.fromEntries(params)).toMatchObject({
      language: "es-ES",
      page: "3",
      "primary_release_date.gte": "1980-01-01",
      "primary_release_date.lte": "1999-12-31",
      "vote_average.gte": "6.5",
      "vote_average.lte": "9",
      "with_runtime.gte": "80",
      "with_runtime.lte": "150",
      with_genres: "12,878",
      without_genres: "27",
      with_keywords: "9715",
      without_keywords: "180734",
      with_companies: "33",
      with_origin_country: "ES|FR|DE",
    });
    expect(params.has("rottenTomatoesMin")).toBe(false);
    expect(params.has("rottenTomatoesMax")).toBe(false);
    expect(activeMovieFilterCount(filters)).toBe(11);
    expect(matchesRottenTomatoes(91, filters)).toBe(true);
    expect(matchesRottenTomatoes(undefined, filters)).toBe(false);
  });

  it("builds equivalent series filters without movie runtime", () => {
    const filters = {
      ...createMovieFilters(),
      yearMin: 2000,
      yearMax: 2020,
      runtimeMin: 40,
      includeGenres: [18],
      productionCountries: ["US"],
    };
    const params = discoverParams(filters, "es-ES", 2, "tv");
    expect(params.get("first_air_date.gte")).toBe("2000-01-01");
    expect(params.get("primary_release_date.gte")).toBeNull();
    expect(params.get("with_runtime.gte")).toBeNull();
    expect(params.get("with_genres")).toBe("18");
  });

  it("maps US ratings to universal ages and retains unrated titles", () => {
    const movie = {
      release_dates: {
        results: [
          { iso_3166_1: "US", release_dates: [{ certification: "PG-13" }] },
        ],
      },
    };
    const show = {
      content_ratings: { results: [{ iso_3166_1: "US", rating: "TV-14" }] },
    };
    expect(usAgeFromDetails(movie, "movie")).toBe(13);
    expect(usAgeFromDetails(show, "tv")).toBe(14);
    const filters = { ...createMovieFilters(), ageMin: 10, ageMax: 16 };
    expect(matchesAge(13, filters)).toBe(true);
    expect(matchesAge(17, filters)).toBe(false);
    expect(matchesAge(null, filters)).toBe(true);
  });

  it("normalizes reversed numeric ranges and conflicting genres", () => {
    const normalized = normalizeMovieFilters({
      ...createMovieFilters(),
      yearMin: 2020,
      yearMax: 1990,
      ratingMin: 11,
      ratingMax: -1,
      rottenTomatoesMin: 101,
      rottenTomatoesMax: -1,
      includeGenres: [12, 27],
      excludeGenres: [27],
      includeKeywords: [
        { id: 9715, name: "superhero" },
        { id: 9715, name: "superhero" },
        { id: 180734, name: "masked superhero" },
      ],
      excludeKeywords: [{ id: 180734, name: "masked superhero" }],
    });
    expect([normalized.yearMin, normalized.yearMax]).toEqual([1990, 2020]);
    expect([normalized.ratingMin, normalized.ratingMax]).toEqual([null, null]);
    expect([
      normalized.rottenTomatoesMin,
      normalized.rottenTomatoesMax,
    ]).toEqual([null, null]);
    expect(normalized.includeGenres).toEqual([12]);
    expect(normalized.includeKeywords).toEqual([
      { id: 9715, name: "superhero" },
    ]);
  });

  it("treats full slider extents as inactive filters", () => {
    const normalized = normalizeMovieFilters({
      ...createMovieFilters(),
      yearMin: 1870,
      yearMax: new Date().getFullYear() + 5,
      ratingMin: 0,
      ratingMax: 10,
      rottenTomatoesMin: 0,
      rottenTomatoesMax: 100,
      runtimeMin: 0,
      runtimeMax: 600,
      ageMin: 0,
      ageMax: 18,
    });
    expect(activeMovieFilterCount(normalized)).toBe(0);
  });

  it("cycles a genre from neutral to include to exclude to neutral", () => {
    let filters = createMovieFilters();
    filters = cycleGenre(filters, 12);
    expect(genreState(filters, 12)).toBe("include");
    filters = cycleGenre(filters, 12);
    expect(genreState(filters, 12)).toBe("exclude");
    filters = cycleGenre(filters, 12);
    expect(genreState(filters, 12)).toBe("neutral");
  });

  it("keeps main production countries independent and regional groups non-overlapping", () => {
    const countries = [
      "US",
      "FR",
      "GB",
      "DE",
      "JP",
      "ES",
      "IN",
      "BR",
      "CA",
      "IT",
      "CN",
      "MX",
      "RU",
      "KR",
      "PL",
      "UA",
      "HK",
      "TW",
      "AU",
    ].map((code) => ({ iso_3166_1: code, english_name: code }));
    const options = productionCountryOptions(countries);
    expect(options.length).toBeLessThanOrEqual(15);
    expect(
      options
        .filter((option) => option.kind === "country")
        .map((option) => option.countryCode),
    ).toEqual(["US", "FR", "GB", "DE", "JP", "ES", "IN", "BR"]);
    expect(
      options
        .find((option) => option.id === "region:easternEurope")
        ?.value.split("|"),
    ).toEqual(["PL", "UA"]);
    expect(
      options
        .find((option) => option.id === "region:eastAsia")
        ?.value.split("|"),
    ).toEqual(["CN", "HK", "KR", "TW"]);
    const allCodes = options.flatMap((option) => option.value.split("|"));
    expect(new Set(allCodes).size).toBe(countries.length);
  });
});
