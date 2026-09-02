import { cleanup, fireEvent, render } from "@testing-library/vue";
import { afterEach, describe, expect, it } from "vitest";
import AppShellHeader from "./AppShellHeader.vue";

afterEach(cleanup);

describe("AppShellHeader", () => {
  it("owns menu visibility and emits account actions", async () => {
    const view = render(AppShellHeader, {
      props: {
        user: {
          id: 1,
          username: "admin",
          isAdmin: true,
          language: "en",
          accentColor: "#000000",
        },
        impersonating: false,
        path: "/",
        translate: (key) => key,
      },
      global: {
        stubs: { RouterLink: { template: "<a><slot /></a>" } },
      },
    });

    await fireEvent.click(view.getByRole("button", { name: "A" }));
    await fireEvent.click(view.getByRole("menuitem", { name: "logout" }));

    expect(view.emitted().logout).toHaveLength(1);
    expect(view.queryByRole("menuitem", { name: "logout" })).toBeNull();
    expect(view.queryByText("library")).toBeNull();
  });

  it("closes the account menu with Escape and when the route changes", async () => {
    const props = {
      user: {
        id: 1,
        username: "admin",
        isAdmin: true,
        language: "en",
        accentColor: "#000000",
      },
      impersonating: false,
      path: "/",
      translate: (key: string) => key,
    };
    const view = render(AppShellHeader, {
      props,
      global: {
        stubs: { RouterLink: { template: "<a><slot /></a>" } },
      },
    });
    const avatar = view.getByRole("button", { name: "A" });

    await fireEvent.click(avatar);
    await fireEvent.keyDown(view.getByRole("menu"), { key: "Escape" });
    expect(view.queryByRole("menu")).toBeNull();
    expect(document.activeElement).toBe(avatar);

    await fireEvent.click(avatar);
    await view.rerender({ ...props, path: "/settings" });
    expect(view.queryByRole("menu")).toBeNull();
  });

  it("keeps the menu mounted while focus moves between its actions", async () => {
    const view = render(AppShellHeader, {
      props: {
        user: {
          id: 1,
          username: "admin",
          isAdmin: true,
          language: "en",
          accentColor: "#000000",
        },
        impersonating: false,
        path: "/recommendations",
        translate: (key: string) => key,
      },
      global: {
        stubs: { RouterLink: { template: "<a><slot /></a>" } },
      },
    });

    await fireEvent.click(view.getByRole("button", { name: "A" }));
    const settings = view.getByRole("menuitem", { name: "settings" });
    const admin = view.getByRole("menuitem", { name: "admin" });

    await fireEvent.focusOut(settings, { relatedTarget: admin });

    expect(view.getByRole("menu")).toBeTruthy();
    expect(view.getByRole("menuitem", { name: "admin" })).toBeTruthy();
  });
});
