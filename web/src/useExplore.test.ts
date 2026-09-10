// @vitest-environment jsdom
import { defineComponent, reactive, ref } from "vue";
import { render } from "@testing-library/vue";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import { useExplore } from "./useExplore";

vi.mock("./api", () => ({
  api: vi.fn(),
  session: { user: { language: "en-US" } },
}));

describe("useExplore downloaded dataset", () => {
  beforeEach(() => {
    vi.mocked(api).mockReset();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("loads the library directly and does not page through TMDB discover", async () => {
    vi.mocked(api).mockResolvedValue({
      items: [
        {
          tmdbId: 9,
          mediaType: "movie",
          metadata: { id: 9, title: "Local film", popularity: 12 },
        },
      ],
    });
    const route = reactive({
      path: "/",
      fullPath: "/?downloaded=1",
      query: { downloaded: "1" },
    });
    const router = {
      resolve: vi.fn(() => ({ fullPath: route.fullPath })),
      push: vi.fn(),
    };
    const availability = reactive<Record<string, any>>({});
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: router as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy: ref(false),
          error: ref(""),
          availability,
          checkCards: vi.fn(),
          rottenTomatoesEnabled: ref(false),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    await explore.loadExploreFromURL();

    expect(api).toHaveBeenCalledTimes(1);
    expect(api).toHaveBeenCalledWith("/library");
    expect(explore.visibleItems.value).toMatchObject([
      { id: 9, title: "Local film", media_type: "movie" },
    ]);
    expect(availability["movie:9"]).toMatchObject({
      available: true,
      state: "complete",
    });
    expect(explore.hasMoreSearchResults.value).toBe(false);
  });

  it("hydrates sorting from the URL and writes later changes back to it", async () => {
    vi.mocked(api).mockResolvedValue({
      page: 1,
      total_pages: 1,
      total_results: 1,
      results: [{ id: 5, title: "Recent film" }],
    });
    const route = reactive({
      path: "/",
      fullPath: "/?type=Movies&sort=release",
      query: { type: "Movies", sort: "release" },
    });
    const router = {
      resolve: vi.fn((target: { query: Record<string, string> }) => ({
        fullPath: `/?${new URLSearchParams(target.query).toString()}`,
      })),
      push: vi.fn(),
    };
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: router as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy: ref(false),
          error: ref(""),
          availability: reactive({}),
          checkCards: vi.fn(),
          rottenTomatoesEnabled: ref(false),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    await explore.loadExploreFromURL();

    expect(explore.exploreSort.value).toBe("release");
    expect(api).toHaveBeenCalledWith(
      expect.stringContaining("sort_by=primary_release_date.desc"),
    );

    await explore.changeExploreSort("rating");

    expect(router.push).toHaveBeenCalledWith({
      path: "/",
      query: { type: "Movies", sort: "rating" },
    });
  });

  it("keeps the current grid mounted and serves cached media while revalidating", async () => {
    vi.useFakeTimers();
    const deferred = <T>() => {
      let resolve!: (value: T) => void;
      const promise = new Promise<T>((done) => {
        resolve = done;
      });
      return { promise, resolve };
    };
    const tvResponse = deferred<any>();
    const movieRefresh = deferred<any>();
    let movieRequests = 0;
    vi.mocked(api).mockImplementation((path) => {
      const requestPath = String(path);
      if (requestPath.includes("/catalog/discover/tv"))
        return tvResponse.promise;
      if (requestPath.includes("/catalog/discover/movie")) {
        movieRequests += 1;
        if (movieRequests === 1)
          return Promise.resolve({
            page: 1,
            total_pages: 1,
            total_results: 1,
            results: [{ id: 1, title: "Movie cache" }],
          });
        return movieRefresh.promise;
      }
      throw new Error(`Unexpected request: ${requestPath}`);
    });
    const route = reactive({ path: "/", fullPath: "/", query: {} as any });
    const busy = ref(false);
    const router = {
      resolve: vi.fn(() => ({ fullPath: route.fullPath })),
      push: vi.fn(),
    };
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: router as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy,
          error: ref(""),
          availability: reactive({}),
          checkCards: vi.fn(),
          rottenTomatoesEnabled: ref(false),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    await explore.loadExploreFromURL();
    expect(explore.visibleItems.value[0]?.title).toBe("Movie cache");

    route.query = { type: "Shows" };
    route.fullPath = "/?type=Shows";
    const switchToTV = explore.loadExploreFromURL();
    await Promise.resolve();
    expect(explore.resultsTransitioning.value).toBe(true);
    expect(explore.visibleItems.value[0]?.title).toBe("Movie cache");
    expect(api).toHaveBeenCalledWith(
      expect.stringContaining("/catalog/discover/tv"),
    );

    tvResponse.resolve({
      page: 1,
      total_pages: 1,
      total_results: 1,
      results: [{ id: 2, name: "TV cache" }],
    });
    await vi.advanceTimersByTimeAsync(100);
    await switchToTV;
    expect(explore.visibleItems.value[0]?.name).toBe("TV cache");

    route.query = {};
    route.fullPath = "/";
    const switchToCachedMovies = explore.loadExploreFromURL();
    expect(explore.visibleItems.value[0]?.title).toBe("Movie cache");
    expect(explore.resultsTransitioning.value).toBe(false);
    expect(busy.value).toBe(true);

    movieRefresh.resolve({
      page: 1,
      total_pages: 1,
      total_results: 1,
      results: [{ id: 3, title: "Fresh movie" }],
    });
    await switchToCachedMovies;
    expect(explore.visibleItems.value[0]?.title).toBe("Fresh movie");
    expect(busy.value).toBe(false);
  });

  it("applies Rotten Tomatoes ranges locally without forwarding them to TMDB", async () => {
    vi.mocked(api).mockImplementation((path) => {
      const requestPath = String(path);
      if (requestPath.includes("/genre/movie/list"))
        return Promise.resolve({ genres: [] });
      if (requestPath.includes("/configuration/countries"))
        return Promise.resolve([]);
      if (requestPath.includes("/catalog/discover/movie"))
        return Promise.resolve({
          page: 1,
          total_pages: 1,
          total_results: 2,
          results: [
            { id: 1, title: "Fresh", rotten_tomatoes_score: 92 },
            { id: 2, title: "Low", rotten_tomatoes_score: 61 },
          ],
        });
      throw new Error(`Unexpected request: ${requestPath}`);
    });
    const route = reactive({
      path: "/",
      fullPath: "/?rottenTomatoesMin=80",
      query: { rottenTomatoesMin: "80" },
    });
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: {
            resolve: vi.fn(() => ({ fullPath: route.fullPath })),
            push: vi.fn(),
          } as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy: ref(false),
          error: ref(""),
          availability: reactive({}),
          checkCards: vi.fn(),
          rottenTomatoesEnabled: ref(true),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    await explore.loadExploreFromURL();

    expect(explore.visibleItems.value.map((item) => item.id)).toEqual([1]);
    const discoverCall = vi
      .mocked(api)
      .mock.calls.find(([path]) => String(path).includes("/discover/movie"));
    expect(String(discoverCall?.[0])).not.toContain("rottenTomatoes");
  });

  it("intersects local people and company relations before the remote director result", async () => {
    let resolveRemote!: (value: any) => void;
    const remote = new Promise<any>((resolve) => {
      resolveRemote = resolve;
    });
    vi.mocked(api).mockImplementation((path) => {
      const requestPath = String(path);
      if (requestPath.includes("/genre/movie/list"))
        return Promise.resolve({ genres: [] });
      if (requestPath.includes("/configuration/countries"))
        return Promise.resolve([]);
      if (requestPath.includes("/people/local-media"))
        return Promise.resolve({
          results: [
            {
              id: 10,
              title: "Cached collaboration",
              production_companies: [{ id: 40 }],
            },
            { id: 12, title: "Different company" },
          ],
        });
      if (requestPath.includes("/companies/local-media"))
        return Promise.resolve({
          results: [
            {
              id: 10,
              title: "Cached collaboration",
              production_companies: [{ id: 40 }],
            },
          ],
        });
      if (requestPath.includes("/people/discover/movie")) return remote;
      throw new Error(`Unexpected request: ${requestPath}`);
    });
    const route = reactive({
      path: "/",
      fullPath: "/?type=Movies&cast=20&directors=30&includeCompanies=40",
      query: {
        type: "Movies",
        cast: "20",
        castNames: '["Actor"]',
        directors: "30",
        directorNames: '["Director"]',
        includeCompanies: "40",
        companyNames: '["Studio"]',
      },
    });
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: {
            resolve: vi.fn(() => ({ fullPath: route.fullPath })),
            push: vi.fn(),
          } as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy: ref(false),
          error: ref(""),
          availability: reactive({}),
          checkCards: vi.fn(),
          rottenTomatoesEnabled: ref(false),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    const loading = explore.loadExploreFromURL();
    await vi.waitFor(() =>
      expect(explore.visibleItems.value[0]?.title).toBe("Cached collaboration"),
    );
    const localCall = vi
      .mocked(api)
      .mock.calls.find(([path]) => String(path).includes("/local-media"));
    expect(String(localCall?.[0])).toContain("cast=20");
    expect(String(localCall?.[0])).toContain("directors=30");
    const companyCall = vi
      .mocked(api)
      .mock.calls.find(([path]) => String(path).includes("/companies/local"));
    expect(String(companyCall?.[0])).toContain("companies=40");
    const remoteCall = vi
      .mocked(api)
      .mock.calls.find(([path]) => String(path).includes("/discover/movie"));
    expect(String(remoteCall?.[0])).toContain("with_cast=20");
    expect(String(remoteCall?.[0])).toContain("directors=30");
    expect(String(remoteCall?.[0])).toContain("with_companies=40");

    resolveRemote({
      page: 1,
      total_pages: 1,
      total_results: 1,
      results: [{ id: 11, title: "TMDB collaboration" }],
    });
    await loading;
    expect(explore.visibleItems.value[0]?.title).toBe("TMDB collaboration");
  });

  it("shows throttled SQLite results while typing and merges the settled TMDB response", async () => {
    vi.useFakeTimers();
    let resolveRemote!: (value: any) => void;
    const remoteResponse = new Promise<any>((resolve) => {
      resolveRemote = resolve;
    });
    const exploreBusy = ref(false);
    vi.mocked(api).mockImplementation((path) => {
      const requestPath = String(path);
      if (requestPath.includes("/catalog/local/search/movie"))
        return Promise.resolve({
          page: 1,
          total_pages: 1,
          total_results: 2,
          results: [
            { id: 1, title: "Malditos bastardos" },
            { id: 2, title: "Malditos local" },
          ],
        });
      if (requestPath.includes("/genre/movie/list"))
        return Promise.resolve({ genres: [] });
      if (requestPath.includes("/configuration/countries"))
        return Promise.resolve([]);
      if (requestPath.includes("/catalog/search/movie")) return remoteResponse;
      throw new Error(`Unexpected request: ${requestPath}`);
    });
    const route = reactive({ path: "/", fullPath: "/", query: {} as any });
    const router = {
      resolve: vi.fn((target: { query: Record<string, string> }) => ({
        fullPath: `/?${new URLSearchParams(target.query).toString()}`,
      })),
      push: vi.fn(),
    };
    let explore!: ReturnType<typeof useExplore>;
    const Harness = defineComponent({
      setup() {
        explore = useExplore({
          router: router as any,
          currentRoute: route as any,
          pathOnly: ref("/") as any,
          busy: exploreBusy,
          error: ref(""),
          availability: reactive({}),
          checkCards: vi.fn().mockResolvedValue(undefined),
          rottenTomatoesEnabled: ref(false),
          t: (key) => key,
          title: (media) => media.title || media.name || "",
        });
        return () => null;
      },
    });
    render(Harness);

    explore.query.value = "ma";
    await vi.advanceTimersByTimeAsync(40);
    explore.query.value = "malditos";
    await vi.advanceTimersByTimeAsync(40);
    await Promise.resolve();
    expect(api).toHaveBeenCalledTimes(1);
    expect(api).toHaveBeenCalledWith(
      expect.stringContaining("/catalog/local/search/movie?query=malditos"),
    );
    expect(explore.visibleItems.value.map((item) => item.id)).toEqual([1, 2]);

    await vi.advanceTimersByTimeAsync(2_000);
    expect(router.push).toHaveBeenCalledWith({
      path: "/",
      query: { type: "Movies", q: "malditos" },
    });

    route.query = { type: "Movies", q: "malditos" };
    route.fullPath = "/?type=Movies&q=malditos";
    const loading = explore.loadExploreFromURL();
    await Promise.resolve();
    expect(explore.visibleItems.value.map((item) => item.id)).toEqual([1, 2]);
    expect(explore.resultsTransitioning.value).toBe(false);
    expect(exploreBusy.value).toBe(false);

    resolveRemote({
      page: 1,
      total_pages: 3,
      total_results: 41,
      results: [
        { id: 1, title: "Malditos bastardos" },
        { id: 3, title: "Malditos TMDB" },
      ],
    });
    await loading;
    expect(explore.visibleItems.value.map((item) => item.id)).toEqual([
      1, 3, 2,
    ]);
    expect(explore.searchTotalResults.value).toBe(41);
  });
});
