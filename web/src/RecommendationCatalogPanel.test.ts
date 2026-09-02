// @vitest-environment jsdom
import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import RecommendationCatalogPanel from "./RecommendationCatalogPanel.vue";

afterEach(cleanup);

const status = {
  id: 1,
  state: "running" as const,
  phase: "hydration",
  usersScanned: 3,
  signals: { genres: 4, keywords: 5, actors: 6, directors: 2, companies: 3 },
  discovery: { total: 10, done: 8, failed: 1 },
  hydration: { total: 20, done: 10, failed: 2 },
  catalogCount: 120,
  pending: 10,
};

describe("RecommendationCatalogPanel", () => {
  it("renders stable progress and disables duplicate runs while active", () => {
    const view = render(RecommendationCatalogPanel, {
      props: { status, translate: (key: string) => key },
    });
    expect(view.getByText("8/10")).toBeTruthy();
    expect(view.getByText("10/20")).toBeTruthy();
    expect(
      view.getByRole("button", { name: "running" }).hasAttribute("disabled"),
    ).toBe(true);
  });

  it("emits a run action when idle", async () => {
    const view = render(RecommendationCatalogPanel, {
      props: {
        status: { ...status, state: "completed" as const },
        translate: (key: string) => key,
      },
    });
    await fireEvent.click(view.getByRole("button", { name: "runNow" }));
    expect(view.emitted().run).toHaveLength(1);
  });
});
