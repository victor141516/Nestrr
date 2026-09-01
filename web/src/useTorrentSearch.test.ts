import { effectScope } from "vue";
import { describe, expect, it, vi } from "vitest";
import { useTorrentSearch, type TorrentRequest } from "./useTorrentSearch";

describe("useTorrentSearch", () => {
  it("owns search filters and submits the selected result", async () => {
    const result = {
      ID: 7,
      Title: "Show S02E03 1080p",
      Tracker: "Indexer",
      Seeders: 42,
      MagnetURI: "magnet:?xt=test",
    };
    const request = vi.fn(async (path: string) =>
      path.startsWith("/torrents/search") ? { results: [result] } : undefined,
    ) as unknown as TorrentRequest;
    const scope = effectScope();
    const torrent = scope.run(() =>
      useTorrentSearch({
        item: {
          id: 91,
          show: "Show",
          season_number: 2,
          episode_number: 3,
        },
        kind: "episode",
        categories: { movie: "Movies", show: "Shows" },
        failureMessage: "Failed",
        request,
      }),
    )!;

    expect(torrent.state.query).toBe("Show");
    expect(torrent.state.include).toEqual(["S02E03"]);

    await torrent.search();

    expect(torrent.trackerOptions.value).toEqual(["Indexer"]);
    expect(torrent.state.trackers).toEqual(["Indexer"]);
    expect(torrent.filteredResults.value).toEqual([result]);

    const accepted = await torrent.submit(result, "Shows");

    expect(accepted).toMatchObject({ kind: "episode", category: "Shows" });
    expect(request).toHaveBeenLastCalledWith("/torrents/submit", {
      method: "POST",
      body: JSON.stringify({
        URL: "magnet:?xt=test",
        Category: "Shows",
        MediaType: "episode",
        TMDBID: 91,
      }),
    });
    scope.stop();
  });
});
