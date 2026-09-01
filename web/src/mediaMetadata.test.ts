import { describe, expect, it } from "vitest";
import {
  companyMetadataItems,
  countryMetadataItems,
  genreExploreRoute,
  genreMetadataItems,
  keywordMetadataItems,
} from "./mediaMetadata";

describe("media metadata", () => {
  it("builds individual and combined genre routes", () => {
    const genres = genreMetadataItems("movie", [
      { id: 878, name: "Science Fiction" },
      { id: 12, name: "Adventure" },
    ]);

    expect(genres[0]?.to).toEqual({
      path: "/",
      query: { type: "Movies", includeGenres: "878" },
    });
    expect(
      genreExploreRoute(
        "movie",
        genres.map((genre) => Number(genre.id)),
      ),
    ).toEqual({
      path: "/",
      query: { type: "Movies", includeGenres: "878,12" },
    });
  });

  it("uses the current media type for keyword navigation", () => {
    expect(
      keywordMetadataItems("tv", [{ id: 9715, name: "superhero" }]),
    ).toEqual([
      {
        id: 9715,
        label: "superhero",
        to: {
          path: "/",
          query: { type: "Shows", includeKeywords: "9715" },
        },
      },
    ]);
  });

  it("normalizes companies and countries without duplicates", () => {
    expect(
      companyMetadataItems("movie", [
        { id: 1, name: "Studio" },
        { id: 1, name: "Studio" },
      ]),
    ).toEqual([
      {
        id: 1,
        label: "Studio",
        to: {
          path: "/",
          query: {
            type: "Movies",
            includeCompanies: "1",
            companyNames: '["Studio"]',
          },
        },
      },
    ]);
    expect(
      countryMetadataItems("tv", [
        { iso_3166_1: "us", name: "United States" },
        { iso_3166_1: "US", name: "United States" },
      ]),
    ).toEqual([
      {
        id: "US",
        label: "United States",
        to: { path: "/", query: { type: "Shows", countries: "US" } },
      },
    ]);
  });
});
