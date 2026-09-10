// @vitest-environment jsdom
import { cleanup, render, waitFor } from "@testing-library/vue";
import { defineComponent, h } from "vue";
import { createMemoryHistory, createRouter } from "vue-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useRecommendations } from "./useRecommendations";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

async function testRouter(path = "/recommendations") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: "/recommendations", component: { render: () => null } }],
  });
  await router.push(path);
  await router.isReady();
  return router;
}

describe("useRecommendations", () => {
  it("loads the URL media type and writes canonical selections back to it", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const tv = String(input).includes("kind=tv");
      return new Response(
        JSON.stringify({
          results: [
            {
              id: tv ? 2 : 1,
              media_type: tv ? "tv" : "movie",
              name: tv ? "Series" : undefined,
              title: tv ? undefined : "Movie",
            },
          ],
          total_results: 1,
          media_type: tv ? "tv" : "movie",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });
    vi.stubGlobal("fetch", fetchMock);
    const router = await testRouter("/recommendations?type=Shows");

    let recommendations!: ReturnType<typeof useRecommendations>;
    render(
      defineComponent({
        setup() {
          recommendations = useRecommendations();
          return () => h("span", recommendations.items.value[0]?.id ?? "empty");
        },
      }),
      { global: { plugins: [router] } },
    );

    await waitFor(() => expect(recommendations.items.value[0]?.id).toBe(2));
    await recommendations.selectMediaType("movie");
    await waitFor(() => expect(recommendations.items.value[0]?.id).toBe(1));
    expect(router.currentRoute.value.query.type).toBe("Movies");
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("aborts its active request when the owner unmounts", async () => {
    let signal: AbortSignal | undefined;
    vi.stubGlobal(
      "fetch",
      vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
        signal = init?.signal ?? undefined;
        return new Promise<Response>(() => {});
      }),
    );
    const router = await testRouter();
    const view = render(
      defineComponent({
        setup() {
          useRecommendations();
          return () => h("span");
        },
      }),
      { global: { plugins: [router] } },
    );
    await waitFor(() => expect(signal).toBeDefined());
    view.unmount();
    expect(signal?.aborted).toBe(true);
  });
});
