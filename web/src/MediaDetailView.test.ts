// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/vue";
import { reactive, ref } from "vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import MediaDetailView from "./MediaDetailView.vue";
import { nestrrApplicationKey } from "./useNestrrApplication";

afterEach(cleanup);

describe("MediaDetailView", () => {
  it("keeps playback and list actions wired through the application boundary", async () => {
    const playDetail = vi.fn();
    const toggleSystemList = vi.fn();
    const application = {
      detail: ref({
        id: 42,
        title: "The Test Film",
        overview: "Overview",
        release_date: "2026-01-01",
        rotten_tomatoes_score: 94,
        genres: [],
      }),
      artwork: () => "",
      availability: reactive({}),
      currentAvailability: ref({
        available: true,
        playerUrl: "/player",
      }),
      currentKind: ref("movie"),
      detailProgress: ref(null),
      fmtSeconds: () => "",
      forceTorrent: vi.fn(),
      forcingTorrents: reactive({}),
      inSystemList: () => false,
      integrations: reactive({
        plarr: { healthy: true },
        jackett: { healthy: true },
        qbittorrent: { healthy: true },
      }),
      listEditor: ref(null),
      mediaLists: ref([]),
      membershipsFor: () => [],
      openTorrent: vi.fn(),
      playDetail,
      refreshDetail: vi.fn(),
      rottenTomatoesEnabled: ref(true),
      renditionURL: () => "",
      t: (key: string) => key,
      title: (item: { title: string }) => item.title,
      toggleCustomList: vi.fn(),
      toggleSystemList,
      year: () => "2026",
      startedFor: () => false,
      systemMembershipsFor: () => [],
    } as any;

    render(MediaDetailView, {
      global: {
        provide: { [nestrrApplicationKey as symbol]: application },
        stubs: { RouterLink: { template: "<a><slot /></a>" } },
      },
    });

    expect(screen.getByRole("heading", { name: "The Test Film" })).toBeTruthy();
    expect(screen.getByText("94%")).toBeTruthy();
    await fireEvent.click(screen.getByRole("button", { name: "play" }));
    expect(playDetail).toHaveBeenCalledOnce();
    await fireEvent.click(screen.getByRole("button", { name: "favorites" }));
    expect(toggleSystemList).toHaveBeenCalledWith(
      "favorites",
      application.detail.value,
      "movie",
    );
  });
});
