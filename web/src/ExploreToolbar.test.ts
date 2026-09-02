// @vitest-environment jsdom
import { fireEvent, render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import ExploreToolbar from "./ExploreToolbar.vue";

const props = {
  query: "",
  sort: "popularity" as const,
  searchType: "movie" as const,
  downloadedOnly: false,
  busy: false,
  filterPanelOpen: false,
  movieFilterCount: 0,
  translate: (key: string) => key,
};

describe("ExploreToolbar", () => {
  it("uses a stable segmented track and emits media changes", async () => {
    const view = render(ExploreToolbar, {
      props,
      global: { stubs: { AppIcon: true } },
    });

    expect(view.container.querySelector(".segmented-selection")).toBeTruthy();
    await fireEvent.click(view.getByRole("button", { name: "shows" }));
    expect(view.emitted().selectType).toEqual([["tv"]]);
  });

  it("emits the downloaded dataset selection", async () => {
    const view = render(ExploreToolbar, {
      props,
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.click(
      view.container.querySelector('input[type="checkbox"]')!,
    );
    expect(view.emitted().setDownloadedOnly).toEqual([[true]]);
  });

  it("uses the custom sort menu and emits its selection", async () => {
    const view = render(ExploreToolbar, {
      props,
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.click(view.container.querySelector(".app-select-trigger")!);
    await fireEvent.click(
      view.container.querySelector(
        '[role="menuitemradio"][data-value="release"]',
      )!,
    );

    expect(view.emitted()["update:sort"]).toEqual([["release"]]);
    expect(view.emitted().changeSort).toEqual([["release"]]);
  });

  it("keeps the search action label stable during background loading", () => {
    const view = render(ExploreToolbar, {
      props: { ...props, query: "Dune", busy: true },
      global: { stubs: { AppIcon: true } },
    });

    const search = view.container.querySelector(
      ".primary.explore-control",
    ) as HTMLButtonElement;
    expect(search.textContent?.trim()).toBe("search");
    expect((search as HTMLButtonElement).disabled).toBe(false);
  });
});
