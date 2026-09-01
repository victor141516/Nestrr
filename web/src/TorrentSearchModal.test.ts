import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import TorrentSearchModal from "./TorrentSearchModal.vue";

vi.mock("./api", () => ({ api: vi.fn() }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("TorrentSearchModal", () => {
  it("searches, renders results and emits an accepted selection", async () => {
    const result = {
      ID: "torrent-1",
      Title: "Film 1080p",
      Tracker: "Indexer",
      Seeders: 20,
      Size: 2_147_483_648,
      MagnetURI: "magnet:?xt=film",
    };
    vi.mocked(api).mockImplementation(async (path) =>
      path.startsWith("/torrents/search") ? { results: [result] } : undefined,
    );
    const view = render(TorrentSearchModal, {
      props: {
        item: { id: 4, title: "Film" },
        kind: "movie",
        categories: { movie: "Movies", show: "Shows" },
        translate: (key) => key,
      },
      global: {
        stubs: {
          AppIcon: true,
          TorrentTagInput: true,
          TrackerMultiSelect: true,
        },
      },
    });

    await fireEvent.click(view.getByRole("button", { name: "search" }));
    expect(await view.findByText("Film 1080p")).toBeTruthy();

    await fireEvent.click(view.getByRole("button", { name: "sendTo Movies" }));

    const accepted = view.emitted().accepted as unknown[][];
    expect(accepted[0]?.[0]).toMatchObject({
      item: { id: 4, title: "Film" },
      kind: "movie",
      category: "Movies",
      result,
    });
    expect(view.emitted().notice).toEqual([["torrentAccepted"]]);
  });
});
