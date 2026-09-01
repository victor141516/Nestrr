import { describe, expect, it } from "vitest";
import { createMovieFilters } from "./movieFilters";
import {
  downloadedFiltersNeedDetails,
  downloadedMediaFromLibrary,
  matchesDownloadedMedia,
  sortDownloadedMedia,
} from "./downloadedExplore";

describe("downloaded exploration", () => {
  it("deduplicates episodes into one downloaded series card", () => {
    const items = downloadedMediaFromLibrary(
      [
        { tmdbId: 42, mediaType: "tv", metadata: { id: 42, name: "Show" } },
        { tmdbId: 42, mediaType: "tv", metadata: { id: 42, name: "Show" } },
        { tmdbId: 7, mediaType: "movie", metadata: { id: 7, title: "Film" } },
      ],
      "tv",
    );

    expect(items).toEqual([{ id: 42, name: "Show", media_type: "tv" }]);
  });

  it("applies text and simultaneous metadata filters to the library dataset", () => {
    const filters = {
      ...createMovieFilters(),
      yearMin: 2000,
      yearMax: 2010,
      ratingMin: 7,
      rottenTomatoesMin: 80,
      includeGenres: [12, 878],
      excludeGenres: [27],
      includeKeywords: [{ id: 9715, name: "superhero" }],
      excludeKeywords: [{ id: 180734, name: "masked superhero" }],
      includeCompanies: [{ id: 33, name: "Universal Pictures" }],
      productionCountries: ["US|CA"],
      runtimeMax: 150,
      ageMax: 13,
    };
    const media = {
      id: 1,
      title: "Ocean Signal",
      release_date: "2006-01-01",
      vote_average: 8.1,
      rotten_tomatoes_score: 91,
      genre_ids: [12, 878],
      production_countries: [{ iso_3166_1: "US" }],
      production_companies: [{ id: 33 }],
      runtime: 121,
      _ageRating: 13,
      keywords: { keywords: [{ id: 9715 }] },
    };

    expect(matchesDownloadedMedia(media, "signal", filters)).toBe(true);
    expect(matchesDownloadedMedia(media, "missing", filters)).toBe(false);
    expect(
      matchesDownloadedMedia(
        { ...media, rotten_tomatoes_score: undefined },
        "",
        filters,
      ),
    ).toBe(false);
    expect(
      matchesDownloadedMedia({ ...media, genre_ids: [12, 27] }, "", filters),
    ).toBe(false);
    expect(downloadedFiltersNeedDetails(filters)).toBe(true);
    expect(
      matchesDownloadedMedia(
        { ...media, keywords: { keywords: [{ id: 180734 }] } },
        "",
        filters,
      ),
    ).toBe(false);
  });

  it("sorts without mutating the downloaded collection", () => {
    const original = [
      { id: 1, title: "Zulu", popularity: 4 },
      { id: 2, title: "Alpha", popularity: 9 },
    ];

    expect(
      sortDownloadedMedia(original, "title").map((item) => item.id),
    ).toEqual([2, 1]);
    expect(
      sortDownloadedMedia(original, "popularity").map((item) => item.id),
    ).toEqual([2, 1]);
    expect(original.map((item) => item.id)).toEqual([1, 2]);
  });
});
