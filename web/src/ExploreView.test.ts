// @vitest-environment jsdom
import { render, screen } from "@testing-library/vue";
import { ref } from "vue";
import { describe, expect, it, vi } from "vitest";
import ExploreView from "./ExploreView.vue";
import { nestrrApplicationKey } from "./useNestrrApplication";

describe("ExploreView", () => {
  it("renders the discovery controls when the catalog has not loaded yet", () => {
    const application = {
      appliedFilterChips: ref([]),
      appliedMovieFilters: ref(null),
      applyMovieFilters: vi.fn(),
      artwork: vi.fn(),
      availability: {},
      busy: ref(false),
      changeExploreSort: vi.fn(),
      continueItems: ref([]),
      doSearch: vi.fn(),
      downloadedOnly: ref(false),
      error: ref(""),
      exploreSort: ref("popularity"),
      filterOptionsLoading: ref(false),
      filterPanelOpen: ref(false),
      hasMoreSearchResults: ref(false),
      items: ref([]),
      loadMoreSearchResults: vi.fn(),
      loadingMore: ref(false),
      movieCountries: ref([]),
      movieFilterCount: ref(0),
      movieFilterDraft: ref({}),
      movieGenres: ref([]),
      openMovieFilters: vi.fn(),
      percent: vi.fn(() => 0),
      query: ref(""),
      removeMovieFilter: vi.fn(),
      resetMovieFilters: vi.fn(),
      resultsTransitioning: ref(false),
      searchMode: ref("idle"),
      searchStarted: ref(false),
      searchTotalResults: ref(0),
      searchType: ref("movie"),
      selectSearchType: vi.fn(),
      setDownloadedOnly: vi.fn(),
      startedFor: vi.fn(() => false),
      systemMembershipsFor: vi.fn(() => []),
      t: (key: string) => key,
      title: vi.fn(),
      visibleItems: ref([]),
    };

    render(ExploreView, {
      global: {
        provide: {
          [nestrrApplicationKey as symbol]: application,
        },
        stubs: {
          AdvancedMovieFilters: true,
          AppIcon: true,
          MediaCard: true,
          RouterLink: true,
        },
      },
    });

    expect(screen.getByRole("heading", { name: "discover" })).toBeTruthy();
    expect(screen.getByRole("button", { name: /search/ })).toBeTruthy();
  });
});
