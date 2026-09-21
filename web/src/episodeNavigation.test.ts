import { describe, expect, it } from "vitest";
import { episodeNavigationCapabilities } from "./episodeNavigation";

const show = {
  seasons: [
    { season_number: 0, episode_count: 3 },
    { season_number: 1, episode_count: 2 },
    { season_number: 2, episode_count: 3 },
  ],
};

describe("episodeNavigationCapabilities", () => {
  it("does not expose navigation before the first regular episode", () => {
    expect(
      episodeNavigationCapabilities(show, {
        season_number: 1,
        episode_number: 1,
      }),
    ).toEqual({ hasPrevious: false, hasNext: true });
  });

  it("uses adjacent seasons at season boundaries", () => {
    expect(
      episodeNavigationCapabilities(show, {
        season_number: 1,
        episode_number: 2,
      }),
    ).toEqual({ hasPrevious: true, hasNext: true });
    expect(
      episodeNavigationCapabilities(show, {
        season_number: 2,
        episode_number: 3,
      }),
    ).toEqual({ hasPrevious: true, hasNext: false });
  });

  it("prefers the loaded episode order when it is available", () => {
    expect(
      episodeNavigationCapabilities(
        {
          seasons: [
            {
              season_number: 1,
              episode_count: 5,
              _episodes: [
                { episode_number: 2 },
                { episode_number: 4 },
              ],
            },
          ],
        },
        { season_number: 1, episode_number: 4 },
      ),
    ).toEqual({ hasPrevious: true, hasNext: false });
  });
});
