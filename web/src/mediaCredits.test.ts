import { describe, expect, it } from "vitest";
import { movieCredits } from "./mediaCredits";

describe("movieCredits", () => {
  it("extracts directors and the leading cast in credited order", () => {
    expect(
      movieCredits(
        {
          crew: [
            { id: 1, name: "Writer", job: "Writer" },
            {
              id: 2,
              name: "Director",
              job: "Director",
              profile_path: "/director.jpg",
            },
            { id: 2, name: "Director", job: "Director" },
          ],
          cast: [
            { id: 11, name: "Second", character: "B", order: 1 },
            {
              id: 10,
              name: "First",
              character: "A",
              order: 0,
              profile_path: "/first.jpg",
            },
            { id: 11, name: "Second duplicate", order: 2 },
            { id: 12, name: "Third", character: "C", order: 3 },
          ],
        },
        2,
      ),
    ).toEqual({
      directors: [
        {
          id: 2,
          name: "Director",
          profilePath: "/director.jpg",
        },
      ],
      cast: [
        {
          id: 10,
          name: "First",
          profilePath: "/first.jpg",
          character: "A",
          order: 0,
        },
        {
          id: 11,
          name: "Second",
          profilePath: null,
          character: "B",
          order: 1,
        },
      ],
    });
  });

  it("ignores malformed people and accepts an empty response", () => {
    expect(movieCredits(null)).toEqual({ directors: [], cast: [] });
    expect(
      movieCredits({
        crew: [{ id: 0, name: "Nobody", job: "Director" }],
        cast: [{ id: 3, name: "" }],
      }),
    ).toEqual({ directors: [], cast: [] });
  });
});
