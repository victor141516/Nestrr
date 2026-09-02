// @vitest-environment jsdom
import { fireEvent, render, within } from "@testing-library/vue";
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
          CompanyTagInput: true,
          DualRangeSlider: true,
          KeywordTagInput: true,
          PersonTagInput: true,
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

  it("offers cast and director filters for movies only", () => {
    const props = {
      filters: createMovieFilters("en-US"),
      genres: [],
      countries: [],
      loading: false,
      rottenTomatoesEnabled: false,
    };
    const movie = render(AdvancedMovieFilters, {
      props: { ...props, mediaType: "movie" },
      global: { stubs: { AppIcon: true, PersonTagInput: false } },
    });
    const movieView = within(movie.container as HTMLElement);
    expect(movieView.getByText("People")).toBeTruthy();
    expect(movieView.getByLabelText("Cast")).toBeTruthy();
    expect(movieView.getByLabelText("Directors")).toBeTruthy();
    expect(movieView.getByText("Production companies")).toBeTruthy();
    expect(movieView.getByLabelText("Production company")).toBeTruthy();
    movie.unmount();

    const show = render(AdvancedMovieFilters, {
      props: { ...props, mediaType: "tv" },
      global: { stubs: { AppIcon: true, PersonTagInput: true } },
    });
    expect(
      within(show.container as HTMLElement).queryByText("People"),
    ).toBeNull();
    expect(
      within(show.container as HTMLElement).queryByText("Production companies"),
    ).toBeNull();
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
