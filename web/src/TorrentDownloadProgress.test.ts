import { render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import TorrentDownloadProgress from "./TorrentDownloadProgress.vue";

describe("TorrentDownloadProgress", () => {
  it("hides Force Start when qBittorrent confirms forced downloading", async () => {
    const props = {
      torrent: {
        infoHash: "hash",
        name: "Movie",
        state: "queuedDL",
        progress: 0.21,
      },
      completedLabel: "Download finished · Pending organization",
      forceLabel: "Force download",
      forcing: true,
    };
    const view = render(TorrentDownloadProgress, { props });

    expect(
      (
        view.getByRole("button", {
          name: "Force download",
        }) as HTMLButtonElement
      ).disabled,
    ).toBe(true);

    await view.rerender({
      ...props,
      torrent: { ...props.torrent, state: "forcedDL", progress: 0.24 },
      forcing: false,
    });

    expect(view.queryByRole("button", { name: "Force download" })).toBeNull();
    const progress = view.getByRole("progressbar", {
      name: "Forced downloading: 24%",
    });
    expect(progress.classList.contains("active")).toBe(true);
  });
});
