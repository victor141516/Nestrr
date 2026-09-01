import { describe, expect, it } from "vitest";
import { mergeKeywordSuggestions } from "./keywordCatalog";

describe("keyword catalog", () => {
  it("keeps prefix order and deduplicates later internal matches", () => {
    expect(
      mergeKeywordSuggestions(
        [
          { id: 9715, name: "superhero" },
          { id: 155030, name: "superhero team" },
        ],
        [
          { id: 9715, name: "superhero" },
          { id: 180734, name: "masked superhero" },
        ],
      ),
    ).toEqual([
      { id: 9715, name: "superhero" },
      { id: 155030, name: "superhero team" },
      { id: 180734, name: "masked superhero" },
    ]);
  });
});
