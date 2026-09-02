import { beforeEach, describe, expect, it, vi } from "vitest";
import { api } from "./api";
import { mergePeople, resolvePeople, searchPeople } from "./personCatalog";

vi.mock("./api", () => ({ api: vi.fn() }));

describe("person catalog", () => {
  beforeEach(() => vi.mocked(api).mockReset());

  it("queries local and remote phases independently", async () => {
    vi.mocked(api).mockResolvedValue({ results: [{ id: 1, name: "Actor" }] });
    await searchPeople("actor", "local");
    await searchPeople("actor", "remote");
    expect(api).toHaveBeenNthCalledWith(
      1,
      "/people/search?q=actor&phase=local&limit=12",
      { signal: undefined },
    );
    expect(api).toHaveBeenNthCalledWith(
      2,
      "/people/search?q=actor&phase=remote&limit=12",
      { signal: undefined },
    );
  });

  it("keeps local ranking while appending non-duplicate TMDB matches", () => {
    expect(
      mergePeople(
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

  it("deduplicates person identifiers when hydrating URL state", async () => {
    vi.mocked(api).mockResolvedValue({ results: [] });
    await resolvePeople([7, 7, -1, 9]);
    expect(api).toHaveBeenCalledWith("/people/search?ids=7,9", {
      signal: undefined,
    });
  });
});
