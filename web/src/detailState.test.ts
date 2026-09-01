import { reactive } from "vue";
import { describe, expect, it } from "vitest";
import {
  createDetailRequest,
  isCurrentDetailRequest,
  isLatestDetailRequest,
} from "./detailState";

describe("detail request state", () => {
  it("matches a reactive proxy by stable media identity", () => {
    const raw = { id: 1275779 };
    const proxied = reactive(raw);
    const request = createDetailRequest(4, "movie", raw.id);

    expect(proxied).not.toBe(raw);
    expect(isCurrentDetailRequest(request, 4, "movie", proxied)).toBe(true);
  });

  it("rejects a response from an older request", () => {
    const request = createDetailRequest(4, "movie", 1275779);

    expect(isLatestDetailRequest(request, 5)).toBe(false);
    expect(isCurrentDetailRequest(request, 5, "movie", { id: 1275779 })).toBe(
      false,
    );
  });

  it("rejects a response for a different route or media item", () => {
    const request = createDetailRequest(4, "movie", 1275779);

    expect(isCurrentDetailRequest(request, 4, "tv", { id: 1275779 })).toBe(
      false,
    );
    expect(isCurrentDetailRequest(request, 4, "movie", { id: 157336 })).toBe(
      false,
    );
  });
});
