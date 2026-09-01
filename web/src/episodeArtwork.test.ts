import { describe, expect, it } from "vitest";
import { episodeImagePath } from "./episodeArtwork";

describe("episode artwork", () => {
  it("prefers the episode still", () => {
    expect(
      episodeImagePath(
        { still_path: "/episode.jpg" },
        { backdrop_path: "/show.jpg", poster_path: "/poster.jpg" },
      ),
    ).toBe("/episode.jpg");
  });

  it("falls back to the show backdrop and then its poster", () => {
    expect(
      episodeImagePath(
        { still_path: null },
        { backdrop_path: "/show.jpg", poster_path: "/poster.jpg" },
      ),
    ).toBe("/show.jpg");
    expect(
      episodeImagePath(
        { still_path: "" },
        { backdrop_path: null, poster_path: "/poster.jpg" },
      ),
    ).toBe("/poster.jpg");
  });

  it("returns no path only when neither episode nor show has artwork", () => {
    expect(episodeImagePath({}, {})).toBe("");
  });
});
