// @vitest-environment jsdom
import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import ListToolbar from "./ListToolbar.vue";

const props = {
  mediaType: "movie" as const,
  sort: "addedAt" as const,
  direction: "desc" as const,
  translate: (key: string) => key,
};

afterEach(cleanup);

describe("ListToolbar", () => {
  it("reuses the media switch and custom sort menu", async () => {
    const view = render(ListToolbar, {
      props,
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.click(view.getByRole("button", { name: "shows" }));
    expect(view.emitted().selectMediaType).toEqual([["tv"]]);

    await fireEvent.click(view.container.querySelector(".app-select-trigger")!);
    await fireEvent.click(
      view.container.querySelector(
        '[role="menuitemradio"][data-value="rating"]',
      )!,
    );
    expect(view.emitted().selectSort).toEqual([["rating"]]);
  });

  it("toggles direction through its typed model", async () => {
    const view = render(ListToolbar, {
      props,
      global: { stubs: { AppIcon: true } },
    });
    await fireEvent.click(view.getByRole("button", { name: "sortDirection" }));
    expect(view.emitted().toggleDirection).toHaveLength(1);
  });
});
