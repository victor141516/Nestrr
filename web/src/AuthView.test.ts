import { fireEvent, render } from "@testing-library/vue";
import { describe, expect, it } from "vitest";
import AuthView from "./AuthView.vue";

describe("AuthView", () => {
  it("emits the selected authentication mode with the form event", async () => {
    const view = render(AuthView, {
      props: {
        needsBootstrap: true,
        busy: false,
        error: "",
        translate: (key) => key,
      },
    });

    await fireEvent.update(view.getByLabelText("username"), "admin");
    await fireEvent.update(view.getByLabelText("password"), "password-123");
    await fireEvent.submit(view.container.querySelector("form")!);

    const emission = view.emitted("authenticate")[0] as unknown as [
      string,
      Event,
    ];
    expect(emission[0]).toBe("bootstrap");
    expect(emission[1]).toBeInstanceOf(Event);
  });
});
