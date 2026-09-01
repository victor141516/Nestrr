import { describe, expect, it } from "vitest";
import { router } from "./router";

describe("application routes", () => {
  it("maps each product area to a concrete route component", () => {
    const paths = new Set(router.getRoutes().map((route) => route.path));
    for (const path of [
      "/",
      "/library",
      "/lists",
      "/lists/:id(\\d+)",
      "/settings",
      "/admin",
      "/movie/:id(\\d+)",
      "/tv/:id(\\d+)",
      "/collection/:id(\\d+)",
    ])
      expect(paths.has(path)).toBe(true);
  });

  it("resolves media details without falling through to the not-found route", () => {
    expect(router.resolve("/movie/42").matched[0]?.path).toBe(
      "/movie/:id(\\d+)",
    );
    expect(router.resolve("/tv/7").matched[0]?.path).toBe("/tv/:id(\\d+)");
    expect(router.resolve("/missing").matched[0]?.path).toBe(
      "/:pathMatch(.*)*",
    );
  });

  it("keeps old library links working through downloaded exploration", () => {
    const route = router.getRoutes().find((item) => item.path === "/library");
    expect(typeof route?.redirect).toBe("function");
    const redirect = (route?.redirect as Function)({ query: { type: "tv" } });
    expect(redirect).toEqual({
      path: "/",
      query: { type: "tv", downloaded: "1" },
    });
  });
});
