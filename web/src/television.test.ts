import { describe, expect, it } from "vitest";
import {
  directionalTarget,
  remoteKey,
  televisionEnabled,
  televisionPlayerURL,
} from "./television";

describe("television navigation", () => {
  it("requires explicit opt-in outside the hash router", () => {
    expect(televisionEnabled("?tv=1")).toBe(true);
    expect(televisionEnabled("?tv=0")).toBe(false);
    expect(televisionEnabled("")).toBe(false);
  });
  it("preserves the media, resume time and trusted parent when opting the iframe in", () => {
    const url = new URL(
      televisionPlayerURL(
        "https://plarr.example/#/play/abc?start=12.5&parentOrigin=https%3A%2F%2Fnestrr.example",
      ),
    );
    expect(url.origin).toBe("https://plarr.example");
    expect(url.hash.split("?")[0]).toBe("#/play/abc");
    const params = new URLSearchParams(url.hash.split("?")[1]);
    expect(params.get("start")).toBe("12.5");
    expect(params.get("parentOrigin")).toBe("https://nestrr.example");
    expect(params.get("tv")).toBe("1");
  });
  it("recognizes the LG Back key and legacy arrow events", () => {
    expect(remoteKey({ key: "Unidentified", keyCode: 461 })).toBe("Back");
    expect(remoteKey({ key: "Unidentified", keyCode: 39 })).toBe("ArrowRight");
  });
  it("keeps movement within a row before crossing to another row", () => {
    const adjacent = document.createElement("button");
    const diagonal = document.createElement("button");
    expect(
      directionalTarget(
        new DOMRect(0, 0, 100, 100),
        [
          { element: diagonal, rect: new DOMRect(110, 120, 100, 100) },
          { element: adjacent, rect: new DOMRect(150, 0, 100, 100) },
        ],
        "ArrowRight",
      ),
    ).toBe(adjacent);
    expect(
      directionalTarget(
        new DOMRect(0, 0, 100, 100),
        [{ element: adjacent, rect: new DOMRect(150, 0, 100, 100) }],
        "ArrowLeft",
      ),
    ).toBeUndefined();
  });
});
