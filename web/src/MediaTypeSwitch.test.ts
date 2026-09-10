// @vitest-environment jsdom
import { fireEvent, render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import MediaTypeSwitch from "./MediaTypeSwitch.vue";

describe("MediaTypeSwitch", () => {
  it("exposes only the shared movie and show options", async () => {
    const view = render(MediaTypeSwitch, {
      props: {
        modelValue: "movie",
        label: "contentType",
        translate: (key: string) => key,
      },
      global: { stubs: { AppIcon: true } },
    });

    expect(view.getAllByRole("button")).toHaveLength(2);
    await fireEvent.click(view.getByRole("button", { name: "shows" }));
    expect(view.emitted().change).toEqual([["tv"]]);
    expect(view.emitted()["update:modelValue"]).toEqual([["tv"]]);
  });
});
