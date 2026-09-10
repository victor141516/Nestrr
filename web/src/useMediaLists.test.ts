// @vitest-environment jsdom
import { cleanup, render, waitFor } from "@testing-library/vue";
import { computed, defineComponent, h, ref } from "vue";
import {
  createMemoryHistory,
  createRouter,
  useRoute,
  useRouter,
} from "vue-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useMediaLists } from "./useMediaLists";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function response(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("useMediaLists list view", () => {
  it("keeps the current list mounted while another media type loads", async () => {
    let resolveShows!: (value: Response) => void;
    const showsResponse = new Promise<Response>((resolve) => {
      resolveShows = resolve;
    });
    vi.stubGlobal(
      "fetch",
      vi.fn((input: RequestInfo | URL) => {
        const url = String(input);
        if (url.includes("mediaType=tv")) return showsResponse;
        return Promise.resolve(
          response({
            list: {
              id: 1,
              name: "Favorites",
              color: "coral",
              icon: "heart",
              itemCount: 2,
              createdAt: "",
              updatedAt: "",
            },
            items: [{ mediaType: "movie", tmdbId: 1, title: "Movie" }],
          }),
        );
      }),
    );

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: "/lists/:id", component: { render: () => null } }],
    });
    await router.push("/lists/1?type=Movies&sort=addedAt&direction=desc");
    await router.isReady();

    let lists!: ReturnType<typeof useMediaLists>;
    render(
      defineComponent({
        setup() {
          const route = useRoute();
          lists = useMediaLists({
            router: useRouter(),
            currentRoute: route,
            pathOnly: computed(() => route.path),
            progress: ref([]),
            detail: ref(null),
            loadSeasonEpisodes: async () => {},
            t: (key) => key,
            flash: () => {},
          });
          return () => h("span", lists.activeList.value?.name ?? "empty");
        },
      }),
      { global: { plugins: [router] } },
    );

    await lists.loadListsPage();
    expect(lists.activeList.value?.id).toBe(1);
    expect(lists.activeListItems.value[0]?.title).toBe("Movie");

    lists.setListMediaType("tv");
    await waitFor(() =>
      expect(router.currentRoute.value.query.type).toBe("Shows"),
    );

    expect(lists.activeList.value?.id).toBe(1);
    expect(lists.activeListItems.value[0]?.title).toBe("Movie");

    resolveShows(
      response({
        list: {
          id: 1,
          name: "Favorites",
          color: "coral",
          icon: "heart",
          itemCount: 2,
          createdAt: "",
          updatedAt: "",
        },
        items: [{ mediaType: "tv", tmdbId: 2, title: "Show" }],
      }),
    );
    await waitFor(() =>
      expect(lists.activeListItems.value[0]?.title).toBe("Show"),
    );
  });
});
