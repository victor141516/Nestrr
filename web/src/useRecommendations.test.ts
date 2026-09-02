// @vitest-environment jsdom
import { cleanup, render, waitFor } from "@testing-library/vue";
import { defineComponent, h } from "vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useRecommendations } from "./useRecommendations";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("useRecommendations", () => {
  it("loads local recommendations and replaces results when the media type changes", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      const tv = url.includes("kind=tv");
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

    let recommendations!: ReturnType<typeof useRecommendations>;
    render(
      defineComponent({
        setup() {
          recommendations = useRecommendations();
          return () => h("span", recommendations.items.value[0]?.id ?? "empty");
        },
      }),
    );

    await waitFor(() => expect(recommendations.items.value[0]?.id).toBe(1));
    recommendations.mediaType.value = "tv";
    await waitFor(() => expect(recommendations.items.value[0]?.id).toBe(2));
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
    const view = render(
      defineComponent({
        setup() {
          useRecommendations();
          return () => h("span");
        },
      }),
    );
    await waitFor(() => expect(signal).toBeDefined());
    view.unmount();
    expect(signal?.aborted).toBe(true);
  });
});
