import { describe, expect, it } from "vitest";
import { mediaTypeFromQuery, mediaTypeQueryValue } from "./mediaType";

describe("media type URL state", () => {
  it("accepts canonical names and the legacy TV value", () => {
    expect(mediaTypeFromQuery("Movies")).toBe("movie");
    expect(mediaTypeFromQuery("Shows")).toBe("tv");
    expect(mediaTypeFromQuery("tv")).toBe("tv");
    expect(mediaTypeFromQuery(undefined)).toBe("movie");
  });

  it("serializes canonical category names", () => {
    expect(mediaTypeQueryValue("movie")).toBe("Movies");
    expect(mediaTypeQueryValue("tv")).toBe("Shows");
  });
});
