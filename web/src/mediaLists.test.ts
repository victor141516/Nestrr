import { describe, expect, it } from "vitest";
import { mediaListItem, membershipKey } from "./mediaLists";

describe("media list snapshots", () => {
  it("normalizes localized movies without losing their catalog fields", () => {
    expect(
      mediaListItem({
        id: 550,
        title: "El club de la lucha",
        original_title: "Fight Club",
        overview: "Un oficinista insomne conoce a Tyler Durden.",
        poster_path: "/poster.jpg",
        release_date: "1999-10-15",
        vote_average: 8.4,
        vote_count: 30000,
        runtime: 139,
      }),
    ).toMatchObject({
      mediaType: "movie",
      tmdbId: 550,
      title: "El club de la lucha",
      originalTitle: "Fight Club",
      overview: "Un oficinista insomne conoce a Tyler Durden.",
      runtime: 139,
    });
  });

  it("uses the same membership key for show and tv aliases", () => {
    expect(membershipKey("show", 4607)).toBe(membershipKey("tv", 4607));
  });
});
