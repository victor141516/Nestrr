// @vitest-environment jsdom
import { fireEvent, render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import AdvancedMovieFilters from "./AdvancedMovieFilters.vue";
import { createMovieFilters } from "./movieFilters";

describe("AdvancedMovieFilters", () => {
  it("leaves closing to the filter trigger while retaining filter actions", async () => {
    const view = render(AdvancedMovieFilters, {
      props: {
        filters: createMovieFilters("en-US"),
        genres: [],
        countries: [],
        loading: false,
        mediaType: "movie",
        rottenTomatoesEnabled: false,
      },
      global: {
        stubs: {
          AppIcon: true,
          CountryMultiSelect: true,
          DualRangeSlider: true,
          KeywordTagInput: true,
        },
      },
    });

    expect(view.queryByRole("button", { name: "Close" })).toBeNull();
    expect(view.queryByRole("button", { name: "Cancel" })).toBeNull();

    await fireEvent.click(view.getByRole("button", { name: /Reset filters/ }));
    await fireEvent.click(view.getByRole("button", { name: /Show results/ }));
    expect(view.emitted().reset).toHaveLength(1);
    expect(view.emitted().apply).toHaveLength(1);
  });

  it("only exposes the Rotten Tomatoes range when the backend supports it", () => {
    const baseProps = {
      filters: createMovieFilters("en-US"),
      genres: [],
      countries: [],
      loading: false,
      mediaType: "movie" as const,
    };
    const disabled = render(AdvancedMovieFilters, {
      props: { ...baseProps, rottenTomatoesEnabled: false },
      global: { stubs: { AppIcon: true, DualRangeSlider: true } },
    });
    expect(disabled.queryByText("Rotten Tomatoes")).toBeNull();
    disabled.unmount();

    const enabled = render(AdvancedMovieFilters, {
      props: { ...baseProps, rottenTomatoesEnabled: true },
      global: { stubs: { AppIcon: true, DualRangeSlider: true } },
    });
    expect(enabled.getByText("Rotten Tomatoes")).toBeTruthy();
  });
});
