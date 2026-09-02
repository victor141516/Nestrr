import { describe, expect, it } from "vitest";
import { exploreQuery, parseExploreQuery } from "./discoverState";

describe("explore URL state", () => {
  it("round-trips every simultaneous movie filter", () => {
    const original = {
      type: "movie",
      q: "",
      yearMin: "1990",
      yearMax: "2010",
      ratingMin: "6.5",
      ratingMax: "9",
      rottenTomatoesMin: "70",
      rottenTomatoesMax: "95",
      runtimeMin: "80",
      runtimeMax: "150",
      ageMin: "10",
      ageMax: "17",
      includeGenres: "12,878",
      excludeGenres: "27",
      includeKeywords: "9715,155030",
      excludeKeywords: "180734",
      includeCompanies: "33,4",
      companyNames: '["Universal Pictures","Paramount Pictures"]',
      cast: "234352",
      castNames: '["Margot Robbie"]',
      directors: "1032",
      directorNames: '["Martin Scorsese"]',
      countries: "US,FR|DE",
    };
    const parsed = parseExploreQuery(original, "es-ES");
    expect(exploreQuery(parsed.kind, parsed.query, parsed.filters)).toEqual({
      type: "Movies",
      yearMin: "1990",
      yearMax: "2010",
      ratingMin: "6.5",
      ratingMax: "9",
      rottenTomatoesMin: "70",
      rottenTomatoesMax: "95",
      runtimeMin: "80",
      runtimeMax: "150",
      ageMin: "10",
      ageMax: "17",
      includeGenres: "12,878",
      excludeGenres: "27",
      includeKeywords: "9715,155030",
      excludeKeywords: "180734",
      includeCompanies: "33,4",
      companyNames: '["Universal Pictures","Paramount Pictures"]',
      cast: "234352",
      castNames: '["Margot Robbie"]',
      directors: "1032",
      directorNames: '["Martin Scorsese"]',
      countries: "US,FR|DE",
    });
  });

  it("drops runtime for series while preserving the other filters", () => {
    const parsed = parseExploreQuery({
      type: "tv",
      runtimeMin: "40",
      yearMin: "2000",
      ageMax: "14",
    });
    expect(parsed.filters?.runtimeMin).toBeNull();
    expect(exploreQuery(parsed.kind, "", parsed.filters)).toEqual({
      type: "Shows",
      yearMin: "2000",
      ageMax: "14",
    });
  });

  it("round-trips downloaded mode without inventing an advanced filter", () => {
    const parsed = parseExploreQuery({ type: "tv", downloaded: "1" });
    expect(parsed.downloaded).toBe(true);
    expect(parsed.filters).toBeNull();
    expect(exploreQuery(parsed.kind, "", null, parsed.downloaded)).toEqual({
      type: "Shows",
      downloaded: "1",
    });
  });

  it("round-trips sorting without treating it as an advanced filter", () => {
    const parsed = parseExploreQuery({ type: "tv", sort: "release" });
    expect(parsed.sort).toBe("release");
    expect(parsed.filters).toBeNull();
    expect(
      exploreQuery(
        parsed.kind,
        parsed.query,
        parsed.filters,
        parsed.downloaded,
        parsed.sort,
      ),
    ).toEqual({ type: "Shows", sort: "release" });
  });

  it("falls back to popularity for missing or invalid sorting", () => {
    expect(parseExploreQuery({}).sort).toBe("popularity");
    expect(parseExploreQuery({ sort: "unknown" }).sort).toBe("popularity");
    expect(exploreQuery("movie", "", null, false, "popularity")).toEqual({
      type: "Movies",
    });
  });

  it("uses movies for an empty URL and canonical category names after selection", () => {
    expect(parseExploreQuery({}).kind).toBe("movie");
    expect(exploreQuery("movie", "", null)).toEqual({ type: "Movies" });
    expect(parseExploreQuery({ type: "Shows" }).kind).toBe("tv");
    expect(parseExploreQuery({ type: "tv" }).kind).toBe("tv");
  });
});
