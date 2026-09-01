// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import MediaKeywordsMenu from "./MediaKeywordsMenu.vue";

afterEach(cleanup);

const items = [
  "superhero",
  "space opera",
  "alien",
  "future",
  "spaceship",
  "dystopia",
  "astronaut",
  "time travel",
  "artificial intelligence",
].map((label, index) => ({
  id: index + 1,
  label,
  to: `/?includeKeywords=${index + 1}`,
}));

describe("MediaKeywordsMenu", () => {
  it("sorts keywords alphabetically", async () => {
    const view = render(MediaKeywordsMenu, {
      props: {
        items,
        label: "Keywords",
        noResultsLabel: "No matches found",
        searchLabel: "Search keywords",
      },
      global: { stubs: { RouterLink: { template: "<a><slot /></a>" } } },
    });

    await fireEvent.click(screen.getByText("Keywords"));
    expect(
      [...view.container.querySelectorAll(".detail-keyword-list a")].map(
        (element) => element.textContent?.trim(),
      ),
    ).toEqual([
      "alien",
      "artificial intelligence",
      "astronaut",
      "dystopia",
      "future",
      "space opera",
      "spaceship",
      "superhero",
      "time travel",
    ]);
  });

  it("filters a long keyword list locally and closes with Escape", async () => {
    const view = render(MediaKeywordsMenu, {
      props: {
        items,
        label: "Keywords",
        noResultsLabel: "No matches found",
        searchLabel: "Search keywords",
      },
      global: { stubs: { RouterLink: { template: "<a><slot /></a>" } } },
    });

    await fireEvent.click(screen.getByText("Keywords"));
    const search = screen.getByRole("searchbox", { name: "Search keywords" });
    await fireEvent.update(search, "space");

    expect(screen.getByText("space opera")).toBeTruthy();
    expect(screen.getByText("spaceship")).toBeTruthy();
    expect(screen.queryByText("superhero")).toBeNull();

    await fireEvent.keyDown(document, { key: "Escape" });
    expect(view.container.querySelector("details")?.open).toBe(false);
  });
});
