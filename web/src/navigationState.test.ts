import { describe, expect, it } from "vitest";
import {
  episodeCode,
  playbackQuery,
  playbackRequested,
  routeLoadQuery,
  withoutPlayback,
} from "./navigationState";

describe("playback navigation state", () => {
  it("adds a shareable movie playback flag without losing other query state", () => {
    const query = playbackQuery({ source: "library" }, {}, "movie");
    expect(query).toEqual({ source: "library", play: "1" });
    expect(playbackRequested(query)).toBe(true);
    expect(withoutPlayback(query)).toEqual({ source: "library" });
  });

  it("stores the selected season and episode for episode playback", () => {
    expect(
      playbackQuery({}, { season_number: 2, episode_number: 7 }, "episode"),
    ).toEqual({ play: "1", season: "2", episode: "7" });
    expect(episodeCode({ season_number: 2, episode_number: 7 })).toBe("S02E07");
  });

  it("does not reload a TV detail when only playback selection changes", () => {
    expect(
      routeLoadQuery("/tv/42", {
        play: "1",
        season: "2",
        episode: "7",
        source: "library",
      }),
    ).toEqual({ source: "library" });
  });
});
