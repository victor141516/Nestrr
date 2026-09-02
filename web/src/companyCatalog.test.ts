import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import {
  mergeCompanies,
  resolveCompanies,
  searchCompanies,
} from "./companyCatalog";

vi.mock("./api", () => ({ api: vi.fn() }));

describe("company catalog", () => {
  beforeEach(() => vi.mocked(api).mockReset());

  it("queries local and remote phases independently", async () => {
    vi.mocked(api).mockResolvedValue({ results: [{ id: 1, name: "Studio" }] });
    await searchCompanies("studio", "local");
    await searchCompanies("studio", "remote");
    expect(api).toHaveBeenNthCalledWith(
      1,
      "/companies/search?q=studio&phase=local&limit=12",
      { signal: undefined },
    );
    expect(api).toHaveBeenNthCalledWith(
      2,
      "/companies/search?q=studio&phase=remote&limit=12",
      { signal: undefined },
    );
  });

  it("keeps local ranking while appending deduplicated TMDB matches", () => {
    expect(
      mergeCompanies(
        [
          { id: 1, name: "Local first" },
          { id: 2, name: "Shared local" },
        ],
        [
          { id: 2, name: "Shared remote" },
          { id: 3, name: "Remote only" },
        ],
      ),
    ).toEqual([
      { id: 1, name: "Local first" },
      { id: 2, name: "Shared local" },
      { id: 3, name: "Remote only" },
    ]);
  });

  it("deduplicates identifiers when hydrating URL state", async () => {
    vi.mocked(api).mockResolvedValue({ results: [] });
    await resolveCompanies([7, 7, -1, 9]);
    expect(api).toHaveBeenCalledWith("/companies/search?ids=7,9", {
      signal: undefined,
    });
  });
});
