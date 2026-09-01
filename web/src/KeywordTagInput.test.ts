// @vitest-environment jsdom
import { fireEvent, render } from "@testing-library/vue";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import KeywordTagInput from "./KeywordTagInput.vue";
import { resolveKeywords, searchKeywords } from "./keywordCatalog";

vi.mock("./keywordCatalog", () => ({
  resolveKeywords: vi.fn(),
  searchKeywords: vi.fn(),
}));

describe("KeywordTagInput", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(resolveKeywords).mockResolvedValue([]);
    vi.mocked(searchKeywords).mockReset();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("debounces input and reveals prefix results before deduplicated internal matches", async () => {
    let finishContains!: (value: { id: number; name: string }[]) => void;
    vi.mocked(searchKeywords).mockImplementation((_query, phase) => {
      if (phase === "prefix")
        return Promise.resolve([
          { id: 9715, name: "superhero" },
          { id: 155030, name: "superhero team" },
        ]);
      return new Promise((resolve) => {
        finishContains = resolve;
      });
    });

    const view = render(KeywordTagInput, {
      props: {
        id: "keywords",
        label: "Keywords",
        modelValue: [],
        placeholder: "Search keywords",
      },
    });
    const editor = view.container.querySelector<HTMLElement>(".tagify__input")!;
    editor.focus();
    await fireEvent.focus(editor);
    editor.textContent = "super";
    await fireEvent.input(editor);

    expect(searchKeywords).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(180);
    await vi.advanceTimersByTimeAsync(0);
    expect(searchKeywords).toHaveBeenNthCalledWith(
      1,
      "super",
      "prefix",
      expect.any(AbortSignal),
    );
    finishContains([
      { id: 9715, name: "superhero" },
      { id: 180734, name: "masked superhero" },
    ]);
    await vi.advanceTimersByTimeAsync(0);
    expect(searchKeywords).toHaveBeenNthCalledWith(
      2,
      "super",
      "contains",
      expect.any(AbortSignal),
    );
  });
});
