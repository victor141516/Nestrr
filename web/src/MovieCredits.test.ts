// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import MovieCredits from "./MovieCredits.vue";

afterEach(cleanup);

describe("MovieCredits", () => {
  it("renders non-interactive, identified directors and leading cast", () => {
    const artwork = vi.fn(
      (path?: string | null, size?: string) => `${size}:${path}`,
    );
    const { container } = render(MovieCredits, {
      props: {
        directors: [{ id: 1, name: "Jane Director", profilePath: null }],
        cast: [
          {
            id: 2,
            name: "Ada Actor",
            profilePath: "/ada.jpg",
            character: "The Lead",
            order: 0,
          },
          {
            id: 3,
            name: "No Portrait",
            profilePath: null,
            character: "",
            order: 1,
          },
        ],
        directorLabel: "Director",
        castLabel: "Cast",
        artwork,
        directorTo: (person) => `/directors/${person.id}`,
        castTo: (person) => `/cast/${person.id}`,
      },
      global: {
        stubs: {
          RouterLink: {
            props: ["to"],
            template: '<a :href="to"><slot /></a>',
          },
        },
      },
    });

    expect(screen.getByText("Jane Director")).toBeTruthy();
    expect(screen.getByText("Ada Actor")).toBeTruthy();
    expect(screen.getByText("The Lead")).toBeTruthy();
    expect(container.querySelector('[data-person-id="1"]')).toBeTruthy();
    expect(container.querySelector('[data-person-id="2"]')).toBeTruthy();
    expect(container.querySelectorAll("a")).toHaveLength(3);
    expect(artwork).toHaveBeenCalledWith("/ada.jpg", "w185");
  });

  it("renders nothing when no credits are available", () => {
    const { container } = render(MovieCredits, {
      props: {
        directors: [],
        cast: [],
        directorLabel: "Director",
        castLabel: "Cast",
        artwork: () => "",
        directorTo: () => "/",
        castTo: () => "/",
      },
    });

    expect(container.firstElementChild).toBeNull();
  });
});
