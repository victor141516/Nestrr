import { describe, expect, it } from "vitest";
import { playbackPlayerURL } from "./playbackPlayerURL";

describe("playbackPlayerURL", () => {
  it("adds Nestrr context without losing the watch-party room or grant", () => {
    const context = new URLSearchParams({
      title: "The Matrix",
      parentOrigin: "http://localhost:8080",
    });
    const result = new URL(
      playbackPlayerURL(
        "http://localhost:8090/#/play/movie?room=room-1&playbackToken=grant-1",
        context,
      ),
    );
    const params = new URLSearchParams(result.hash.split("?")[1]);
    expect(params.get("room")).toBe("room-1");
    expect(params.get("playbackToken")).toBe("grant-1");
    expect(params.get("title")).toBe("The Matrix");
    expect(params.get("parentOrigin")).toBe("http://localhost:8080");
  });
});
