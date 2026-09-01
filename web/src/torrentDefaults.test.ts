import { describe, expect, it } from "vitest";
import {
  formatEpisodeCode,
  torrentCategoryOptions,
  torrentSearchDefaults,
} from "./torrentDefaults";

describe("torrent search defaults", () => {
  it("uses the localized show name and puts the episode code in include words", () => {
    expect(
      torrentSearchDefaults(
        {
          name: "Freedom Day",
          show: "El silo",
          showOriginalName: "Silo",
          season_number: 1,
          episode_number: 1,
        },
        "episode",
      ),
    ).toEqual({ query: "El silo", include: ["S01E01"] });
  });

  it("falls back to the localized show name instead of the episode title", () => {
    expect(
      torrentSearchDefaults(
        {
          name: "El día de la libertad",
          show: "Silo",
          season_number: 2,
          episode_number: 10,
        },
        "episode",
      ),
    ).toEqual({ query: "Silo", include: ["S02E10"] });
  });

  it("prefers localized movie and show titles", () => {
    expect(
      torrentSearchDefaults(
        {
          title: "Mad Max: Furia en la carretera",
          original_title: "Mad Max: Fury Road",
        },
        "movie",
      ),
    ).toEqual({ query: "Mad Max: Furia en la carretera", include: [] });
    expect(
      torrentSearchDefaults(
        { name: "La casa del dragón", original_name: "House of the Dragon" },
        "tv",
      ),
    ).toEqual({ query: "La casa del dragón", include: [] });
  });

  it("zero-pads season and episode numbers", () => {
    expect(formatEpisodeCode(3, 7)).toBe("S03E07");
  });

  it("orders the default and alternative categories by media type", () => {
    const categories = { movie: "Movies", show: "Shows" };
    expect(torrentCategoryOptions("movie", categories)).toEqual({
      primary: "Movies",
      alternative: "Shows",
    });
    expect(torrentCategoryOptions("tv", categories)).toEqual({
      primary: "Shows",
      alternative: "Movies",
    });
    expect(torrentCategoryOptions("episode", categories)).toEqual({
      primary: "Shows",
      alternative: "Movies",
    });
  });
});
