import { describe, expect, it } from "vitest";
import { episodeDisplayTitle, initialOpenSeason } from "./seasonState";

describe("season detail state", () => {
  it("keeps every season collapsed when the show has no progress", () => {
    expect(initialOpenSeason(42, [])).toBe(0);
  });

  it("opens only the most recently active season", () => {
    expect(
      initialOpenSeason(42, [
        {
          mediaType: "episode",
          showTmdbId: 42,
          seasonNumber: 1,
          positionSeconds: 100,
          updatedAt: "2026-01-01T00:00:00Z",
        },
        {
          mediaType: "episode",
          showTmdbId: 42,
          seasonNumber: 3,
          positionSeconds: 20,
          updatedAt: "2026-02-01T00:00:00Z",
        },
      ]),
    ).toBe(3);
  });

  it("gives a playback URL season precedence", () => {
    expect(
      initialOpenSeason(
        42,
        [
          {
            mediaType: "episode",
            showTmdbId: 42,
            seasonNumber: 3,
            positionSeconds: 20,
          },
        ],
        5,
      ),
    ).toBe(5);
  });

  it("uses a human episode label without zero padding", () => {
    expect(episodeDisplayTitle("Episodio", 2, "El comienzo")).toBe(
      "Episodio 2 · El comienzo",
    );
  });
});
