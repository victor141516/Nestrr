import { describe, expect, it } from "vitest";
import {
  isTorrentDownloaded,
  isTorrentDownloading,
  isTorrentForceable,
  isTorrentStalled,
  torrentPercent,
  torrentStateName,
  waitForTorrentForceConfirmation,
} from "./torrentState";

describe("qBittorrent download state", () => {
  it("keeps state names in English", () => {
    expect(torrentStateName("stalledDL")).toBe("Stalled");
    expect(torrentStateName("forcedDL")).toBe("Forced downloading");
  });

  it("only treats active download states as downloading", () => {
    expect(isTorrentDownloading("downloading")).toBe(true);
    expect(isTorrentDownloading("forcedDL")).toBe(true);
    expect(isTorrentDownloading("queuedDL")).toBe(false);
    expect(isTorrentDownloading("stalledDL")).toBe(false);
    expect(isTorrentStalled("stalledDL")).toBe(true);
    expect(isTorrentForceable("stalledDL")).toBe(true);
    expect(isTorrentForceable("queuedDL")).toBe(true);
    expect(isTorrentForceable("downloading")).toBe(false);
  });

  it("recognizes completed downloads that qBittorrent is uploading", () => {
    for (const state of [
      "uploading",
      "pausedUP",
      "stoppedUP",
      "queuedUP",
      "stalledUP",
      "checkingUP",
      "forcedUP",
    ]) {
      expect(isTorrentDownloaded(state)).toBe(true);
    }

    for (const state of [
      "downloading",
      "forcedDL",
      "queuedDL",
      "stalledDL",
      "moving",
    ]) {
      expect(isTorrentDownloaded(state)).toBe(false);
    }
  });

  it("clamps and rounds qBittorrent progress", () => {
    expect(torrentPercent(0.424)).toBe(42);
    expect(torrentPercent(2)).toBe(100);
    expect(torrentPercent(-1)).toBe(0);
  });

  it("keeps polling until qBittorrent confirms forced downloading", async () => {
    const states = ["queuedDL", "stalledDL", "forcedDL"];
    const refreshed: string[] = [];
    const waits: number[] = [];
    const result = await waitForTorrentForceConfirmation(
      async () => {
        const state = states.shift() || "forcedDL";
        refreshed.push(state);
        return { infoHash: "hash", name: "Movie", state, progress: 0.42 };
      },
      {
        attempts: 5,
        intervalMs: 250,
        wait: async (milliseconds) => {
          waits.push(milliseconds);
        },
      },
    );

    expect(refreshed).toEqual(["queuedDL", "stalledDL", "forcedDL"]);
    expect(waits).toEqual([250, 250]);
    expect(result?.state).toBe("forcedDL");
  });

  it("stops waiting after the configured limit", async () => {
    let refreshes = 0;
    const result = await waitForTorrentForceConfirmation(
      async () => {
        refreshes += 1;
        return {
          infoHash: "hash",
          name: "Movie",
          state: "queuedDL",
          progress: 0.1,
        };
      },
      { attempts: 3, intervalMs: 0, wait: async () => {} },
    );

    expect(refreshes).toBe(3);
    expect(result?.state).toBe("queuedDL");
  });
});
