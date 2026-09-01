// @vitest-environment jsdom
import { cleanup, render, screen } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import MediaCard from "./MediaCard.vue";

afterEach(cleanup);

const global = {
  stubs: {
    RouterLink: { template: "<a><slot /></a>" },
    AppIcon: {
      props: ["name"],
      template: '<span :data-icon="name" />',
    },
  },
};

describe("MediaCard", () => {
  it("shows a cached Rotten Tomatoes score beside TMDB", () => {
    render(MediaCard, {
      props: {
        m: {
          id: 278,
          title: "The Shawshank Redemption",
          release_date: "1994-09-23",
          vote_average: 8.7,
          rotten_tomatoes_score: 89,
        },
        to: "/movie/278",
      },
      global,
    });

    expect(screen.getByLabelText("TMDB: 8.7")).toBeTruthy();
    expect(screen.getByLabelText("Rotten Tomatoes: 89%")).toBeTruthy();
  });

  it("omits Rotten Tomatoes when the score is not present", () => {
    render(MediaCard, {
      props: {
        m: { id: 238, title: "The Godfather", vote_average: 8.7 },
        to: "/movie/238",
      },
      global,
    });

    expect(screen.queryByLabelText(/Rotten Tomatoes/)).toBeNull();
  });
});
