// @vitest-environment jsdom
import { fireEvent, render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import AppSelect from "./AppSelect.vue";

const options = [
  { value: "popularity", label: "Popularity" },
  { value: "release", label: "Release date" },
  { value: "rating", label: "Rating" },
] as const;

describe("AppSelect", () => {
  it("selects an option and reports the model change", async () => {
    const view = render(AppSelect, {
      props: { modelValue: "popularity", label: "Sort by", options },
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.click(
      view.getByRole("button", { name: "Sort by: Popularity" }),
    );
    await fireEvent.click(
      view.getByRole("menuitemradio", { name: "Release date" }),
    );

    expect(view.emitted()["update:modelValue"]).toEqual([["release"]]);
    expect(view.emitted().change).toEqual([["release"]]);
    expect(view.queryByRole("menu")).toBeNull();
  });

  it("keeps the option available until a pointer click finishes", async () => {
    const view = render(AppSelect, {
      props: { modelValue: "popularity", label: "Sort by", options },
      global: { stubs: { AppIcon: true } },
    });

    await fireEvent.click(
      view.getByRole("button", { name: "Sort by: Popularity" }),
    );
    const selected = view.getByRole("menuitemradio", { name: "Popularity" });
    const release = view.getByRole("menuitemradio", { name: "Release date" });
    await fireEvent.focusOut(selected, { relatedTarget: null });
    await fireEvent.click(release);

    expect(view.emitted()["update:modelValue"]).toEqual([["release"]]);
  });

  it("supports arrow navigation and returns focus on Escape", async () => {
    const view = render(AppSelect, {
      props: { modelValue: "popularity", label: "Sort by", options },
      global: { stubs: { AppIcon: true } },
    });
    const trigger = view.getByRole("button", { name: "Sort by: Popularity" });

    await fireEvent.keyDown(trigger, { key: "ArrowDown" });
    const first = view.getByRole("menuitemradio", { name: "Popularity" });
    const second = view.getByRole("menuitemradio", { name: "Release date" });
    expect(document.activeElement).toBe(first);
    await fireEvent.keyDown(first, { key: "ArrowDown" });
    expect(document.activeElement).toBe(second);
    await fireEvent.keyDown(second, { key: "Escape" });
    expect(document.activeElement).toBe(trigger);
  });
});
