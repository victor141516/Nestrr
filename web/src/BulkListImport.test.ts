import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it, vi } from "vitest";
import BulkListImport from "./BulkListImport.vue";
import { streamAPI } from "./api";

vi.mock("./api", () => ({ streamAPI: vi.fn() }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("BulkListImport", () => {
  it("renders streamed states and emits each saved item immediately", async () => {
    vi.mocked(streamAPI).mockImplementation(
      async (_path, _options, onEvent) => {
        onEvent({ index: 0, input: "Fight Club", status: "database_search" });
        onEvent({
          index: 0,
          input: "Fight Club",
          status: "saved",
          source: "database",
          added: true,
          item: { mediaType: "movie", tmdbId: 550, title: "Fight Club" },
        });
        onEvent({ index: 1, input: "Unknown", status: "database_miss" });
        onEvent({ index: 1, input: "Unknown", status: "tmdb_search" });
        onEvent({
          index: 1,
          input: "Unknown",
          status: "not_found",
          source: "tmdb",
        });
        onEvent({ status: "complete" });
      },
    );
    const view = render(BulkListImport, {
      props: { listId: 7, translate: (key) => key },
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.update(
      view.getByRole("textbox", { name: "titles" }),
      "Fight Club\nUnknown",
    );
    await fireEvent.click(view.getByRole("button", { name: "bulkAdd" }));

    expect(streamAPI).toHaveBeenCalledWith(
      "/lists/7/bulk-import",
      expect.objectContaining({ method: "POST" }),
      expect.any(Function),
    );
    expect(await view.findByText("addedToList")).toBeTruthy();
    expect(view.getAllByText("Unknown")).toHaveLength(2);
    expect(view.getAllByText("notFound")).toHaveLength(2);
    const imported = view.emitted().imported as unknown[][];
    expect(imported[0]?.[0]).toMatchObject({
      added: true,
      item: { tmdbId: 550 },
    });
    expect(view.emitted().finished).toHaveLength(1);
  });

  it("aborts an active import when the panel is unmounted", async () => {
    let signal: AbortSignal | undefined;
    vi.mocked(streamAPI).mockImplementation(
      async (_path, options) =>
        new Promise<void>((resolve) => {
          signal = options.signal ?? undefined;
          signal?.addEventListener("abort", () => resolve());
        }),
    );
    const view = render(BulkListImport, {
      props: { listId: 7, translate: (key) => key },
      global: { stubs: { AppIcon: true } },
    });
    await fireEvent.update(
      view.getByRole("textbox", { name: "titles" }),
      "Fight Club",
    );
    void fireEvent.click(view.getByRole("button", { name: "bulkAdd" }));
    await vi.waitFor(() => expect(signal).toBeTruthy());
    view.unmount();
    expect(signal?.aborted).toBe(true);
  });
});
