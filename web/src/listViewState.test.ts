import { describe, expect, it } from "vitest";
import {
  isCanonicalListViewQuery,
  listViewQuery,
  parseListViewQuery,
} from "./listViewState";

describe("list view URL state", () => {
  it("defaults to movies and the existing list ordering", () => {
    expect(parseListViewQuery({})).toEqual({
      mediaType: "movie",
      sort: "addedAt",
      direction: "desc",
    });
  });

  it("parses shows, sorting, and direction from the URL", () => {
    expect(
      parseListViewQuery({
        type: "Shows",
        sort: "rating",
        direction: "asc",
      }),
    ).toEqual({ mediaType: "tv", sort: "rating", direction: "asc" });
  });

  it("serializes a canonical URL without dropping unrelated parameters", () => {
    const state = {
      mediaType: "tv" as const,
      sort: "title" as const,
      direction: "asc" as const,
    };
    const query = listViewQuery({ source: "lists" }, state);
    expect(query).toEqual({
      source: "lists",
      type: "Shows",
      sort: "title",
      direction: "asc",
    });
    expect(isCanonicalListViewQuery(query, state)).toBe(true);
  });
});
