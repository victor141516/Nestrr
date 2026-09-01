import { describe, expect, it } from "vitest";
import { filterTorrentResults } from "./torrentFilters";

const results = [
  { Title: "Mad Max Fury Road 2015 1080p, HDR", Seeders: 80, Tracker: "A" },
  { Title: "Mad Max Fury Road 2015 720p", Seeders: 40, Tracker: "A" },
  { Title: "Mad Max 1979 1080p, HDR", Seeders: 100, Tracker: "B" },
];

describe("torrent result filters", () => {
  it("requires every include term and rejects any exclude term", () => {
    expect(
      filterTorrentResults(results, {
        minSeeds: 0,
        trackers: [],
        include: ["mad max", "2015"],
        exclude: ["720p"],
      }),
    ).toEqual([results[0]]);
  });

  it("keeps spaces and commas inside one atomic term", () => {
    expect(
      filterTorrentResults(results, {
        minSeeds: 0,
        trackers: [],
        include: ["1080p, hdr"],
        exclude: [],
      }).map((result) => result.Title),
    ).toEqual([results[2]?.Title, results[0]?.Title]);
  });

  it("combines seed and tracker filters with the word filters", () => {
    expect(
      filterTorrentResults(results, {
        minSeeds: 50,
        trackers: ["A"],
        trackersActive: true,
        include: ["2015"],
        exclude: [],
      }),
    ).toEqual([results[0]]);
  });

  it("returns no results when the active tracker selection is empty", () => {
    expect(
      filterTorrentResults(results, {
        minSeeds: 0,
        trackers: [],
        trackersActive: true,
        include: [],
        exclude: [],
      }),
    ).toEqual([]);
  });
});
